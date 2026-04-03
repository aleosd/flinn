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

type fieldOption func(*Field)
type parser[T any] func(raw string) (T, error)

func Env(key string) fieldOption {
	return func(f *Field) {
		f.envKey = key
	}
}

func FileKey(key string) fieldOption {
	return func(f *Field) {
		f.fileKey = key
	}
}

func Required() fieldOption {
	return func(f *Field) {
		f.required = true
	}
}

func Default(val any) fieldOption {
	return func(f *Field) {
		f.hasDefault = true
		f.defaultVal = val
	}
}

type Field struct {
	kind     fieldKind
	name     string
	assign   func(raw string) error
	dest     any
	children []Field

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

func (f *Field) set(value any) {
	f.dest = value
}

// makeField is the shared constructor logic for any leaf type.
func makeField[T any](name string, dest *T, parse parser[T], opts []fieldOption) Field {
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
	return f
}

func String(name string, dest *string, opts ...fieldOption) Field {
	assigner := func(raw string) (string, error) {
		return raw, nil
	}
	return makeField(name, dest, assigner, opts)
}

func Int(name string, dest *int, opts ...fieldOption) Field {
	assigner := func(raw string) (int, error) {
		i, err := strconv.Atoi(raw)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	return makeField(name, dest, assigner, opts)
}

// Group wraps a set of children under a named scope.
// Options on a group apply to the group itself (prefix, file key).
func Group(name string, children []Field, opts ...fieldOption) Field {
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
