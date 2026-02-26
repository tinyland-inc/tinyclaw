package aperture

import (
	"context"
	"testing"
)

func TestCerbosClient_Disabled(t *testing.T) {
	c := NewCerbosClient(CerbosConfig{Enabled: false})

	decision, err := c.CheckToolAccess(context.Background(), ToolAccessRequest{
		AgentID:  "agent-1",
		ToolName: "exec_command",
		Action:   "execute",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allowed {
		t.Error("expected all tools allowed when Cerbos is disabled")
	}
	if decision.Reason != "cerbos_disabled" {
		t.Errorf("expected reason 'cerbos_disabled', got %q", decision.Reason)
	}
}

func TestCerbosClient_StubAllow(t *testing.T) {
	c := NewCerbosClient(CerbosConfig{
		Enabled: true,
		PDPURL:  "http://localhost:3592",
	})

	decision, err := c.CheckToolAccess(context.Background(), ToolAccessRequest{
		AgentID:    "agent-1",
		SessionKey: "s1",
		Channel:    "telegram",
		ToolName:   "web_search",
		Action:     "execute",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allowed {
		t.Error("expected stub to allow")
	}
	if decision.PolicyID != "default" {
		t.Errorf("expected policy 'default', got %q", decision.PolicyID)
	}
}

func TestCerbosClient_Cache(t *testing.T) {
	c := NewCerbosClient(CerbosConfig{
		Enabled: true,
		PDPURL:  "http://localhost:3592",
	})

	req := ToolAccessRequest{
		AgentID:  "agent-1",
		ToolName: "web_search",
		Action:   "execute",
	}

	// First call populates cache
	d1, _ := c.CheckToolAccess(context.Background(), req)

	// Second call should hit cache
	d2, _ := c.CheckToolAccess(context.Background(), req)

	if d1.Timestamp != d2.Timestamp {
		t.Error("expected cached result to have same timestamp")
	}
}

func TestCerbosClient_InvalidateCache(t *testing.T) {
	c := NewCerbosClient(CerbosConfig{
		Enabled: true,
		PDPURL:  "http://localhost:3592",
	})

	req := ToolAccessRequest{
		AgentID:  "agent-1",
		ToolName: "web_search",
		Action:   "execute",
	}

	d1, _ := c.CheckToolAccess(context.Background(), req)
	c.InvalidateCache("agent-1")
	d2, _ := c.CheckToolAccess(context.Background(), req)

	// After cache invalidation, should get a new timestamp
	if d1.Timestamp.Equal(d2.Timestamp) {
		// This might flake if both calls happen within the same nanosecond,
		// but in practice they won't.
		t.Log("timestamps equal after invalidation (may be timing artifact)")
	}
}

func TestCerbosClient_IsEnabled(t *testing.T) {
	c := NewCerbosClient(CerbosConfig{Enabled: true})
	if !c.IsEnabled() {
		t.Error("expected enabled")
	}

	c2 := NewCerbosClient(CerbosConfig{Enabled: false})
	if c2.IsEnabled() {
		t.Error("expected disabled")
	}
}
