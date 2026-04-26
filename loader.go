// Package flinn provides a declarative configuration loader for Go structs.
// It supports loading values from environment variables, file-based sources (like YAML/JSON),
// and validating them with rules.
//
// # Core Concepts
//
//   - Loader: Orchestrates loading from multiple sources.
//   - Field: Definition for a single configuration value, created by `String()`, `Int()`, etc.
//   - Source: An interface for providing values from a structured source (e.g., a YAML file).
//
// Values are resolved in the following order of precedence:
//
//  1. Environment variable (if an enabled for field or loader)
//  2. Source (e.g., configuration file)
//  3. Default value (if set)
//
// If a required field has no value, loading fails with a FieldErrors collection.
package flinn

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// envKeyFunc derives the effective environment variable key for a field.
// prefix is the accumulated env prefix at the current walk depth.
type envKeyFunc func(f ConfigItem, prefix string) string

// explicitEnvKey is the default: only use a key if Env() was explicitly set.
func explicitEnvKey(f ConfigItem, prefix string) string {
	return joinEnvPrefix(prefix, f.envKey())
}

// autoEnvKey falls back to the uppercased snake_case field name.
func autoEnvKey(f ConfigItem, prefix string) string {
	key := f.envKey()
	if key == "" {
		key = strings.ToUpper(f.getPathSegment())
	}
	return joinEnvPrefix(prefix, key)
}

// Source is the interface that wraps the Get method for retrieving configuration values.
// Any configuration provider (JSON, YAML, etc.) must implement this interface.
type Source interface {
	// Get retrieves a configuration value from the source at the specified path.
	// It returns the raw string value, a boolean indicating if the value was found,
	// and an error if the retrieval failed.
	Get(path []string) (string, bool, error)
}

// Loader is responsible for base configuration and configuration loading.
type Loader struct {
	source     Source
	envPrefix  string
	log        *slog.Logger
	envKeyFunc envKeyFunc
}

// Load populates the configuration based on the provided fields.
// Each field is resolved sequentially, with environment variables taking precedence
// over other sources. It returns a FieldErrors collection if any errors occur.
func (l *Loader) Load(fields []ConfigItem) error {
	var errs FieldErrors
	l.walk(fields, []string{}, l.envPrefix, &errs)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// LoaderOption is a function type used to configure a Loader.
type LoaderOption func(*Loader)

// WithSource sets the configuration source (e.g., JSONSource) for the loader.
func WithSource(source Source) LoaderOption {
	return func(l *Loader) {
		l.source = source
	}
}

// WithEnvPrefix sets a global prefix for all environment variables.
func WithEnvPrefix(envPrefix string) LoaderOption {
	return func(l *Loader) {
		l.envPrefix = envPrefix
	}
}

// WithLogger sets the logger used by the loader for debugging and warnings.
func WithLogger(logger *slog.Logger) LoaderOption {
	return func(l *Loader) {
		l.log = logger
	}
}

// WithAutoEnv enables automatic resolution of environment variables based on field names.
// If Env() is not explicitly called on a field, the environment variable name
// will be derived from the field's path (e.g., "DATABASE_PORT").
func WithAutoEnv() LoaderOption {
	return func(l *Loader) {
		l.envKeyFunc = autoEnvKey
	}
}

// NewLoader returns a new Loader instance.
func NewLoader(opts ...LoaderOption) *Loader {
	l := &Loader{
		log:        discardLogger,
		envKeyFunc: explicitEnvKey,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Loader) walk(fields []ConfigItem, pathSegments []string, envPrefix string, errs *FieldErrors) {
	for _, f := range fields {
		l.log.Debug("walking field", "field", f.fieldName(), "kind", f.fieldKind())
		if f.fieldKind() == kindGroup {
			// Groups don't hold a value themselves.
			// They contribute a path segment and optionally an env prefix.
			f := f.(*Group)
			childEnvPrefix := joinEnvPrefix(envPrefix, f.comm.envKey)
			childPathSegments := append(pathSegments, f.getPathSegment())
			l.walk(f.childrenNodes(), childPathSegments, childEnvPrefix, errs)
			continue
		}

		// Leaf field: resolve, coerce, validate.
		f := f.(leafField)
		envKey := l.envKeyFunc(f, envPrefix)
		keyPath := append(pathSegments, f.getPathSegment())
		logicalPath := strings.Join(keyPath, ".")
		rawVal, found, err := l.resolve(keyPath, envKey)
		if err != nil {
			errs.add(logicalPath, "resolve", nil, err.Error())
			continue
		}
		if !found {
			if f.hasDefaultVal() {
				l.log.Debug("applying default value", "field", f.fieldName())
				f.applyDefault()
				continue
			}
			if f.isRequired() {
				errs.add(logicalPath, "required", nil, "value is required but was not provided")
				l.log.Warn("required value is missing", "path", logicalPath)
			}
			continue
		}

		if err := f.assignRaw(rawVal); err != nil {
			errs.add(logicalPath, "parse", rawVal, err.Error())
			continue
		}

		// Run validation rules against the now-typed value.
		f.runValidators(logicalPath, errs, l.log)
	}
}

// resolve tries each source in order, returning the first hit.
// The env key used is: envPrefix + "_" + def.envKey (if both are set).
func (l *Loader) resolve(pathSegments []string, envKey string) (string, bool, error) {
	// Try env variable first
	if envKey != "" {
		if envValue, ok := os.LookupEnv(envKey); ok {
			l.log.Debug("resolved value from env", envKey, pathSegments)
			return envValue, true, nil
		}
	}

	if l.source == nil {
		l.log.Warn("no source configured")
		return "", false, nil
	}
	v, found, err := l.source.Get(pathSegments)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}
	l.log.Debug("resolved value from source", "", pathSegments)
	return v, true, nil
}

func joinEnvPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "_" + key
}

// DefineSchema is a helper that groups multiple configuration items into a slice.
func DefineSchema(fields ...ConfigItem) []ConfigItem { return fields }
