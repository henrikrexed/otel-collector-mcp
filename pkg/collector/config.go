package collector

import "gopkg.in/yaml.v3"

// CollectorConfig represents the parsed OTel Collector configuration.
type CollectorConfig struct {
	Receivers  map[string]interface{} `yaml:"receivers"`
	Processors map[string]interface{} `yaml:"processors"`
	Exporters  map[string]interface{} `yaml:"exporters"`
	Connectors map[string]interface{} `yaml:"connectors"`
	Service    ServiceConfig          `yaml:"service"`
}

// ServiceConfig holds the service section of the collector config.
type ServiceConfig struct {
	Pipelines  map[string]PipelineConfig `yaml:"pipelines"`
	Extensions []string                  `yaml:"extensions,omitempty"`
}

// PipelineConfig represents a single pipeline within the service config.
type PipelineConfig struct {
	Receivers  []string `yaml:"receivers"`
	Processors []string `yaml:"processors"`
	Exporters  []string `yaml:"exporters"`
}

// ParseConfig parses a YAML string into CollectorConfig.
func ParseConfig(data []byte) (*CollectorConfig, error) {
	var cfg CollectorConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
