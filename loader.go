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
type envKeyFunc func(f Field, prefix string) string

// explicitEnvKey is the default: only use a key if Env() was explicitly set.
func explicitEnvKey(f Field, prefix string) string {
	return joinEnvPrefix(prefix, f.envKey)
}

// autoEnvKey falls back to the uppercased snake_case field name.
func autoEnvKey(f Field, prefix string) string {
	key := f.envKey
	if key == "" {
		key = strings.ToUpper(f.getPathSegment())
	}
	return joinEnvPrefix(prefix, key)
}

// A Source is an interface that must be implemented by any other struct
// in order to be used as a source for configuration values.
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

// Load populates configuration struct, based on fields configuration provided
// as an input array of Field objects. Each field is loaded sequentially,
// environment variables take precedence over other sources.
// Error of type FieldErrors is returned in case of any errors.
func (l *Loader) Load(fields []Field) error {
	var errs FieldErrors
	l.walk(fields, []string{}, l.envPrefix, &errs)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// LoaderOption is a function type used to configure a Loader.
type LoaderOption func(*Loader)

// WithSource is a loader option that sets the source to use for loading configuration.
// It accepts only objects with Source interface.
func WithSource(source Source) LoaderOption {
	return func(l *Loader) {
		l.source = source
	}
}

// WithEnvPrefix is a loader option that sets the prefix to use for environment variables.
func WithEnvPrefix(envPrefix string) LoaderOption {
	return func(l *Loader) {
		l.envPrefix = envPrefix
	}
}

// WithLogger is a loader option that sets the logger to use for logging.
func WithLogger(logger *slog.Logger) LoaderOption {
	return func(l *Loader) {
		l.log = logger
	}
}

// WithAutoEnv is a loader option that enables automatic load of configuration from environment.
// Variable names can be set per field using `Env()` or will be derived from the field name.
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

func (l *Loader) walk(fields []Field, pathSegments []string, envPrefix string, errs *FieldErrors) {
	for _, f := range fields {
		l.log.Debug("walking field", "field", f.name, "kind", f.kind)
		if f.kind == kindGroup {
			// Groups don't hold a value themselves.
			// They contribute a path segment and optionally an env prefix.
			childEnvPrefix := joinEnvPrefix(envPrefix, f.envKey)
			childPathSegments := append(pathSegments, f.getPathSegment())
			l.walk(f.children, childPathSegments, childEnvPrefix, errs)
			continue
		}

		// Leaf field: resolve, coerce, validate.
		envKey := l.envKeyFunc(f, envPrefix)
		keyPath := append(pathSegments, f.getPathSegment())
		logicalPath := strings.Join(keyPath, ".")
		rawVal, found, err := l.resolve(keyPath, envKey)
		if err != nil {
			errs.add(logicalPath, "resolve", nil, err.Error())
			continue
		}
		if !found {
			if f.hasDefault {
				if f.applyDefault != nil {
					l.log.Debug("applying default value", "field", f.name, "value", f.defaultVal)
					f.applyDefault()
				} else {
					l.log.Warn("default value is set, but can not be applied", "path", logicalPath, "value", f.defaultVal)
					errs.add(logicalPath, "default", nil, "bad default value (type mistmatch?)")
				}
				continue
			}
			if f.required {
				errs.add(logicalPath, "required", nil, "value is required but was not provided")
				l.log.Warn("required value is missing", "path", logicalPath)
			}
			continue
		}

		if err := f.assign(rawVal); err != nil {
			errs.add(logicalPath, "parse", rawVal, err.Error())
			continue
		}

		// Run validation rules against the now-typed value.
		l.validate(logicalPath, f.dest, f, errs)
	}
}

func (l *Loader) validate(path string, val any, f Field, errs *FieldErrors) {
	for _, v := range f.validators {
		if err := v(val); err != nil {
			errs.add(path, "validate", val, err.Error())
			l.log.Warn("validation failed", path, err.Error())
		}
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
