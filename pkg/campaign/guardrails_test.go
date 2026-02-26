package campaign

import (
	"testing"
	"time"
)

func TestGuardrailCheck_Clean(t *testing.T) {
	exec := &Execution{
		ID:        "test",
		StartTime: time.Now(),
		Definition: &Definition{
			Guardrails: DefaultGuardrails(),
		},
	}

	if err := GuardrailCheck(exec); err != nil {
		t.Errorf("expected no guardrail violation, got: %v", err)
	}
}

func TestGuardrailCheck_BudgetExhausted(t *testing.T) {
	exec := &Execution{
		ID:         "test",
		StartTime:  time.Now(),
		SpentCents: 1500,
		Definition: &Definition{
			Guardrails: Guardrails{
				MaxDurationMinutes: 60,
				AIAPIBudgetCents:   1000,
				MaxToolCalls:       100,
				MaxIterations:      50,
			},
		},
	}

	err := GuardrailCheck(exec)
	if err == nil {
		t.Fatal("expected guardrail error for budget exhaustion")
	}
	ge, ok := err.(*GuardrailError)
	if !ok {
		t.Fatalf("expected *GuardrailError, got %T", err)
	}
	if ge.Reason != "budget_exhausted" {
		t.Errorf("expected reason 'budget_exhausted', got %q", ge.Reason)
	}
}

func TestGuardrailCheck_ToolCallLimit(t *testing.T) {
	exec := &Execution{
		ID:        "test",
		StartTime: time.Now(),
		ToolCalls: 100,
		Definition: &Definition{
			Guardrails: Guardrails{
				MaxDurationMinutes: 60,
				AIAPIBudgetCents:   1000,
				MaxToolCalls:       100,
				MaxIterations:      50,
			},
		},
	}

	err := GuardrailCheck(exec)
	if err == nil {
		t.Fatal("expected guardrail error for tool call limit")
	}
}

func TestGuardrailCheck_IterationLimit(t *testing.T) {
	exec := &Execution{
		ID:         "test",
		StartTime:  time.Now(),
		Iterations: 50,
		Definition: &Definition{
			Guardrails: Guardrails{
				MaxDurationMinutes: 60,
				AIAPIBudgetCents:   1000,
				MaxToolCalls:       100,
				MaxIterations:      50,
			},
		},
	}

	err := GuardrailCheck(exec)
	if err == nil {
		t.Fatal("expected guardrail error for iteration limit")
	}
}

func TestGuardrailCheck_KillSwitch(t *testing.T) {
	exec := &Execution{
		ID:             "test",
		StartTime:      time.Now(),
		KillSwitchUsed: true,
		Definition: &Definition{
			Guardrails: DefaultGuardrails(),
		},
	}

	err := GuardrailCheck(exec)
	if err == nil {
		t.Fatal("expected guardrail error for kill switch")
	}
}

func TestCanExecuteTool_Allowed(t *testing.T) {
	exec := &Execution{
		StartTime: time.Now(),
		Definition: &Definition{
			Guardrails: DefaultGuardrails(),
		},
	}

	if !CanExecuteTool(exec, "web_search") {
		t.Error("expected web_search to be allowed")
	}
	if !CanExecuteTool(exec, "exec_command") {
		t.Error("expected exec_command to be allowed")
	}
}

func TestCanExecuteTool_ReadOnly(t *testing.T) {
	exec := &Execution{
		StartTime: time.Now(),
		Definition: &Definition{
			Guardrails: Guardrails{
				ReadOnly:           true,
				MaxDurationMinutes: 60,
				AIAPIBudgetCents:   1000,
				MaxToolCalls:       100,
				MaxIterations:      50,
			},
		},
	}

	if !CanExecuteTool(exec, "web_search") {
		t.Error("expected web_search to be allowed in read-only mode")
	}
	if !CanExecuteTool(exec, "read_file") {
		t.Error("expected read_file to be allowed in read-only mode")
	}
	if CanExecuteTool(exec, "write_file") {
		t.Error("expected write_file to be denied in read-only mode")
	}
	if CanExecuteTool(exec, "exec_command") {
		t.Error("expected exec_command to be denied in read-only mode")
	}
}

func TestCanExecuteTool_KillSwitch(t *testing.T) {
	exec := &Execution{
		StartTime:      time.Now(),
		KillSwitchUsed: true,
		Definition: &Definition{
			Guardrails: DefaultGuardrails(),
		},
	}

	if CanExecuteTool(exec, "web_search") {
		t.Error("expected all tools denied after kill switch")
	}
}

func TestCanExecuteTool_ToolCallLimit(t *testing.T) {
	exec := &Execution{
		StartTime: time.Now(),
		ToolCalls: 100,
		Definition: &Definition{
			Guardrails: Guardrails{
				MaxToolCalls:       100,
				MaxDurationMinutes: 60,
				AIAPIBudgetCents:   1000,
				MaxIterations:      50,
			},
		},
	}

	if CanExecuteTool(exec, "web_search") {
		t.Error("expected tool denied at tool call limit")
	}
}

func TestRecordToolCall(t *testing.T) {
	exec := &Execution{
		StartTime: time.Now(),
		Definition: &Definition{
			Guardrails: DefaultGuardrails(),
		},
	}

	RecordToolCall(exec, 10)
	if exec.ToolCalls != 1 {
		t.Errorf("tool calls: got %d, want 1", exec.ToolCalls)
	}
	if exec.SpentCents != 10 {
		t.Errorf("spent cents: got %d, want 10", exec.SpentCents)
	}

	RecordToolCall(exec, 25)
	if exec.ToolCalls != 2 {
		t.Errorf("tool calls: got %d, want 2", exec.ToolCalls)
	}
	if exec.SpentCents != 35 {
		t.Errorf("spent cents: got %d, want 35", exec.SpentCents)
	}
}

func TestRecordToolCall_Nil(t *testing.T) {
	// Should not panic
	RecordToolCall(nil, 10)
}

func TestDefaultGuardrails(t *testing.T) {
	g := DefaultGuardrails()
	if g.MaxDurationMinutes != 60 {
		t.Errorf("max duration: got %d, want 60", g.MaxDurationMinutes)
	}
	if !g.KillSwitch {
		t.Error("expected kill switch enabled by default")
	}
	if g.AIAPIBudgetCents != 1000 {
		t.Errorf("budget: got %d, want 1000", g.AIAPIBudgetCents)
	}
}

func TestGuardrailError_Error(t *testing.T) {
	err := &GuardrailError{Reason: "budget_exhausted", CampaignID: "test-1"}
	expected := "campaign test-1: guardrail violation: budget_exhausted"
	if err.Error() != expected {
		t.Errorf("error: got %q, want %q", err.Error(), expected)
	}
}
