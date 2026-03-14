package mutator

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const debugExporterKey = "debug"

// InjectDebugExporter adds a debug exporter to the collector config YAML.
// If pipelines is empty, the debug exporter is added to ALL pipelines.
// The injection is append-only — no existing components are modified.
func InjectDebugExporter(configYAML string, pipelines []string) (string, []string, error) {
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return "", nil, fmt.Errorf("failed to parse collector config: %w", err)
	}

	// Add debug exporter to exporters section
	exporters, _ := config["exporters"].(map[string]interface{})
	if exporters == nil {
		exporters = make(map[string]interface{})
		config["exporters"] = exporters
	}

	// Check if debug exporter already exists (idempotent)
	if _, exists := exporters[debugExporterKey]; exists {
		return configYAML, nil, nil // Already injected, skip
	}

	exporters[debugExporterKey] = map[string]interface{}{
		"verbosity": "basic",
	}

	// Add debug to pipeline exporter lists
	service, _ := config["service"].(map[string]interface{})
	if service == nil {
		return "", nil, fmt.Errorf("no service section found in config")
	}

	allPipelines, _ := service["pipelines"].(map[string]interface{})
	if allPipelines == nil {
		return "", nil, fmt.Errorf("no pipelines section found in service config")
	}

	var injectedPipelines []string

	for pipelineName, pipelineVal := range allPipelines {
		// If specific pipelines are requested, skip non-matching
		if len(pipelines) > 0 && !contains(pipelines, pipelineName) {
			continue
		}

		pipeline, ok := pipelineVal.(map[string]interface{})
		if !ok {
			continue
		}

		exporterList := toStringSlice(pipeline["exporters"])
		if !contains(exporterList, debugExporterKey) {
			exporterList = append(exporterList, debugExporterKey)
			pipeline["exporters"] = exporterList
			injectedPipelines = append(injectedPipelines, pipelineName)
		}
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(out), injectedPipelines, nil
}

// RemoveDebugExporter removes the debug exporter from the collector config YAML.
// Approved fixes applied during the session are preserved.
func RemoveDebugExporter(configYAML string) (string, []string, error) {
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return "", nil, fmt.Errorf("failed to parse collector config: %w", err)
	}

	// Remove debug from exporters section
	exporters, _ := config["exporters"].(map[string]interface{})
	if exporters != nil {
		delete(exporters, debugExporterKey)
	}

	// Remove debug from all pipeline exporter lists
	service, _ := config["service"].(map[string]interface{})
	if service == nil {
		out, err := yaml.Marshal(config)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		return string(out), nil, nil
	}
	allPipelines, _ := service["pipelines"].(map[string]interface{})

	var removedFrom []string

	for pipelineName, pipelineVal := range allPipelines {
		pipeline, ok := pipelineVal.(map[string]interface{})
		if !ok {
			continue
		}

		exporterList := toStringSlice(pipeline["exporters"])
		newList := removeFromSlice(exporterList, debugExporterKey)
		if len(newList) != len(exporterList) {
			pipeline["exporters"] = newList
			removedFrom = append(removedFrom, pipelineName)
		}
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(out), removedFrom, nil
}

func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case []string:
		return s
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
