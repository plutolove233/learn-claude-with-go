package interfaces

import (
	"claudego/pkg/types"
	"context"
)

type Skill interface {
	Name() string
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

type SkillRegistry interface {
	Register(s *types.Skill) error
	Get(name string) (*types.Skill, bool)
	LoadFromDir(dir string) error
	Completions(prefix string) []string
}
