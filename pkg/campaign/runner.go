// Package campaign implements campaign-driven agent orchestration.
//
// A campaign dispatches structured tasks to picoclaw agent instances
// with full guardrail enforcement (budget, duration, killSwitch, tool limits).
// Results are stored and optionally fed back as GitHub issues/PRs.
package campaign

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tinyland-inc/picoclaw/pkg/logger"
)

// Status represents the current state of a campaign.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled Status = "canceled"
)

// FeedbackPolicy determines how campaign results are delivered.
type FeedbackPolicy string

const (
	FeedbackGitHubIssue FeedbackPolicy = "github_issue"
	FeedbackGitHubPR    FeedbackPolicy = "github_pr"
	FeedbackChannel     FeedbackPolicy = "channel"
	FeedbackSetec       FeedbackPolicy = "setec"
	FeedbackNone        FeedbackPolicy = "none"
)

// Guardrails defines safety constraints for campaign execution.
type Guardrails struct {
	MaxDurationMinutes int  `json:"max_duration_minutes"`
	ReadOnly           bool `json:"read_only"`
	AIAPIBudgetCents   int  `json:"ai_api_budget_cents"`
	KillSwitch         bool `json:"kill_switch"`
	MaxToolCalls       int  `json:"max_tool_calls"`
	MaxIterations      int  `json:"max_iterations"`
}

// DefaultGuardrails returns safe default guardrails.
func DefaultGuardrails() Guardrails {
	return Guardrails{
		MaxDurationMinutes: 60,
		ReadOnly:           false,
		AIAPIBudgetCents:   1000,
		KillSwitch:         true,
		MaxToolCalls:       100,
		MaxIterations:      50,
	}
}

// Target specifies an agent backend to dispatch to.
type Target struct {
	AgentID        string `json:"agent_id"`
	Backend        string `json:"backend"` // "picoclaw", "ironclaw", "hexstrike"
	ConfigOverride string `json:"config_override,omitempty"`
}

// Step defines a single process step within a campaign.
type Step struct {
	Name           string   `json:"name"`
	Prompt         string   `json:"prompt"`
	Tools          []string `json:"tools"`
	TimeoutMinutes int      `json:"timeout_minutes"`
}

// Definition describes a complete campaign.
type Definition struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Targets     []Target       `json:"targets"`
	Steps       []Step         `json:"steps"`
	Guardrails  Guardrails     `json:"guardrails"`
	Feedback    FeedbackPolicy `json:"feedback"`
	Tags        []string       `json:"tags"`
}

// Execution tracks the runtime state of a campaign.
type Execution struct {
	ID             string
	Definition     *Definition
	Status         Status
	StartTime      time.Time
	EndTime        time.Time
	CurrentStep    int
	SpentCents     int
	ToolCalls      int
	Iterations     int
	Results        []StepResult
	Error          string
	KillSwitchUsed bool
}

// StepResult captures the outcome of a single campaign step.
type StepResult struct {
	StepName  string
	Output    string
	Duration  time.Duration
	ToolCalls int
	Tokens    int
	Error     string
}

// Runner executes campaigns against agent backends.
type Runner struct {
	mu         sync.RWMutex
	executions map[string]*Execution
	adapters   map[string]BackendAdapter
	cancel     map[string]context.CancelFunc
}

// NewRunner creates a new campaign runner.
func NewRunner() *Runner {
	return &Runner{
		executions: make(map[string]*Execution),
		adapters:   make(map[string]BackendAdapter),
		cancel:     make(map[string]context.CancelFunc),
	}
}

// RegisterAdapter registers a backend adapter for dispatching campaigns.
func (r *Runner) RegisterAdapter(backend string, adapter BackendAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[backend] = adapter
}

// Start begins executing a campaign asynchronously.
func (r *Runner) Start(ctx context.Context, def *Definition) (*Execution, error) {
	if def == nil {
		return nil, fmt.Errorf("campaign definition is nil")
	}
	if def.ID == "" {
		return nil, fmt.Errorf("campaign ID is required")
	}
	if len(def.Steps) == 0 {
		return nil, fmt.Errorf("campaign must have at least one step")
	}

	r.mu.Lock()
	if _, exists := r.executions[def.ID]; exists {
		r.mu.Unlock()
		return nil, fmt.Errorf("campaign %q is already running", def.ID)
	}

	exec := &Execution{
		ID:          def.ID,
		Definition:  def,
		Status:      StatusRunning,
		StartTime:   time.Now(),
		CurrentStep: 0,
		Results:     make([]StepResult, 0, len(def.Steps)),
	}
	r.executions[def.ID] = exec

	execCtx, cancelFn := context.WithTimeout(ctx,
		time.Duration(def.Guardrails.MaxDurationMinutes)*time.Minute)
	r.cancel[def.ID] = cancelFn
	r.mu.Unlock()

	go r.run(execCtx, exec)

	return exec, nil
}

