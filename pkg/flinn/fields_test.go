package flinn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{".", "_"},
		{"_", "_"},
		{"foo", "foo"},
		{"Foo", "foo"},
		{"FOO", "foo"},
		{"FOOBar", "foo_bar"},
		{"FooBar", "foo_bar"},
		{"FooBar_baz", "foo_bar_baz"},
		{"Foo_Bar", "foo_bar"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toSnakeCase(tt.input))
		})
	}
}

func TestMakeField(t *testing.T) {
	var value string
	var parser = func(raw string) (string, error) {
		return raw, nil
	}
	t.Run("TestFileKeyWithOption", func(t *testing.T) {
		f := makeField("MyValue", &value, parser, []FieldOption{FileKey("foo_bar")})
		assert.Equal(t, "foo_bar", f.fileKey)
	})
	t.Run("TestWithoutOptions", func(t *testing.T) {
		f := makeField("MyValue", &value, parser, []FieldOption{})
		assert.Equal(t, "my_value", f.fileKey)
		assert.Equal(t, "", f.envPrefix)
		assert.Equal(t, "", f.envKey)
		assert.False(t, f.required)
		assert.False(t, f.hasDefault)
		assert.Nil(t, f.defaultVal)
		assert.Empty(t, f.oneOf)
		assert.Empty(t, f.validators)
	})

	t.Run("TestWithEnvOption", func(t *testing.T) {
		f := makeField("MyValue", &value, parser, []FieldOption{Env("FOO_BAR")})
		assert.Equal(t, "FOO_BAR", f.envKey)
	})

	t.Run("TestWithRequiredOption", func(t *testing.T) {
		f := makeField("MyValue", &value, parser, []FieldOption{Required()})
		assert.True(t, f.required)
	})
	t.Run("TestWithDefaultOption", func(t *testing.T) {
		f := makeField("MyValue", &value, parser, []FieldOption{Default("baZ")})
		assert.True(t, f.hasDefault)
		assert.Equal(t, "baZ", f.defaultVal)
	})
}
