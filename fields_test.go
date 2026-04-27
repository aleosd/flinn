package flinn

import (
	"fmt"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		f := String("MyValue", &value).FileKey("foo_bar")
		assert.Equal(t, "foo_bar", f.comm.fileKey)
	})
	t.Run("TestWithoutOptions", func(t *testing.T) {
		f := makeField("MyValue", &value, parser)
		assert.Equal(t, "my_value", f.comm.fileKey)
		assert.Equal(t, "", f.envSegment())
		assert.False(t, f.required)
		assert.False(t, f.field.hasDefault)
		assert.Equal(t, "", f.field.defaultVal)
		assert.Empty(t, f.field.validators)
	})

	t.Run("TestWithEnvOption", func(t *testing.T) {
		f := makeField("MyValue", &value, parser).Env("FOO_BAR")
		assert.Equal(t, "FOO_BAR", f.comm.envSegment)
	})

	t.Run("TestWithRequiredOption", func(t *testing.T) {
		f := makeField("MyValue", &value, parser).Required()
		assert.True(t, f.required)
	})
	t.Run("TestWithDefaultOption", func(t *testing.T) {
		f := makeField("MyValue", &value, parser).Default("baZ")
		assert.True(t, f.field.hasDefault)
		assert.Equal(t, "baZ", f.field.defaultVal)
	})
}

func TestNumericFieldRequired(t *testing.T) {
	t.Run("IntRequired", func(t *testing.T) {
		var value int
		f := Int("MyValue", &value).Required()
		assert.True(t, f.required)
		assert.True(t, f.isRequired())
	})

	t.Run("FloatRequired", func(t *testing.T) {
		var value float64
		f := Float("MyValue", &value).Required()
		assert.True(t, f.required)
		assert.True(t, f.isRequired())
	})
}

func TestUUIDField(t *testing.T) {
	t.Run("TestSuccess", func(t *testing.T) {
		var value uuid.UUID
		var UUIDValue = uuid.New()
		f := UUID("MyValue", &value)
		err := f.assign(UUIDValue.String())

		require.NoError(t, err)
		assert.Equal(t, UUIDValue, value)
	})

	t.Run("TestError", func(t *testing.T) {
		tests := []string{"", "2", "22", "foo", "-", " ", "965a158e-1e29-4746-9191-8d30efef4axX"}
		for _, input := range tests {
			t.Run(fmt.Sprintf("input=%q", input), func(t *testing.T) {
				var value uuid.UUID
				f := UUID("MyValue", &value)
				err := f.assign(input)

				require.Error(t, err)
			})
		}
	})
}

func TestBoolField(t *testing.T) {
	t.Run("TestSuccess", func(t *testing.T) {
		tests := []struct {
			input string
			want  bool
		}{
			{"0", false},
			{"f", false},
			{"F", false},
			{"false", false},
			{"False", false},
			{"FaLsE", false},
			{"true", true},
			{"1", true},
			{"T", true},
			{"t", true},
			{"True", true},
			{"TRUE", true},
			{"tRUe", true},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
				var value bool
				f := Bool("MyValue", &value)
				err := f.assign(tt.input)

				require.NoError(t, err)
				assert.Equal(t, tt.want, value)
			})
		}
	})

	t.Run("TestError", func(t *testing.T) {
		tests := []string{"", "2", "22", "foo", "-", "111", "000", " "}
		for _, input := range tests {
			t.Run(fmt.Sprintf("input=%q", input), func(t *testing.T) {
				var value bool
				f := Bool("MyValue", &value)
				err := f.assign(input)

				require.Error(t, err)
			})
		}
	})
}

func TestFloatField(t *testing.T) {
	t.Run("TestSuccess", func(t *testing.T) {
		tests := []struct {
			input string
			want  float64
		}{
			{"0", 0.0},
			{"-0", 0.0},
			{".1", 0.1},
			{"0.0", 0.0},
			{"0.1", 0.1},
			{"-3.14", -3.14},
			{"1", 1.0},
			{"999", 999.0},
			{"1e3", 1000.0},
			{"2.5E-1", 0.25},
			{"Inf", math.Inf(1)},
			{"-Inf", math.Inf(-1)},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
				var value float64
				f := Float("MyValue", &value)
				err := f.assign(tt.input)

				require.NoError(t, err)
				assert.Equal(t, tt.want, value)
			})
		}
	})

	t.Run("TestError", func(t *testing.T) {
		tests := []string{"", "foo", "-", " "}
		for _, input := range tests {
			t.Run(fmt.Sprintf("input=%q", input), func(t *testing.T) {
				var value float64
				f := Float("MyValue", &value)
				err := f.assign(input)

				require.Error(t, err)
			})
		}
	})
}
