package campaign

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// testAdapter is a simple backend adapter for testing.
type testAdapter struct {
	name      string
	responses map[string]string
	delay     time.Duration
	failAfter int
	callCount int
}

func (a *testAdapter) Execute(_ context.Context, agentID, prompt string, _ []string) (string, error) {
	a.callCount++
	if a.failAfter > 0 && a.callCount > a.failAfter {
		return "", fmt.Errorf("adapter failure after %d calls", a.failAfter)
	}
	if a.delay > 0 {
		time.Sleep(a.delay)
	}
	if resp, ok := a.responses[prompt]; ok {
		return resp, nil
	}
	return fmt.Sprintf("processed by %s", a.name), nil
}

func (a *testAdapter) Name() string { return a.name }

func TestRunner_StartAndComplete(t *testing.T) {
	runner := NewRunner()
	adapter := &testAdapter{name: "picoclaw"}
	runner.RegisterAdapter("picoclaw", adapter)

	def := &Definition{
		ID:   "test-1",
		Name: "Test Campaign",
		Targets: []Target{
			{AgentID: "agent-1", Backend: "picoclaw"},
		},
		Steps: []Step{
			{Name: "step-1", Prompt: "do something", Tools: []string{"web_search"}, TimeoutMinutes: 5},
		},
		Guardrails: DefaultGuardrails(),
		Feedback:   FeedbackNone,
	}

	exec, err := runner.Start(context.Background(), def)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if exec.Status != StatusRunning {
		t.Errorf("expected running, got %s", exec.Status)
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	exec, err = runner.GetStatus("test-1")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	if len(exec.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(exec.Results))
	}
}

func TestRunner_MultiStep(t *testing.T) {
	runner := NewRunner()
	adapter := &testAdapter{name: "picoclaw"}
	runner.RegisterAdapter("picoclaw", adapter)

	def := &Definition{
		ID:   "multi-step",
		Name: "Multi-Step",
		Targets: []Target{
			{AgentID: "agent-1", Backend: "picoclaw"},
		},
		Steps: []Step{
			{Name: "step-1", Prompt: "first", TimeoutMinutes: 5},
			{Name: "step-2", Prompt: "second", TimeoutMinutes: 5},
			{Name: "step-3", Prompt: "third", TimeoutMinutes: 5},
		},
		Guardrails: DefaultGuardrails(),
		Feedback:   FeedbackNone,
	}

	runner.Start(context.Background(), def)
	time.Sleep(200 * time.Millisecond)

	exec, _ := runner.GetStatus("multi-step")
	if exec.Status != StatusCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	if len(exec.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(exec.Results))
	}
}

func TestRunner_KillSwitch(t *testing.T) {
	runner := NewRunner()
	adapter := &testAdapter{name: "picoclaw", delay: 500 * time.Millisecond}
	runner.RegisterAdapter("picoclaw", adapter)

	def := &Definition{
		ID:   "killable",
		Name: "Killable",
		Targets: []Target{
			{AgentID: "agent-1", Backend: "picoclaw"},
		},
		Steps: []Step{
			{Name: "slow-step", Prompt: "slow", TimeoutMinutes: 5},
			{Name: "never-reached", Prompt: "never", TimeoutMinutes: 5},
		},
		Guardrails: DefaultGuardrails(),
		Feedback:   FeedbackNone,
	}

	runner.Start(context.Background(), def)
	time.Sleep(50 * time.Millisecond)

	err := runner.Stop("killable")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	time.Sleep(600 * time.Millisecond)

	exec, _ := runner.GetStatus("killable")
	if exec.Status != StatusCancelled {
		t.Errorf("expected cancelled, got %s", exec.Status)
	}
}

func TestRunner_DuplicateID(t *testing.T) {
	runner := NewRunner()
	adapter := &testAdapter{name: "picoclaw", delay: 200 * time.Millisecond}
	runner.RegisterAdapter("picoclaw", adapter)

	def := &Definition{
		ID:   "dup",
		Name: "Dup",
		Targets: []Target{
			{AgentID: "agent-1", Backend: "picoclaw"},
		},
		Steps: []Step{
			{Name: "step", Prompt: "x", TimeoutMinutes: 5},
		},
		Guardrails: DefaultGuardrails(),
		Feedback:   FeedbackNone,
	}

	_, err := runner.Start(context.Background(), def)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}

	_, err = runner.Start(context.Background(), def)
	if err == nil {
		t.Error("expected error for duplicate campaign ID")
	}
}

func TestRunner_NilDefinition(t *testing.T) {
	runner := NewRunner()
	_, err := runner.Start(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil definition")
	}
}

func TestRunner_NoSteps(t *testing.T) {
	runner := NewRunner()
	_, err := runner.Start(context.Background(), &Definition{
		ID:         "no-steps",
		Name:       "Empty",
		Targets:    []Target{{AgentID: "a", Backend: "picoclaw"}},
		Steps:      []Step{},
		Guardrails: DefaultGuardrails(),
	})
	if err == nil {
		t.Error("expected error for campaign with no steps")
	}
}

func TestRunner_MissingAdapter(t *testing.T) {
	runner := NewRunner()
	// No adapter registered

	def := &Definition{
		ID:   "no-adapter",
		Name: "No Adapter",
		Targets: []Target{
			{AgentID: "agent-1", Backend: "unknown"},
		},
		Steps: []Step{
			{Name: "step", Prompt: "test", TimeoutMinutes: 5},
		},
		Guardrails: DefaultGuardrails(),
		Feedback:   FeedbackNone,
	}

	runner.Start(context.Background(), def)
	time.Sleep(100 * time.Millisecond)

	exec, _ := runner.GetStatus("no-adapter")
	// Should complete with an error result
	if exec.Status != StatusCompleted {
		t.Errorf("expected completed (with error in result), got %s", exec.Status)
	}
	if len(exec.Results) == 0 {
		t.Fatal("expected at least one result")
	}
	if exec.Results[0].Error == "" {
		t.Error("expected error in step result for missing adapter")
	}
}

func TestRunner_ListExecutions(t *testing.T) {
	runner := NewRunner()
	adapter := &testAdapter{name: "picoclaw"}
	runner.RegisterAdapter("picoclaw", adapter)

	for i := 0; i < 3; i++ {
		def := &Definition{
			ID:   fmt.Sprintf("list-%d", i),
			Name: fmt.Sprintf("Campaign %d", i),
			Targets: []Target{
				{AgentID: "agent-1", Backend: "picoclaw"},
			},
			Steps: []Step{
				{Name: "step", Prompt: "test", TimeoutMinutes: 5},
			},
			Guardrails: DefaultGuardrails(),
			Feedback:   FeedbackNone,
		}
		runner.Start(context.Background(), def)
	}

	time.Sleep(200 * time.Millisecond)

	execs := runner.ListExecutions()
	if len(execs) != 3 {
		t.Errorf("expected 3 executions, got %d", len(execs))
	}
}
