# Story 1.1: v2 Configuration & Error Codes

Status: done

## Story

As a platform engineer,
I want the MCP server to support v2 configuration fields (`V2Enabled`, `SessionTTL`, `MaxConcurrentSessions`) via environment variables,
So that I can control v2 behavior without rebuilding the server.

## Acceptance Criteria

1. **Given** the MCP server binary with v2 code
   **When** `V2_ENABLED=true`, `V2_SESSION_TTL=10m`, `V2_MAX_SESSIONS=5` environment variables are set
   **Then** `pkg/config/config.go` parses these into the Config struct with correct types and defaults
   **And** `V2_ENABLED` defaults to `false`, `V2_SESSION_TTL` defaults to `10m`, `V2_MAX_SESSIONS` defaults to `5`

2. **Given** v2 operations encounter error conditions
   **When** errors are returned
   **Then** they use v2-specific error codes (`session_not_found`, `session_expired`, `concurrent_session`, `production_refused`, `backup_failed`, `rollback_failed`, `health_check_failed`, `mutation_failed`, `capture_failed`, `gitops_conflict`) defined in `pkg/types/errors.go`

## Tasks / Subtasks

- [x] Task 1: Add v2 fields to Config struct (AC: #1)
  - [x] Add `V2Enabled bool` field to `Config` struct in `pkg/config/config.go`
  - [x] Add `SessionTTL time.Duration` field to `Config` struct
  - [x] Add `MaxConcurrentSessions int` field to `Config` struct
  - [x] Parse `V2_ENABLED` env var as bool, default `false`
  - [x] Parse `V2_SESSION_TTL` env var as duration string via `time.ParseDuration`, default `10m`
  - [x] Parse `V2_MAX_SESSIONS` env var as int via `strconv.Atoi`, default `5`
  - [x] Add import for `time` package

- [x] Task 2: Add v2 error codes to errors.go (AC: #2)
  - [x] Add 10 new error code constants to `pkg/types/errors.go`
  - [x] Error codes use UPPER_SNAKE_CASE matching v1 convention: `SESSION_NOT_FOUND`, `SESSION_EXPIRED`, `CONCURRENT_SESSION`, `PRODUCTION_REFUSED`, `BACKUP_FAILED`, `ROLLBACK_FAILED`, `HEALTH_CHECK_FAILED`, `MUTATION_FAILED`, `CAPTURE_FAILED`, `GITOPS_CONFLICT`

- [x] Task 3: Add unit tests for v2 config parsing (AC: #1)
  - [x] Test default values when no env vars are set
  - [x] Test custom values when env vars are set
  - [x] Test invalid V2_SESSION_TTL falls back to default with warning
  - [x] Test invalid V2_MAX_SESSIONS falls back to default with warning
  - [x] Follow existing test pattern in `pkg/config/config_test.go` using `t.Setenv()`

## Dev Notes

### Architecture Compliance

- **Config pattern**: Follow existing `NewFromEnv()` pattern exactly — env var reading with sensible defaults, slog warnings on parse failures
- **Error code convention**: v1 uses UPPER_SNAKE_CASE (`COLLECTOR_NOT_FOUND`, `CONFIG_PARSE_FAILED`). v2 codes MUST follow same convention
- **No config files**: All config via environment variables only
- **Module path**: `github.com/hrexed/otel-collector-mcp`

### Existing Code to Modify

| File | Change |
|------|--------|
| `pkg/config/config.go` | Add 3 fields to `Config` struct, add parsing logic in `NewFromEnv()` |
| `pkg/types/errors.go` | Add 10 new `ErrCode` constants in the existing `const` block |
| `pkg/config/config_test.go` | Add test functions for v2 config fields |

### Technical Requirements

- `SessionTTL` must be `time.Duration` type — use `time.ParseDuration()` for parsing
- Duration default is `10m` (10 minutes) — `10 * time.Minute`
- `MaxConcurrentSessions` default is `5`
- `V2Enabled` default is `false` — follows same `strconv.ParseBool` pattern as `OTelEnabled`
- On invalid `V2_SESSION_TTL`, log warning with `slog.Warn` and use default
- On invalid `V2_MAX_SESSIONS`, log warning with `slog.Warn` and use default

### Code Patterns to Follow

```go
// Config struct addition pattern (pkg/config/config.go):
type Config struct {
    // ... existing fields ...
    V2Enabled            bool
    SessionTTL           time.Duration
    MaxConcurrentSessions int
}

// Parsing pattern in NewFromEnv() - follows existing OTelEnabled pattern:
v2Enabled := false
if v := os.Getenv("V2_ENABLED"); v != "" {
    parsed, err := strconv.ParseBool(v)
    if err != nil {
        slog.Warn("invalid V2_ENABLED value, defaulting to false")
    } else {
        v2Enabled = parsed
    }
}

sessionTTL := 10 * time.Minute
if v := os.Getenv("V2_SESSION_TTL"); v != "" {
    parsed, err := time.ParseDuration(v)
    if err != nil {
        slog.Warn("invalid V2_SESSION_TTL value, defaulting to 10m")
    } else {
        sessionTTL = parsed
    }
}

maxSessions := 5
if v := os.Getenv("V2_MAX_SESSIONS"); v != "" {
    parsed, err := strconv.Atoi(v)
    if err != nil {
        slog.Warn("invalid V2_MAX_SESSIONS value, defaulting to 5")
    } else {
        maxSessions = parsed
    }
}

// Error code pattern (pkg/types/errors.go):
const (
    // ... existing codes ...
    ErrCodeSessionNotFound   = "SESSION_NOT_FOUND"
    ErrCodeSessionExpired    = "SESSION_EXPIRED"
    ErrCodeConcurrentSession = "CONCURRENT_SESSION"
    ErrCodeProductionRefused = "PRODUCTION_REFUSED"
    ErrCodeBackupFailed      = "BACKUP_FAILED"
    ErrCodeRollbackFailed    = "ROLLBACK_FAILED"
    ErrCodeHealthCheckFailed = "HEALTH_CHECK_FAILED"
    ErrCodeMutationFailed    = "MUTATION_FAILED"
    ErrCodeCaptureFailed     = "CAPTURE_FAILED"
    ErrCodeGitOpsConflict    = "GITOPS_CONFLICT"
)
```

### Testing Pattern

```go
// Follow existing config_test.go pattern:
func TestNewFromEnvV2Defaults(t *testing.T) {
    cfg := NewFromEnv()
    if cfg.V2Enabled != false { t.Errorf("expected V2Enabled=false, got %v", cfg.V2Enabled) }
    if cfg.SessionTTL != 10*time.Minute { t.Errorf("expected SessionTTL=10m, got %v", cfg.SessionTTL) }
    if cfg.MaxConcurrentSessions != 5 { t.Errorf("expected MaxConcurrentSessions=5, got %v", cfg.MaxConcurrentSessions) }
}

func TestNewFromEnvV2Custom(t *testing.T) {
    t.Setenv("V2_ENABLED", "true")
    t.Setenv("V2_SESSION_TTL", "30m")
    t.Setenv("V2_MAX_SESSIONS", "10")
    cfg := NewFromEnv()
    if cfg.V2Enabled != true { t.Errorf(...) }
    if cfg.SessionTTL != 30*time.Minute { t.Errorf(...) }
    if cfg.MaxConcurrentSessions != 10 { t.Errorf(...) }
}
```

### Project Structure Notes

- All source under `pkg/` with package-per-concern layout
- No internal/ directory; public API is pkg-level
- Tests are co-located with source files (`*_test.go` alongside `.go`)
- Build via Makefile: `make build`, `make test`, `make lint`

### References

- [Source: pkg/config/config.go] — Existing Config struct and NewFromEnv() pattern
- [Source: pkg/types/errors.go] — Existing error codes and MCPError type
- [Source: pkg/config/config_test.go] — Existing test patterns
- [Source: _bmad-output/epics-v2.md#Epic1-Story1.1] — Story requirements
- [Source: _bmad-output/architecture-v2.md] — v2 config architecture decisions

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Added 3 v2 config fields (V2Enabled, SessionTTL, MaxConcurrentSessions) to Config struct with env var parsing and sensible defaults
- Added 10 v2 error code constants following existing UPPER_SNAKE_CASE convention
- Added 2 new test functions (TestNewFromEnvV2Custom, TestNewFromEnvV2InvalidValues) and extended TestNewFromEnvDefaults with v2 field assertions
- All 5 config tests pass, full project builds successfully, no regressions

### File List

- pkg/config/config.go (modified)
- pkg/types/errors.go (modified)
- pkg/config/config_test.go (modified)
