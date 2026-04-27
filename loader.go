// Package flinn provides a declarative, type-safe configuration loader for Go.
// It resolves values from environment variables, JSON files, and custom sources,
// with support for defaults, required fields, and validation.
//
// # Core Concepts
//
//   - Loader: Orchestrates loading from multiple sources.
//   - Field: Definition for a single configuration value, created by [String], [Int], etc.
//   - Source: An interface for providing values from a structured source (e.g., a JSON file).
//   - Group: A collection of fields that share a namespace for file paths and env prefixes.
//
// Values are resolved in the following order of precedence (highest to lowest):
//
//  1. Environment variable (if enabled for the field or loader)
//  2. Config source (e.g., JSON or TOML file)
//  3. Default value (if set)
//  4. Required error (if the field is required and nothing resolved)
//
// If one or more fields fail to resolve or validate, loading returns a [FieldErrors]
// collection so every problem is reported at once.
package flinn

import (
	"fmt"
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
	return joinEnvPrefix(prefix, f.envSegment())
}

// autoEnvKey falls back to the uppercased snake_case field name.
func autoEnvKey(f ConfigItem, prefix string) string {
	key := f.envSegment()
	if key == "" {
		key = strings.ToUpper(f.getPathSegment())
	}
	return joinEnvPrefix(prefix, key)
}

// Source is the interface for configuration backends.
// Implementations provide values from structured sources such as JSON or TOML files.
type Source interface {
	// Get retrieves a configuration value at the given path.
	// path is a sequence of key segments corresponding to nested positions
	// (e.g., ["database", "host"]).
	// Returns the raw string value, true when found, or an error on retrieval failure.
	// When the key is absent, return ("", false, nil).
	Get(path []string) (string, bool, error)
}

// Loader resolves configuration values from multiple sources.
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

// WithEnvPrefix sets a global prefix for auto-generated environment variable names.
// Explicit env names set via [Field.Env] are not prefixed.
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
			g, ok := f.(*Group)
			if !ok {
				path := strings.Join(append(pathSegments, f.getPathSegment()), ".")
				errs.add(path, "type", nil,
					fmt.Sprintf("expected *Group for field kind %d, got %T", kindGroup, f))
				continue
			}
			childEnvPrefix := joinEnvPrefix(envPrefix, g.comm.envSegment)
			childPathSegments := append(pathSegments, g.getPathSegment())
			l.walk(g.childrenNodes(), childPathSegments, childEnvPrefix, errs)
			continue
		}

		// Leaf field: resolve, coerce, validate.
		lf, ok := f.(leafField)
		if !ok {
			path := strings.Join(append(pathSegments, f.getPathSegment()), ".")
			errs.add(path, "type", nil,
				fmt.Sprintf("expected leafField for field kind %d, got %T", f.fieldKind(), f))
			continue
		}
		envKey := l.envKeyFunc(lf, envPrefix)
		keyPath := append(pathSegments, lf.getPathSegment())
		logicalPath := strings.Join(keyPath, ".")
		rawVal, found, err := l.resolve(keyPath, envKey)
		if err != nil {
			errs.add(logicalPath, "resolve", nil, err.Error())
			continue
		}
		if !found {
			if lf.hasDefaultVal() {
				l.log.Debug("applying default value", "field", lf.fieldName())
				lf.applyDefault()
				continue
			}
			if lf.isRequired() {
				errs.add(logicalPath, "required", nil, "value is required but was not provided")
				l.log.Warn("required value is missing", "path", logicalPath)
			}
			continue
		}

		if err := lf.assignRaw(rawVal); err != nil {
			errs.add(logicalPath, "parse", rawVal, err.Error())
			continue
		}

		// Run validation rules against the now-typed value.
		lf.runValidators(logicalPath, errs, l.log)
	}
}

// resolve tries each source in order, returning the first hit.
// It first checks the environment variable keyed by envKey, then falls back
// to the registered config source at pathSegments.
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
	l.log.Debug("resolved value from source", "path", pathSegments)
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

// DefineSchema groups configuration items into a slice for [Loader.Load].
// It is a convenience helper with no runtime effect.
func DefineSchema(fields ...ConfigItem) []ConfigItem { return fields }
