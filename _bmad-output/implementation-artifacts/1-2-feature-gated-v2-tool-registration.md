# Story 1.2: Feature-Gated v2 Tool Registration

Status: done

## Story

As a platform engineer,
I want v2 tools to only appear in MCP `tools/list` when `v2.enabled=true`,
So that clusters not ready for v2 see no v2 functionality.

## Acceptance Criteria

1. **Given** the MCP server starts with `V2_ENABLED=false`
   **When** a client calls `tools/list`
   **Then** only the 7 v1 tools are returned — no v2 tools visible

2. **Given** the MCP server starts with `V2_ENABLED=true`
   **When** a client calls `tools/list`
   **Then** all 7 v1 tools AND all 10 v2 tools are returned

3. **Given** `V2_ENABLED=false`
   **When** a client attempts to invoke any v2 tool directly
   **Then** the server returns a structured error indicating the tool is not available

## Tasks / Subtasks

- [x] Task 1: Create v2 tool stubs (AC: #2)
  - [x] Create `pkg/tools/v2_stubs.go` with 10 stub tool types: check_health, start_analysis, rollback_config, capture_signals, cleanup_debug, detect_issues, suggest_fixes, apply_fix, recommend_sampling, recommend_sizing
  - [x] Each stub implements Tool interface with proper Name(), Description(), InputSchema()
  - [x] Each stub's Run() returns a "not yet implemented" MCPError

- [x] Task 2: Add conditional v2 tool registration in main.go (AC: #1, #2)
  - [x] Add `RegisterV2Tools(registry, baseTool)` function in `pkg/tools/v2_registration.go`
  - [x] In `cmd/server/main.go`, call `RegisterV2Tools` only when `cfg.V2Enabled == true`
  - [x] Log v2 tool registration status

- [x] Task 3: Add tests for feature-gated registration (AC: #1, #2, #3)
  - [x] Test registry is empty without v2 registration
  - [x] Test registry has 10 v2 tools after RegisterV2Tools
  - [x] Test v2 tool names are correct
  - [x] Test v2 stub tools return error on invocation

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Completion Notes List

- Created v2_stubs.go with data-driven stub tool implementation using v2ToolDefs slice
- Created v2_registration.go with RegisterV2Tools function
- Modified main.go to conditionally register v2 tools based on cfg.V2Enabled
- Added 3 test functions verifying registration count, tool names, and stub error behavior
- All tests pass, no regressions

### File List

- pkg/tools/v2_stubs.go (new)
- pkg/tools/v2_registration.go (new)
- pkg/tools/v2_registration_test.go (new)
- cmd/server/main.go (modified)
