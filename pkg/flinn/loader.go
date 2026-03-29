package flinn

type Source interface {
	Get(key string) (string, error)
}

type Loader struct {
	source    *Source
	envPrefix string
}

func (l *Loader) Load(fields []Field) error {
	var errs FieldErrors
	l.walk(fields, "", "", &errs)
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

func (l *Loader) walk(fields []Field, pathPrefix, envPrefix string, errs *FieldErrors) {
	for _, f := range fields {
		def := buildDef(f.opts)
		logicalPath := joinPath(pathPrefix, f.name)

		if f.kind == kindGroup {
			// Groups don't hold a value themselves.
			// They contribute a path segment and optionally an env prefix.
			childEnvPrefix := joinEnvPrefix(envPrefix, def.envPrefix)
			l.walk(f.children, logicalPath, childEnvPrefix, errs)
			continue
		}

		// Leaf field: resolve, coerce, validate.
		rawVal, found := l.resolve(def, envPrefix)

		if !found {
			if def.hasDefault {
				f.set(def.defaultVal)
				continue
			}
			if def.required {
				errs.add(logicalPath, "required", nil, "value is required but was not provided")
			}
			continue
		}

		if err := f.assign(rawVal); err != nil {
			errs.add(logicalPath, "parse", rawVal, err.Error())
			continue
		}

		// Run validation rules against the now-typed value.
		l.validate(logicalPath, f.dest, def, errs)
	}
}

func (l *Loader) validate(path string, val any, def fieldDef, errs *FieldErrors) {
	for _, v := range def.validators {
		if err := v(val); err != nil {
			errs.add(path, "validate", val, err.Error())
		}
	}
}

// resolve tries each source in order, returning the first hit.
// The env key used is: envPrefix + "_" + def.envKey (if both are set).
func (l *Loader) resolve(def fieldDef, envPrefix string) (string, bool) {
	// Try env variable first
	key := joinEnvPrefix(envPrefix, def.envKey)

	if l.source == nil {
		return "", false
	}
	if v, err := (*l.source).Get(key); err == nil {
		return v, true
	}
	return "", false
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
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
