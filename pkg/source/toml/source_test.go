package toml

import (
	"strings"
	"testing"

	gotoml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTOMLSource(t *testing.T) {
	t.Run("FailsIfFileDoesNotExist", func(t *testing.T) {
		_, err := NewTOMLSource("does_not_exist.toml")
		require.Error(t, err)
		assert.ErrorContains(t, err, "reading toml source")
	})

	t.Run("FailsIfFileIsNotTOML", func(t *testing.T) {
		_, err := NewTOMLSource("testdata/invalid.toml")
		require.Error(t, err)
		assert.ErrorContains(t, err, "parsing toml source")
	})

	t.Run("LoadsValidTOML", func(t *testing.T) {
		s, err := NewTOMLSource("testdata/config.toml")
		require.NoError(t, err)
		assert.NotNil(t, s)
	})
}

func TestTOMLSource(t *testing.T) {
	var tomlString = `
[baz]
spanm = 13
ham   = false
`
	var data map[string]any
	err := gotoml.Unmarshal([]byte(tomlString), &data)
	require.NoError(t, err)
	source := &tomlSource{data: data}

	t.Run("GetSuccess", func(t *testing.T) {
		tests := []struct {
			path     []string
			expected string
		}{
			{[]string{"baz", "spanm"}, "13"},
			{[]string{"baz", "ham"}, "false"},
		}
		for _, tt := range tests {
			t.Run(strings.Join(tt.path, "."), func(t *testing.T) {
				got, ok, err := source.Get(tt.path)
				require.NoError(t, err)
				assert.True(t, ok)
				assert.Equal(t, tt.expected, got)
			})
		}
	})

	t.Run("ReturnsFalseIfKeyMissing", func(t *testing.T) {
		_, ok, err := source.Get([]string{"does_not_exist"})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("ReturnsFalseIfEmptyPath", func(t *testing.T) {
		_, ok, err := source.Get([]string{})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("ReturnsErrorIfLeafIsTable", func(t *testing.T) {
		_, ok, err := source.Get([]string{"baz"})
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("ReturnsErrorIfPathTooDeep", func(t *testing.T) {
		_, ok, err := source.Get([]string{"baz", "spanm", "extra"})
		require.Error(t, err)
		assert.False(t, ok)
	})
}

func TestTOMLSource_DatetimeTypes(t *testing.T) {
	source, err := NewTOMLSource("testdata/datetime.toml")
	require.NoError(t, err)

	t.Run("OffsetDatetime_FormattedAsRFC3339", func(t *testing.T) {
		got, ok, err := source.Get([]string{"offset_dt"})
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "2024-01-15T10:30:00Z", got)
	})

	t.Run("LocalDatetime", func(t *testing.T) {
		got, ok, err := source.Get([]string{"local_dt"})
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "2024-01-15T10:30:00", got)
	})

	t.Run("LocalDate", func(t *testing.T) {
		got, ok, err := source.Get([]string{"local_d"})
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "2024-01-15", got)
	})

	t.Run("LocalTime", func(t *testing.T) {
		got, ok, err := source.Get([]string{"local_t"})
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "10:30:00", got)
	})
}

func TestTOMLSource_StringifyTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		ok       bool
	}{
		{"string", "hello", "hello", true},
		{"int64", int64(42), "42", true},
		{"float64", float64(3.14), "3.14", true},
		{"bool_true", true, "true", true},
		{"bool_false", false, "false", true},
		{"nil", nil, "", false},
		{"map", map[string]any{"k": "v"}, "", false},
		{"slice", []any{"a", "b"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := stringify(tt.input)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, got)
		})
	}
}
