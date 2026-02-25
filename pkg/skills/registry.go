package skills

import (
	"context"
	"sync"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// Skill is the interface for proactive MCP skills.
type Skill interface {
	Definition() SkillDefinition
	Execute(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error)
}

// Registry holds registered skills.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewRegistry creates a new skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

// Register adds a skill to the registry.
func (r *Registry) Register(skill Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[skill.Definition().Name] = skill
}

// Get returns a skill by name.
func (r *Registry) Get(name string) Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.skills[name]
}

// All returns all registered skills.
func (r *Registry) All() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	return result
}
