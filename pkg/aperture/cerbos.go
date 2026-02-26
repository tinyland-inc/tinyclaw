package aperture

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// CerbosConfig holds Cerbos PDP configuration for tool-level authorization.
type CerbosConfig struct {
	Enabled bool   `json:"enabled"`
	PDPURL  string `json:"pdp_url"` // Cerbos PDP endpoint URL
}

// CerbosDecision represents the result of a Cerbos policy check.
type CerbosDecision struct {
	Allowed   bool
	Reason    string
	PolicyID  string
	Timestamp time.Time
}

// CerbosClient provides tool-call-level authorization via Cerbos PDP.
// Tool calls are intercepted and evaluated against declarative policies
// before execution, with all decisions logged to the audit trail.
type CerbosClient struct {
	config CerbosConfig
	cache  map[string]*CerbosDecision
	mu     sync.RWMutex
}

// NewCerbosClient creates a new Cerbos PDP client.
func NewCerbosClient(cfg CerbosConfig) *CerbosClient {
	return &CerbosClient{
		config: cfg,
		cache:  make(map[string]*CerbosDecision),
	}
}

// CheckToolAccess evaluates whether an agent is authorized to execute a tool.
// The decision is based on the Cerbos policy for the given resource/action pair.
func (c *CerbosClient) CheckToolAccess(ctx context.Context, req ToolAccessRequest) (*CerbosDecision, error) {
	if !c.config.Enabled {
		// When Cerbos is not enabled, allow all tool calls
		return &CerbosDecision{
			Allowed:   true,
			Reason:    "cerbos_disabled",
			Timestamp: time.Now(),
		}, nil
	}

	// Build cache key
	cacheKey := fmt.Sprintf("%s:%s:%s", req.AgentID, req.ToolName, req.Action)

	// Check cache
	c.mu.RLock()
	if cached, ok := c.cache[cacheKey]; ok {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	// Placeholder: when Cerbos client is linked, this would be:
	//   principal := cerbos.NewPrincipal(req.AgentID).
	//       WithRoles("agent").
	//       WithAttributes(map[string]any{
	//           "session_key": req.SessionKey,
	//           "channel":     req.Channel,
	//       })
	//   resource := cerbos.NewResource("tool", req.ToolName).
	//       WithAttributes(map[string]any{"action": req.Action})
	//   resp, err := c.client.CheckResourceSet(ctx, principal, resource, req.Action)
	//   allowed := resp.IsAllowed(req.Action)

	decision := &CerbosDecision{
		Allowed:   true,
		Reason:    "cerbos_stub_allow",
		PolicyID:  "default",
		Timestamp: time.Now(),
	}

	logger.InfoCF("cerbos", "Tool access check", map[string]any{
		"agent_id":  req.AgentID,
		"tool":      req.ToolName,
		"action":    req.Action,
		"allowed":   decision.Allowed,
		"policy_id": decision.PolicyID,
	})

	// Cache the decision
	c.mu.Lock()
	c.cache[cacheKey] = decision
	c.mu.Unlock()

	return decision, nil
}

// InvalidateCache clears the authorization cache for a specific agent.
func (c *CerbosClient) InvalidateCache(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.cache {
		if len(key) > len(agentID) && key[:len(agentID)+1] == agentID+":" {
			delete(c.cache, key)
		}
	}
}

// IsEnabled returns whether Cerbos integration is active.
func (c *CerbosClient) IsEnabled() bool {
	return c.config.Enabled
}

// ToolAccessRequest describes a tool access authorization check.
type ToolAccessRequest struct {
	AgentID    string
	SessionKey string
	Channel    string
	ToolName   string
	Action     string // e.g., "execute", "read", "write"
}
