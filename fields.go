package flinn

import (
	"cmp"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type fieldKind int

const (
	kindLeaf fieldKind = iota
	kindGroup
)

// ConfigItem represents a general item loaded from a configuration file.
// It can be either a leaf value (Field) or a collection of items (Group).
type ConfigItem interface {
	fieldName() string
	envKey() string
	fieldKind() fieldKind
	applyDefault()
	childrenNodes() []ConfigItem
	getPathSegment() string
}

type commonMembers struct {
	name     string
	envKey   string
	fileKey  string
	required bool
}

type leafMembers[T any] struct {
	hasDefault bool
	defaultVal T
	validators []func(T) error
}

// Group represents a collection of configuration items under a named scope.
// It can be used to model nested structures in configuration files.
type Group struct {
	comm     *commonMembers
	children []ConfigItem
}

func (g *Group) envKey() string {
	return g.comm.envKey
}

func (g *Group) fieldName() string {
	return g.comm.name
}

func (g *Group) fieldKind() fieldKind {
	return kindGroup
}

func (g *Group) applyDefault() {}

func (g *Group) childrenNodes() []ConfigItem {
	return g.children
}
func (g *Group) getPathSegment() string {
	if g.comm.fileKey != "" {
		return g.comm.fileKey
	}
	return toSnakeCase(g.comm.name)
}

type parser[T any] func(raw string) (T, error)

// Field represents a single configuration leaf node that parses values into type T.
// It holds configuration for environment variable keys, file keys, default values,
// and validation rules.
type Field[T any] struct {
	comm     *commonMembers
	field    *leafMembers[T]
	assign   func(raw string) error
	dest     *T
	children []ConfigItem

	envPrefix string
	required  bool
	oneOf     []string
}

// leafField is the interface walk uses to interact with any leaf,
// regardless of its concrete type parameter.
type leafField interface {
	ConfigItem
	isRequired() bool
	hasDefaultVal() bool
	assignRaw(raw string) error
	runValidators(path string, errs *FieldErrors, log *slog.Logger)
}

func (f *Field[T]) isRequired() bool           { return f.required }
func (f *Field[T]) hasDefaultVal() bool        { return f.field.hasDefault }
func (f *Field[T]) assignRaw(raw string) error { return f.assign(raw) }

func (f *Field[T]) runValidators(path string, errs *FieldErrors, log *slog.Logger) {
	for _, v := range f.field.validators {
		if err := v(*f.dest); err != nil {
			errs.add(path, "validate", *f.dest, err.Error())
			log.Warn("validation failed", "path", path, "error", err.Error())
		}
	}
}

func (f *Field[T]) envKey() string {
	return f.comm.envKey
}

func (f *Field[T]) fieldName() string {
	return f.comm.name
}

func (f *Field[T]) fieldKind() fieldKind {
	return kindLeaf
}

func (f *Field[T]) childrenNodes() []ConfigItem {
	return nil
}

func (f *Field[T]) getPathSegment() string {
	if f.comm.fileKey != "" {
		return f.comm.fileKey
	}
	return toSnakeCase(f.comm.name)
}

func (f *Field[T]) applyDefault() {
	if f.field.hasDefault {
		*f.dest = f.field.defaultVal
	}
}

// Env is a Field option to set an environment variable name to load a value from.
func (f *Field[T]) Env(key string) *Field[T] {
	f.comm.envKey = key
	return f
}

// FileKey is a name of configuration option in a file to load value from.
func (f *Field[T]) FileKey(key string) *Field[T] {
	f.comm.fileKey = key
	return f
}

// Required is a Field option that marks this field as required.
// Loader will return an error during loading if this field is not set.
func (f *Field[T]) Required() *Field[T] {
	f.required = true
	return f
}

// Default is a Field option to set a default value for a field.
// This value will be used if other sources (env, file) do not provide a value.
func (f *Field[T]) Default(val T) *Field[T] {
	f.field.hasDefault = true
	f.field.defaultVal = val
	return f
}

// AddValidator adds a custom validation function to the field.
// Validators are run after the value is parsed and assigned to the destination.
func (f *Field[T]) AddValidator(fn func(T) error) *Field[T] {
	f.field.validators = append(f.field.validators, fn)
	return f
}

// NumericField is a specialized Field for ordered types (integers, floats)
// that supports additional range-based validators like Min and Max.
type NumericField[T cmp.Ordered] struct {
	*Field[T]
}

// Min adds a validator that ensures the field value is greater than or equal to v.
func (f *NumericField[T]) Min(v T) *NumericField[T] {
	f.Field.AddValidator(func(val T) error {
		if val < v {
			return fmt.Errorf("must be >= %v", v)
		}
		return nil
	})
	return f
}

// Max adds a validator that ensures the field value is less than or equal to v.
func (f *NumericField[T]) Max(v T) *NumericField[T] {
	f.Field.AddValidator(func(val T) error {
		if val > v {
			return fmt.Errorf("must be <= %v", v)
		}
		return nil
	})
	return f
}

// Env sets an explicit environment variable name for this numeric field.
func (f *NumericField[T]) Env(key string) *NumericField[T] { f.comm.envKey = key; return f }

// FileKey sets an explicit configuration file key for this numeric field.
func (f *NumericField[T]) FileKey(key string) *NumericField[T] { f.comm.fileKey = key; return f }

// Required marks this numeric field as required.
func (f *NumericField[T]) Required() *NumericField[T] { f.comm.required = true; return f }

// Default sets a default value for this numeric field.
func (f *NumericField[T]) Default(v T) *NumericField[T] { f.Field.Default(v); return f }

// AddValidator adds a custom validation function to this numeric field.
func (f *NumericField[T]) AddValidator(fn func(T) error) *NumericField[T] {
	f.Field.AddValidator(fn)
	return f
}

// makeField is the shared constructor logic for any leaf type.
func makeField[T any](name string, dest *T, parse parser[T]) *Field[T] {
	comm := &commonMembers{name: name}
	field := &leafMembers[T]{}
	f := Field[T]{
		comm:  comm,
		field: field,
		dest:  dest,
		assign: func(raw string) error {
			v, err := parse(raw)
			if err != nil {
				return err
			}
			*dest = v
			return nil
		},
	}

	if f.comm.fileKey == "" {
		f.comm.fileKey = toSnakeCase(name)
	}
	return &f
}

// String creates a configuration leaf Field that handles string values.
func String(name string, dest *string) *Field[string] {
	assigner := func(raw string) (string, error) {
		return raw, nil
	}
	return makeField(name, dest, assigner)
}

// Int creates a configuration leaf Field that parses string values as base-10 integers.
// It returns a NumericField, allowing for range-based validation (Min, Max).
func Int(name string, dest *int) *NumericField[int] {
	assigner := func(raw string) (int, error) {
		i, err := strconv.Atoi(raw)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	return &NumericField[int]{Field: makeField(name, dest, assigner)}
}

// Bool creates a configuration Field that parses a boolean value from a raw string and writes it to dest.
// The value is converted to lower case before parsing, and strconv.ParseBool is used.
func Bool(name string, dest *bool) *Field[bool] {
	assigner := func(raw string) (bool, error) {
		b, err := strconv.ParseBool(strings.ToLower(raw))
		if err != nil {
			return false, err
		}
		return b, nil
	}
	return makeField(name, dest, assigner)
}

// Float creates a configuration Field that parses a floating-point value.
// It returns a NumericField, allowing for range-based validation (Min, Max).
func Float(name string, dest *float64) *NumericField[float64] {
	assigner := func(raw string) (float64, error) {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	}
	return &NumericField[float64]{Field: makeField(name, dest, assigner)}
}

// UUID constructs a Field representing a configuration leaf whose value is parsed as a UUID and stored in dest.
func UUID(name string, dest *uuid.UUID) *Field[uuid.UUID] {
	assigner := func(raw string) (uuid.UUID, error) {
		return uuid.Parse(raw)
	}
	return makeField(name, dest, assigner)
}

// FieldsGroup creates a new Group that wraps multiple configuration items under a named scope.
// It is used to represent nested configuration structures.
func FieldsGroup(name string, children ...ConfigItem) *Group {
	comm := &commonMembers{name: name}
	var f = Group{comm: comm, children: children}
	return &f
}

// EnvPrefix sets the environment variable prefix for all children of this group.
func (g *Group) EnvPrefix(prefix string) *Group {
	g.comm.envKey = prefix
	return g
}

// FileKey sets the configuration file key for this group.
func (g *Group) FileKey(key string) *Group {
	g.comm.fileKey = key
	return g
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
