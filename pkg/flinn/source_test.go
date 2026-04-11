package flinn

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONSource(t *testing.T) {
	t.Run("FailsIfFileDoesNotExist", func(t *testing.T) {
		_, err := NewJSONSource("does_not_exist.json")
		assert.Error(t, err)
	})
}

func TestJSONSource(t *testing.T) {
	var jsonString = `{"foo": "bar", "baz": {"spanm": 13, "ham": false}}`
	var data map[string]any
	err := json.Unmarshal([]byte(jsonString), &data)
	require.NoError(t, err)
	source := &jsonSource{data: data}

	t.Run("TestGetSuccess", func(t *testing.T) {
		tests := []struct {
			path     []string
			expected string
		}{
			{[]string{"foo"}, "bar"},
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

	t.Run("TestGetReturnsFalseIfNoKey", func(t *testing.T) {
		_, ok, err := source.Get([]string{"does_not_exist"})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("TestGetReturnsFalseIfEmptyPath", func(t *testing.T) {
		_, ok, err := source.Get([]string{})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("TestGetReturnsErrorIfNotFinalSegment", func(t *testing.T) {
		_, ok, err := source.Get([]string{"baz"})
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("TestGetReturnsErrorIfDataNotMap", func(t *testing.T) {
		_, ok, err := source.Get([]string{"baz", "spanm", "ham"})
		require.Error(t, err)
		assert.False(t, ok)
	})
}
