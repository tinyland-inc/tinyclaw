package campaign

import "context"

// BackendAdapter is the interface that campaign backends must implement.
// This allows campaigns to target different agent systems.
type BackendAdapter interface {
	// Execute sends a prompt to the specified agent and returns the response.
	Execute(ctx context.Context, agentID, prompt string, tools []string) (string, error)
	// Name returns the backend identifier.
	Name() string
}

// GuardrailCheck evaluates whether an execution violates any guardrails.
// Returns nil if all guardrails pass, or an error describing the violation.
func GuardrailCheck(exec *Execution) error {
	if exec == nil || exec.Definition == nil {
		return nil
	}
	reason := checkGuardrails(exec, &exec.Definition.Guardrails)
	if reason != "" {
		return &GuardrailError{Reason: reason, CampaignID: exec.ID}
	}
	return nil
}

// GuardrailError represents a guardrail violation.
type GuardrailError struct {
	Reason     string
	CampaignID string
}

func (e *GuardrailError) Error() string {
	return "campaign " + e.CampaignID + ": guardrail violation: " + e.Reason
}

// CanExecuteTool checks whether a tool call is permitted under the current guardrails.
func CanExecuteTool(exec *Execution, toolName string) bool {
	if exec == nil || exec.Definition == nil {
		return true
	}
	g := &exec.Definition.Guardrails

	// Kill switch
	if exec.KillSwitchUsed {
		return false
	}

	// Read-only mode: deny write operations
	if g.ReadOnly {
		switch toolName {
		case "write_file", "exec_command", "delete_file":
			return false
		}
	}

	// Tool call limit
	if exec.ToolCalls >= g.MaxToolCalls {
		return false
	}

	return true
}

// RecordToolCall increments the tool call counter and budget spend.
func RecordToolCall(exec *Execution, costCents int) {
	if exec == nil {
		return
	}
	exec.ToolCalls++
	exec.SpentCents += costCents
}
