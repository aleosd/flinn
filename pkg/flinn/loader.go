// Package flinn populates sutruct values from environment variables and different source formats.
//
// It is designed to load and validate configuration.
package flinn

import (
	"os"
	"strings"
)

// A Source is an interface that must be impelmented by any other struct
// in order to be used as a source for configuration values.
type Source interface {
	Get(path []string) (string, bool, error)
}

// Loader is responsible for base configutation and configuation loading.
type Loader struct {
	source    *Source
	envPrefix string
}

// Load populates configuration struct, based on fields configuration provided
// as an input array of Field objects. Each field is loaded sequentially,
// environment variables are take prcedence over other sources.
// Error of tyoe FieldErrors is returned in case of any errors.
func (l *Loader) Load(fields []Field) error {
	var errs FieldErrors
	l.walk(fields, []string{}, l.envPrefix, &errs)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

type loaderOption func(*Loader)

// WithSource is a loader oprtion that sets the source to use for loading configuration.
// It accepts only objects with Source interface.
func WithSource(source Source) loaderOption {
	return func(l *Loader) {
		l.source = &source
	}
}

// WithEnvPrefix is a loader option that sets the prefix to use for environment variables.
func WithEnvPrefix(envPrefix string) loaderOption {
	return func(l *Loader) {
		l.envPrefix = envPrefix
	}
}

// NewLoader returns a new Loader instance.
func NewLoader(opts ...loaderOption) *Loader {
	l := &Loader{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Loader) walk(fields []Field, pathSegments []string, envPrefix string, errs *FieldErrors) {
	for _, f := range fields {
		if f.kind == kindGroup {
			// Groups don't hold a value themselves.
			// They contribute a path segment and optionally an env prefix.
			childEnvPrefix := joinEnvPrefix(envPrefix, f.envKey)
			childPathSegments := append(pathSegments, f.getPathSegment())
			l.walk(f.children, childPathSegments, childEnvPrefix, errs)
			continue
		}

		// Leaf field: resolve, coerce, validate.
		envKey := joinEnvPrefix(envPrefix, f.envKey)
		keyPath := append(pathSegments, f.getPathSegment())
		logicalPath := strings.Join(keyPath, ".")
		rawVal, found, err := l.resolve(keyPath, envKey)
		if err != nil {
			errs.add(logicalPath, "resolve", nil, err.Error())
			continue
		}
		if !found {
			if f.hasDefault {
				f.set(f.defaultVal)
				continue
			}
			if f.required {
				errs.add(logicalPath, "required", nil, "value is required but was not provided")
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
		}
	}
}

// resolve tries each source in order, returning the first hit.
// The env key used is: envPrefix + "_" + def.envKey (if both are set).
func (l *Loader) resolve(pathSegments []string, envKey string) (string, bool, error) {
	// Try env variable first
	if envValue, ok := os.LookupEnv(envKey); ok {
		return envValue, true, nil
	}

	if l.source == nil {
		return "", false, nil
	}
	v, found, err := (*l.source).Get(pathSegments)
	if err != nil {
		return "", false, err
	}
	if found == false {
		return "", false, nil
	}
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
