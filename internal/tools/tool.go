package tools

type Tool interface {
	Name() string
	Description() string
	Execute(input map[string]interface{}) (string, error)
}
