package flinn

import (
	"fmt"
	"strings"
)

type FieldError struct {
	Path  string // "Database.Primary.Port"
	Rule  string // "required" | "min" | "max" | "oneof" | "parse" | "custom"
	Value any    // the offending value, nil if absent
	Msg   string
}

type FieldErrors []FieldError

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
