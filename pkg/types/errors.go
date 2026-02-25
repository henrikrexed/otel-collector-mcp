package types

import "fmt"

// Error code constants.
const (
	ErrCodeCollectorNotFound = "COLLECTOR_NOT_FOUND"
	ErrCodeConfigParseFailed = "CONFIG_PARSE_FAILED"
	ErrCodeRBACInsufficient  = "RBAC_INSUFFICIENT"
	ErrCodeLogAccessFailed   = "LOG_ACCESS_FAILED"
)

// MCPError is a structured error for MCP tool responses.
type MCPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewMCPError creates a new MCPError.
func NewMCPError(code, message string) *MCPError {
	return &MCPError{Code: code, Message: message}
}
