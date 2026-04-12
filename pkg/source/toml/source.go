package toml

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	gotoml "github.com/pelletier/go-toml/v2"
)

// tomlSource loads configuration values from a TOML file.
// Nested tables are traversed using the path segments passed to Get,
// matching the same dot-separated logical paths that the Loader constructs.
type tomlSource struct {
	data map[string]any
}

// NewTOMLSource reads and parses the TOML file at the given path.
// Returns an error if the file cannot be read or is not valid TOML.
// The returned *tomlSource satisfies the flinn.Source interface.
func NewTOMLSource(path string) (*tomlSource, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("flinn: reading toml source: %w", err)
	}

	var data map[string]any
	if err := gotoml.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("flinn: parsing toml source: %w", err)
	}

	if data == nil {
		return nil, fmt.Errorf("flinn: toml source: root must be a TOML table")
	}

	return &tomlSource{data: data}, nil
}

// Get traverses the parsed TOML using path as a sequence of keys.
// Returns the string representation of the leaf value, or (_, false, nil)
// if any segment along the path is absent.
// Returns an error if an intermediate segment exists but is not a table,
// or if the final value is not a scalar (e.g. it is a nested table or array).
func (s *tomlSource) Get(path []string) (string, bool, error) {
	var current any = s.data

	for i, segment := range path {
		m, ok := current.(map[string]any)
		if !ok {
			// A non-final segment resolved to a scalar — path is too deep.
			return "", false, fmt.Errorf(
				"flinn: toml path %q: segment %q is not a table",
				strings.Join(path, "."), segment,
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
					"flinn: toml path %q: value is not a scalar (got %T)",
					strings.Join(path, "."), val,
				)
			}
			return str, true, nil
		}

		current = val
	}

	return "", false, nil
}

// stringify converts a TOML leaf value to its string representation.
// TOML integers unmarshal as int64 (unlike JSON's float64).
// TOML offset datetimes unmarshal as time.Time, formatted as RFC3339.
// TOML local dates, times, and datetimes unmarshal as go-toml's Local* types,
// formatted via their String() methods (ISO 8601 subsets).
// Returns (_, false) for nil (treated as absent) and non-scalar types
// such as tables and arrays.
func stringify(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true

	case int64:
		return strconv.FormatInt(val, 10), true

	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), true

	case bool:
		return strconv.FormatBool(val), true

	case time.Time:
		return val.Format(time.RFC3339), true

	case gotoml.LocalDateTime:
		return val.String(), true

	case gotoml.LocalDate:
		return val.String(), true

	case gotoml.LocalTime:
		return val.String(), true

	case nil:
		return "", false // null treated as absent

	default:
		// map[string]any or []any — not a scalar
		return "", false
	}
}
