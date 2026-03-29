package flinn

import "strconv"

type fieldKind int

const (
	kindLeaf fieldKind = iota
	kindGroup
)

type fieldDef struct {
	envKey     string
	envPrefix  string
	fileKey    string
	required   bool
	hasDefault bool
	defaultVal any
	min, max   *float64
	oneOf      []string
	validators []func(any) error
}

type fieldOption func(*fieldDef)

func Env(key string) fieldOption {
	return func(d *fieldDef) {
		d.envKey = key
	}
}

func FileKey(key string) fieldOption {
	return func(d *fieldDef) {
		d.fileKey = key
	}
}

func Required() fieldOption {
	return func(d *fieldDef) {
		d.required = true
	}
}

func Default(val any) fieldOption {
	return func(d *fieldDef) {
		d.hasDefault = true
		d.defaultVal = val
	}
}

func buildDef(opts []fieldOption) fieldDef {
	var d fieldDef
	for _, o := range opts {
		o(&d)
	}
	return d
}

type Field struct {
	kind     fieldKind
	name     string
	assign   func(raw string) error
	dest     any
	opts     []fieldOption
	children []Field
}

func (f *Field) set(value any) {
	f.dest = value
}

func String(name string, dest *string, opts ...fieldOption) Field {
	assigner := func(raw string) error {
		*dest = raw
		return nil
	}
	return Field{
		kind:     kindLeaf,
		name:     name,
		dest:     dest,
		assign:   assigner,
		opts:     opts,
		children: nil,
	}
}

func Int(name string, dest *int, opts ...fieldOption) Field {
	assigner := func(raw string) error {
		i, err := strconv.Atoi(raw)
		if err != nil {
			return err
		}
		*dest = i
		return nil
	}
	return Field{
		kind:     kindLeaf,
		name:     name,
		dest:     dest,
		opts:     opts,
		assign:   assigner,
		children: nil,
	}
}

// Group wraps a set of children under a named scope.
// Options on a group apply to the group itself (prefix, file key).
func Group(name string, children []Field, opts ...fieldOption) Field {
	return Field{kind: kindGroup, name: name, children: children, opts: opts}
}
