package tools

type Tool interface {
	Name() string
	Description() string
	Execute(input map[string]any) (string, error)
	Parameters() map[string]any
}
