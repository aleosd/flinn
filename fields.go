package flinn

import (
	"strconv"
	"strings"
)

type fieldKind int

const (
	kindLeaf fieldKind = iota
	kindGroup
)

// FieldOption is a function type used to configure a Field.
type FieldOption func(*Field)

type parser[T any] func(raw string) (T, error)

// Env is a Field option to set an environment variable name to load a value from.
func Env(key string) FieldOption {
	return func(f *Field) {
		f.envKey = key
	}
}

// FileKey is a name of configuration option in a file to load value from.
func FileKey(key string) FieldOption {
	return func(f *Field) {
		f.fileKey = key
	}
}

// Required is a Field option that marks this field as required.
// Loader will fail - return an error during loading - if this field is not set.
func Required() FieldOption {
	return func(f *Field) {
		f.required = true
	}
}

// Default is a Field option to set a default value for a field.
// This value will be used if other sources (env, file) do not provide a value.
func Default(val any) FieldOption {
	return func(f *Field) {
		f.hasDefault = true
		f.defaultVal = val
	}
}

// Field is a struct to configure a single configuration key.
type Field struct {
	kind     fieldKind
	name     string
	assign   func(raw string) error
	dest     any
	children []Field

	envKey       string
	envPrefix    string
	fileKey      string
	required     bool
	hasDefault   bool
	defaultVal   any
	applyDefault func()
	oneOf        []string
	validators   []func(any) error
}

func (f *Field) getPathSegment() string {
	if f.fileKey != "" {
		return f.fileKey
	}
	return toSnakeCase(f.name)
}

// makeField is the shared constructor logic for any leaf type.
func makeField[T any](name string, dest *T, parse parser[T], opts []FieldOption) Field {
	f := Field{
		kind: kindLeaf,
		name: name,
		assign: func(raw string) error {
			v, err := parse(raw)
			if err != nil {
				return err
			}
			*dest = v
			return nil
		},
	}
	for _, o := range opts {
		o(&f)
	}
	if f.hasDefault {
		if typed, ok := f.defaultVal.(T); ok {
			f.applyDefault = func() { *dest = typed }
		}
	}
	if f.fileKey == "" {
		f.fileKey = toSnakeCase(name)
	}
	return f
}

// String is a constructor for a configuration field with value of type string.
func String(name string, dest *string, opts ...FieldOption) Field {
	assigner := func(raw string) (string, error) {
		return raw, nil
	}
	return makeField(name, dest, assigner, opts)
}

// Int is a constructor for a configuration field with value of type int.
func Int(name string, dest *int, opts ...FieldOption) Field {
	assigner := func(raw string) (int, error) {
		i, err := strconv.Atoi(raw)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	return makeField(name, dest, assigner, opts)
}

// Group wraps a set of children fields under a named scope.
// Used to represent a nested configuration structures - maps.
// Options on a group apply to the group itself (prefix, file key).
func Group(name string, children []Field, opts ...FieldOption) Field {
	var f = Field{kind: kindGroup, name: name, children: children}
	for _, o := range opts {
		o(&f)
	}
	return f
}

// converts a string to snake_case
// simplified version from https://github.com/iancoleman/strcase/blob/master/snake.go
func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s) + 2)
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'

		if vIsCap {
			v += 'a' - 'A' // normalize to lowercase
		}

		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'

			if (vIsCap && (nextIsLow || nextIsNum)) || (vIsLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				if vIsCap && nextIsLow {
					if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
						n.WriteByte('_')
					}
				}
				n.WriteByte(v)
				if vIsLow || vIsNum || nextIsNum {
					n.WriteByte('_')
				}
				continue
			}
		}

		if v == ' ' || v == '_' || v == '-' || v == '.' {
			n.WriteByte('_')
		} else {
			n.WriteByte(v)
		}
	}

	return n.String()
}

func stringify(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true

	case int:
		return strconv.Itoa(val), true

	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), true

	case bool:
		return strconv.FormatBool(val), true

	case nil:
		return "", false // null treat as absent

	default:
		// []any or map[string]any — not a leaf value
		return "", false
	}
}
