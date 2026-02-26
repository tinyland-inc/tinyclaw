// Package adapters provides backend adapter interfaces for campaign dispatch.
//
// Each adapter implements the BackendAdapter interface, allowing campaigns
// to target different agent backends (PicoClaw, IronClaw, HexStrike-AI).
package adapters

import (
	"context"
	"fmt"

	"github.com/tinyland-inc/picoclaw/pkg/campaign"
)

// PicoClawAdapter dispatches campaign steps to the local PicoClaw agent loop.
type PicoClawAdapter struct {
	// ProcessFn is the function to call for processing a message.
	// This is wired to agent.AgentLoop.ProcessDirect at gateway startup.
	ProcessFn func(ctx context.Context, prompt, sessionKey string) (string, error)
}

// NewPicoClawAdapter creates a new adapter for the local PicoClaw agent.
func NewPicoClawAdapter() *PicoClawAdapter {
	return &PicoClawAdapter{}
}

func (a *PicoClawAdapter) Execute(ctx context.Context, agentID, prompt string, tools []string) (string, error) {
	if a.ProcessFn == nil {
		return "", fmt.Errorf("picoclaw adapter not initialized: ProcessFn is nil")
	}
	sessionKey := fmt.Sprintf("campaign:%s", agentID)
	return a.ProcessFn(ctx, prompt, sessionKey)
}

func (a *PicoClawAdapter) Name() string { return "picoclaw" }

// Ensure PicoClawAdapter implements BackendAdapter
var _ campaign.BackendAdapter = (*PicoClawAdapter)(nil)

// StubAdapter is a test/development adapter that returns canned responses.
type StubAdapter struct {
	BackendName string
	Response    string
}

func (a *StubAdapter) Execute(_ context.Context, agentID, prompt string, _ []string) (string, error) {
	if a.Response != "" {
		return a.Response, nil
	}
	return fmt.Sprintf("[%s/%s] processed: %s", a.BackendName, agentID, prompt[:min(50, len(prompt))]), nil
}

func (a *StubAdapter) Name() string { return a.BackendName }

var _ campaign.BackendAdapter = (*StubAdapter)(nil)