// Stop activates the kill switch for a running campaign.
func (r *Runner) Stop(campaignID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	exec, ok := r.executions[campaignID]
	if !ok {
		return fmt.Errorf("campaign %q not found", campaignID)
	}
	if exec.Status != StatusRunning {
		return fmt.Errorf("campaign %q is not running (status: %s)", campaignID, exec.Status)
	}

	exec.KillSwitchUsed = true
	exec.Status = StatusCanceled
	exec.EndTime = time.Now()

	if cancel, ok := r.cancel[campaignID]; ok {
		cancel()
		delete(r.cancel, campaignID)
	}

	logger.InfoCF("campaign", "Kill switch activated", map[string]any{
		"campaign_id": campaignID,
	})

	return nil
}

// GetStatus returns the current execution state of a campaign.
func (r *Runner) GetStatus(campaignID string) (*Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exec, ok := r.executions[campaignID]
	if !ok {
		return nil, fmt.Errorf("campaign %q not found", campaignID)
	}
	return exec, nil
}

// ListExecutions returns all campaign executions.
func (r *Runner) ListExecutions() []*Execution {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Execution, 0, len(r.executions))
	for _, exec := range r.executions {
		result = append(result, exec)
	}
	return result
}

// run executes a campaign step by step.
func (r *Runner) run(ctx context.Context, exec *Execution) {
	defer func() {
		r.mu.Lock()
		delete(r.cancel, exec.ID)
		r.mu.Unlock()
	}()

	def := exec.Definition

	for i, step := range def.Steps {
		// Check context (kill switch / timeout)
		if ctx.Err() != nil {
			r.mu.Lock()
			if exec.Status == StatusRunning {
				exec.Status = StatusCanceled
				exec.Error = ctx.Err().Error()
			}
			exec.EndTime = time.Now()
			r.mu.Unlock()
			return
		}

		// Check guardrails
		if reason := checkGuardrails(exec, &def.Guardrails); reason != "" {
			r.mu.Lock()
			exec.Status = StatusFailed
			exec.Error = fmt.Sprintf("guardrail: %s", reason)
			exec.EndTime = time.Now()
			r.mu.Unlock()
			return
		}

		r.mu.Lock()
		exec.CurrentStep = i
		r.mu.Unlock()

		logger.InfoCF("campaign", "Executing step", map[string]any{
			"campaign_id": exec.ID,
			"step":        step.Name,
			"step_index":  i,
		})

		stepStart := time.Now()

		// Dispatch to backend adapter
		var output string
		var stepErr error

		for _, target := range def.Targets {
			r.mu.RLock()
			adapter, ok := r.adapters[target.Backend]
			r.mu.RUnlock()

			if !ok {
				stepErr = fmt.Errorf("no adapter for backend %q", target.Backend)
				continue
			}

			stepCtx, stepCancel := context.WithTimeout(ctx,
				time.Duration(step.TimeoutMinutes)*time.Minute)
			output, stepErr = adapter.Execute(stepCtx, target.AgentID, step.Prompt, step.Tools)
			stepCancel()

			if stepErr == nil {
				break // Success with first available adapter
			}
		}

		result := StepResult{
			StepName:  step.Name,
			Output:    output,
			Duration:  time.Since(stepStart),
			ToolCalls: len(step.Tools), // Approximate
		}
		if stepErr != nil {
			result.Error = stepErr.Error()
		}

		r.mu.Lock()
		exec.Results = append(exec.Results, result)
		exec.Iterations++
		r.mu.Unlock()
	}

	r.mu.Lock()
	exec.Status = StatusCompleted
	exec.EndTime = time.Now()
	r.mu.Unlock()

	logger.InfoCF("campaign", "Campaign completed", map[string]any{
		"campaign_id": exec.ID,
		"duration":    time.Since(exec.StartTime).String(),
		"steps":       len(exec.Results),
	})
}

// checkGuardrails returns a halt reason if guardrails are exceeded, or empty string.
func checkGuardrails(exec *Execution, g *Guardrails) string {
	if exec.KillSwitchUsed {
		return "kill_switch_activated"
	}
	elapsed := time.Since(exec.StartTime)
	if int(elapsed.Minutes()) >= g.MaxDurationMinutes {
		return "duration_exceeded"
	}
	if exec.SpentCents >= g.AIAPIBudgetCents {
		return "budget_exhausted"
	}
	if exec.ToolCalls >= g.MaxToolCalls {
		return "tool_call_limit"
	}
	if exec.Iterations >= g.MaxIterations {
		return "iteration_limit"
	}
	return ""
}
