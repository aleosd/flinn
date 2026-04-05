package flinn

import (
	"os"
	"strings"
)

type Source interface {
	Get(path []string) (string, bool, error)
}

type Loader struct {
	source    *Source
	envPrefix string
}

func (l *Loader) Load(fields []Field) error {
	var errs FieldErrors
	l.walk(fields, []string{}, l.envPrefix, &errs)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

type LoaderOption func(*Loader)

func WithSource(source Source) LoaderOption {
	return func(l *Loader) {
		l.source = &source
	}
}

func WithEnvPrefix(envPrefix string) LoaderOption {
	return func(l *Loader) {
		l.envPrefix = envPrefix
	}
}

func NewLoader(opts ...LoaderOption) *Loader {
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
