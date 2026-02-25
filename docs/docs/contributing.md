# Contributing

Thank you for your interest in contributing to otel-collector-mcp. This guide covers how to add detection rules, add skills, follow code conventions, and submit pull requests.

## Adding a Detection Rule

Detection rules (analyzers) inspect collector configurations and logs to produce diagnostic findings. Each analyzer is a function with the signature defined in `pkg/analysis/analyzer.go`.

### Analyzer Function Signature

```go
type Analyzer func(ctx context.Context, input *AnalysisInput) []types.DiagnosticFinding
```

The `AnalysisInput` struct provides all the data an analyzer can inspect:

```go
type AnalysisInput struct {
    Config       *collector.CollectorConfig  // Parsed collector config (may be nil)
    DeployMode   collector.DeploymentMode    // DaemonSet, Deployment, StatefulSet, OperatorCRD, Unknown
    Logs         []string                    // Collector pod log lines (may be nil)
    OperatorLogs []string                    // Operator pod log lines (may be nil)
    PodInfo      *corev1.Pod                 // Pod metadata (may be nil)
}
```

### Step 1: Create the analyzer file

Create a new file in `pkg/analysis/` following the naming convention `analyzer_<rule_name>.go`:

```go
// pkg/analysis/analyzer_example_rule.go
package analysis

import (
    "context"

    "github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeExampleRule checks for a specific misconfiguration pattern.
func AnalyzeExampleRule(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
    if input.Config == nil {
        return nil
    }

    var findings []types.DiagnosticFinding

    // Implement your detection logic here.
    // Inspect input.Config, input.DeployMode, input.Logs, etc.

    // If an issue is found, append a DiagnosticFinding:
    findings = append(findings, types.DiagnosticFinding{
        Severity:    types.SeverityWarning,
        Category:    types.CategoryConfig,
        Summary:     "Short description of the issue",
        Detail:      "Detailed explanation of why this is a problem and what the impact is.",
        Suggestion:  "What the user should do to fix it",
        Remediation: "# YAML config snippet showing the fix",
    })

    return findings
}
```

### Step 2: Register the analyzer

Add your analyzer to the `AllAnalyzers()` function in `pkg/analysis/analyzer.go`. If your analyzer requires log data, add it to `AllAnalyzersIncludingLogs()` instead:

```go
func AllAnalyzers() []Analyzer {
    return []Analyzer{
        AnalyzeMissingBatch,
        AnalyzeMissingMemoryLimiter,
        // ... existing analyzers ...
        AnalyzeExampleRule,  // Add your analyzer here
    }
}
```

### Step 3: Write tests

Create a test file `pkg/analysis/analyzer_example_rule_test.go`:

```go
package analysis

import (
    "context"
    "testing"

    "github.com/hrexed/otel-collector-mcp/pkg/collector"
)

func TestAnalyzeExampleRule(t *testing.T) {
    tests := []struct {
        name           string
        input          *AnalysisInput
        expectFindings int
    }{
        {
            name:           "nil config returns no findings",
            input:          &AnalysisInput{},
            expectFindings: 0,
        },
        {
            name: "detects the misconfiguration",
            input: &AnalysisInput{
                Config: &collector.CollectorConfig{
                    // Set up a config that triggers the rule
                },
            },
            expectFindings: 1,
        },
        {
            name: "passes with correct configuration",
            input: &AnalysisInput{
                Config: &collector.CollectorConfig{
                    // Set up a config that does NOT trigger the rule
                },
            },
            expectFindings: 0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            findings := AnalyzeExampleRule(context.Background(), tt.input)
            if len(findings) != tt.expectFindings {
                t.Errorf("expected %d findings, got %d", tt.expectFindings, len(findings))
            }
        })
    }
}
```

### Severity Guidelines

| Severity | When to Use |
|---|---|
| `critical` | Data loss, security vulnerability, or crash-causing misconfiguration |
| `warning` | Performance degradation, best-practice violation, or potential issue |
| `info` | Informational observation that may or may not require action |
| `ok` | Explicit confirmation that a check passed (use sparingly) |

### Category Guidelines

| Category | When to Use |
|---|---|
| `pipeline` | Issues with pipeline wiring (connectors, missing components) |
| `config` | General configuration issues |
| `security` | Hardcoded secrets, insecure endpoints |
| `performance` | Missing batch/memory_limiter, high cardinality |
| `operator` | OTel Operator-specific issues |
| `runtime` | Issues detected from logs (backpressure, OOM, exporter failures) |

## Adding a Skill

Skills are proactive capabilities that generate recommendations or configuration. They implement the `Skill` interface in `pkg/skills/`.

### Skill Interface

```go
type Skill interface {
    Definition() SkillDefinition
    Execute(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error)
}

type SkillDefinition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
}
```

