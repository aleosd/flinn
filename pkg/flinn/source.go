package flinn

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// JSONSource loads configuration values from a JSON file.
// Nested objects are traversed using the path segments passed to Get,
// matching the same dot-separated logical paths that the Loader constructs.
type jsonSource struct {
	data map[string]any
}

// NewJSONSource reads and parses the JSON file at the given path.
// Returns an error if the file cannot be read or is not valid JSON.
func NewJSONSource(path string) (Source, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("flinn: reading json source: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("flinn: parsing json source: %w", err)
	}

	if data == nil {
		return nil, fmt.Errorf("flinn: json source: root must be a JSON object")
	}

	return &jsonSource{data: data}, nil
}

// Get traverses the parsed JSON using path as a sequence of keys.
// Returns the string representation of the leaf value, or (_, false, nil)
// if any segment along the path is absent.
// Returns an error if an intermediate segment exists but is not an object,
// or if the final value is not a scalar (e.g. it is a nested object or array).
func (s *jsonSource) Get(path []string) (string, bool, error) {
	var current any = s.data

	for i, segment := range path {
		m, ok := current.(map[string]any)
		if !ok {
			// A non-final segment resolved to a scalar — path is too deep.
			return "", false, fmt.Errorf(
				"flinn: json path %q: segment %q is not an object",
				joinPath(path), segment,
			)
		}

		val, exists := m[segment]
		if !exists {
			return "", false, nil
		}

		if i == len(path)-1 {
			// Leaf: coerce to string via the package-level stringify helper.
			str, ok := stringify(val)
			if !ok {
				return "", false, fmt.Errorf(
					"flinn: json path %q: value is not a scalar (got %T)",
					joinPath(path), val,
				)
			}
			return str, true, nil
		}

		current = val
	}

	return "", false, nil
}

func joinPath(segments []string) string {
	var result strings.Builder
	for i, s := range segments {
		if i > 0 {
			result.WriteString(".")
		}
		result.WriteString(s)
	}
	return result.String()
}
