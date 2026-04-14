package tools

import (
	"claudego/pkg/types"
	"claudego/utils"
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// validate is shared across all TypedTool instances (safe for concurrent use).
var validate = validator.New(validator.WithRequiredStructEnabled())

// BaseTool is a generic base for tools whose inputs are described by a
// concrete struct T with validate tags.
//
// The Execute pipeline is fixed and shared:
//
//	byte slice  →  json.Unmarshal into T
//	→  validate.Struct(T)            (tag-driven field constraints)
//	→  extraValidate(T)              (optional runtime checks, e.g. path sandbox)
//	→  fn(T)                         (business logic)
//
// Concrete tools only need to supply T, fn, and optionally extraValidate.
type BaseTool[T any] struct {
	name          string
	description   string
	parameters    map[string]any
	metadata      types.ToolMetadata
	fn            func(T) (string, error)
	extraValidate func(T) error
}

func (t *BaseTool[T]) Name() string               { return t.name }
func (t *BaseTool[T]) Description() string        { return t.description }
func (t *BaseTool[T]) Metadata() types.ToolMetadata     { return t.metadata }
func (t *BaseTool[T]) Parameters() map[string]any { return t.parameters }

// Execute runs the fixed pipeline: parse → validate → business logic.
func (t *BaseTool[T]) Execute(ctx context.Context, input []byte) (string, error) {
	var p T
	if err := json.Unmarshal(input, &p); err != nil {
		return "", fmt.Errorf("parse input into %T: %w", p, err)
	}

	// Phase 1: tag-driven validation (required, oneof, required_if, …)
	if err := validate.Struct(p); err != nil {
		return "", utils.FormatValidationError(err)
	}

	// Phase 2: optional runtime validation (sandbox checks, cross-field
	// rules that depend on external state, etc.)
	if t.extraValidate != nil {
		if err := t.extraValidate(p); err != nil {
			return "", err
		}
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return t.fn(p)
	}
}