### Step 1: Create the skill file

Create a new file in `pkg/skills/` following the naming convention `skill_<name>.go`:

```go
// pkg/skills/skill_example.go
package skills

import (
    "context"

    "github.com/hrexed/otel-collector-mcp/pkg/config"
    "github.com/hrexed/otel-collector-mcp/pkg/types"
)

type ExampleSkill struct {
    Cfg *config.Config
}

func (s *ExampleSkill) Definition() SkillDefinition {
    return SkillDefinition{
        Name:        "example_skill",
        Description: "Description of what this skill does",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "param1": map[string]interface{}{
                    "type":        "string",
                    "description": "Description of param1",
                },
            },
            "required": []string{"param1"},
        },
    }
}

func (s *ExampleSkill) Execute(_ context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
    param1, _ := args["param1"].(string)

    result := &SkillResult{
        Skill: "example_skill",
        Recommendation: map[string]interface{}{
            "param1": param1,
            "output": "generated content",
        },
        ConfigSnippet: "# Generated YAML config\n",
    }

    meta := s.Cfg.ClusterMetadata()
    return types.NewStandardResponse(meta, "example_skill", result), nil
}
```

### Step 2: Register the skill

Register your skill in `cmd/server/main.go` or wherever skills are initialized. Create the skill instance and add it to the skill registry.

### Step 3: Write tests

Test both the definition (parameter schema) and the execution (output correctness) of your skill.

## Code Style

### Go Conventions

- Follow standard Go formatting (`gofmt`/`goimports`).
- Use meaningful variable names. Avoid single-letter names except for loop indices and receivers.
- Keep functions focused. If a function is longer than 50 lines, consider splitting it.
- Use `context.Context` as the first parameter for any function that does I/O or may be cancelled.
- Return errors rather than panicking. The tool framework has panic recovery, but clean error handling is preferred.

### Logging

Use `log/slog` (Go's structured logging package) for all log output:

```go
slog.Info("descriptive message", "key1", value1, "key2", value2)
slog.Warn("something unexpected", "error", err)
slog.Error("operation failed", "error", err, "namespace", ns)
```

Do not use `log.Printf`, `fmt.Println`, or other unstructured logging. The server outputs JSON-formatted logs via `slog.NewJSONHandler`.

### Testing Requirements

- Every new analyzer must have a corresponding `_test.go` file.
- Test at minimum: nil/empty input, a triggering case, and a passing case.
- Use table-driven tests with `t.Run()` for multiple scenarios.
- Run tests with `go test ./...` before submitting.

### Project Structure

```
cmd/
  server/main.go           # Entry point
pkg/
  analysis/                # Detection rule analyzers
    analyzer.go            # Analyzer type and registration
    analyzer_*.go          # Individual analyzers
    analyzer_*_test.go     # Analyzer tests
    helpers.go             # Shared helper functions
  collector/               # Collector interaction (config, logs, detect)
  config/                  # Server configuration
  discovery/               # CRD feature discovery
  k8s/                     # Kubernetes client initialization
  mcp/                     # MCP HTTP server
  skills/                  # Proactive skills
    types.go               # Skill interface and types
    registry.go            # Skill registry
    skill_*.go             # Individual skills
  telemetry/               # OpenTelemetry instrumentation
  tools/                   # MCP tool implementations
    types.go               # Tool interface and BaseTool
    registry.go            # Tool registry
    tool_*.go              # Individual tools
  types/                   # Shared types (findings, metadata, errors)
deploy/
  helm/                    # Helm chart
```

## PR Process

1. **Fork the repository** and create a feature branch from `main`.
2. **Make your changes** following the code style guidelines above.
3. **Add tests** for any new functionality.
4. **Run the test suite** locally:
   ```bash
   go test ./...
   ```
5. **Run the linter** (if configured):
   ```bash
   go vet ./...
   ```
6. **Commit your changes** with a descriptive commit message. Use conventional commit prefixes:
   - `feat:` for new features (tools, skills, analyzers)
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for test additions or fixes
   - `refactor:` for code restructuring
7. **Open a pull request** against `main` with:
   - A clear title summarizing the change
   - A description explaining what and why
   - A list of testing steps
8. **Address review feedback** and ensure CI passes.

### Checklist for New Analyzers

- [ ] Analyzer function follows the `Analyzer` signature
- [ ] Handles nil `Config` and nil `Logs` gracefully
- [ ] Uses appropriate severity and category constants
- [ ] Includes `Suggestion` and ideally `Remediation` in findings
- [ ] Registered in `AllAnalyzers()` or `AllAnalyzersIncludingLogs()`
- [ ] Test file with nil-input, triggering, and passing test cases
- [ ] All tests pass with `go test ./pkg/analysis/...`
