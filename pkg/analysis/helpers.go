package analysis

import "github.com/hrexed/otel-collector-mcp/pkg/collector"

// pipelineHasProcessor checks if a pipeline contains a processor with the given prefix.
func pipelineHasProcessor(pipeline collector.PipelineConfig, prefix string) bool {
	for _, p := range pipeline.Processors {
		if p == prefix || len(p) > len(prefix) && p[:len(prefix)+1] == prefix+"/" {
			return true
		}
	}
	return false
}

// getNestedMap tries to get a nested map from a parent map.
func getNestedMap(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	nested, ok := v.(map[string]interface{})
	return nested, ok
}

// getNestedString tries to get a nested string from a parent map.
func getNestedString(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
