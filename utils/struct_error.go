package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// formatValidationError converts go-playground/validator errors into a
// single human-readable string with one line per failing field.
//
// Example:
//
//	input validation failed:
//	  - action: must be one of [read write] (got "append")
//	  - content: required when Action is write
func FormatValidationError(err error) error {
	var verr validator.ValidationErrors
	if !errors.As(err, &verr) {
		return err
	}
	var b strings.Builder
	b.WriteString("input validation failed:")
	for _, fe := range verr {
		fmt.Fprintf(&b, "\n  - %s: ", fe.Field())
		switch fe.Tag() {
		case "required":
			b.WriteString("required but missing")
		case "required_if":
			fmt.Fprintf(&b, "required when %s", fe.Param())
		case "oneof":
			fmt.Fprintf(&b, "must be one of [%s] (got %q)", fe.Param(), fe.Value())
		case "filepath":
			fmt.Fprintf(&b, "invalid file path (got %q)", fe.Value())
		default:
			fmt.Fprintf(&b, "failed %q constraint (param: %s, value: %v)",
				fe.Tag(), fe.Param(), fe.Value())
		}
	}
	return errors.New(b.String())
}
