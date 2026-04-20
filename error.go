package flinn

import (
	"fmt"
	"strings"
)

// FieldError represents an error that occurred while loading or validating a specific configuration field.
type FieldError struct {
	Path  string // "Database.Primary.Port"
	Rule  string // "required" | "min" | "max" | "oneof" | "parse" | "custom"
	Value any    // the offending value, nil if absent
	Msg   string
}

// FieldErrors is a collection of FieldError values.
// It implements the error interface and provides a formatted string of all collected errors.
type FieldErrors []FieldError

// Error returns a string representation of all collected field errors, one per line.
func (e FieldErrors) Error() string {
	var b strings.Builder
	for _, fe := range e {
		fmt.Fprintf(&b, "%s: [%s] %s\n", fe.Path, fe.Rule, fe.Msg)
	}
	return b.String()
}

func (e *FieldErrors) add(path, rule string, value any, msg string) {
	*e = append(*e, FieldError{Path: path, Rule: rule, Value: value, Msg: msg})
}
