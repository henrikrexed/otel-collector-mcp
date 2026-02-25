package skills

// SkillDefinition describes a proactive MCP skill.
type SkillDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// SkillResult is the output of a skill execution.
type SkillResult struct {
	Skill          string      `json:"skill"`
	Recommendation interface{} `json:"recommendation"`
	ConfigSnippet  string      `json:"configSnippet,omitempty"`
}
