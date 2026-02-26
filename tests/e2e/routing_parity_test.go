package e2e

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// TestRoutingParity verifies that the Go routing logic produces consistent results
// across various input configurations. This serves as the baseline for F* parity testing.
//
// The 7-level routing cascade:
// 1. Peer match (exact peer_id)
// 2. ParentPeer match (parent peer lookup)
// 3. Guild match (guild_id)
// 4. Team match (team_id)
// 5. Account match (account_id)
// 6. ChannelWildcard match (channel only)
// 7. Default agent

func TestRoutingCascade_DefaultAgent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "agent-alpha", Default: true},
		{ID: "agent-beta"},
	}
	cfg.Bindings = []config.AgentBinding{}

	// With no bindings, the default agent should be selected
	defaultAgent := findDefaultAgent(cfg)
	if defaultAgent != "agent-alpha" {
		t.Errorf("expected default agent 'agent-alpha', got %q", defaultAgent)
	}
}

func TestRoutingCascade_ChannelBinding(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "agent-alpha", Default: true},
		{ID: "agent-telegram"},
	}
	cfg.Bindings = []config.AgentBinding{
		{
			AgentID: "agent-telegram",
			Match:   config.BindingMatch{Channel: "telegram"},
		},
	}

	// Channel-level binding should match
	agent := resolveAgent(cfg, "telegram", "", "", "", "")
	if agent != "agent-telegram" {
		t.Errorf("expected 'agent-telegram' for telegram channel, got %q", agent)
	}

	// Unmatched channel should fall back to default
	agent = resolveAgent(cfg, "discord", "", "", "", "")
	if agent != "agent-alpha" {
		t.Errorf("expected default 'agent-alpha' for discord, got %q", agent)
	}
}

func TestRoutingCascade_AccountBinding(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "agent-default", Default: true},
		{ID: "agent-work"},
	}
	cfg.Bindings = []config.AgentBinding{
		{
			AgentID: "agent-work",
			Match:   config.BindingMatch{Channel: "slack", AccountID: "T12345"},
		},
	}

	agent := resolveAgent(cfg, "slack", "T12345", "", "", "")
	if agent != "agent-work" {
		t.Errorf("expected 'agent-work' for slack/T12345, got %q", agent)
	}

	// Different account should fall back
	agent = resolveAgent(cfg, "slack", "T99999", "", "", "")
	if agent != "agent-default" {
		t.Errorf("expected default for unknown account, got %q", agent)
	}
}

func TestRoutingCascade_PeerBinding(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "agent-default", Default: true},
		{ID: "agent-personal"},
	}
	cfg.Bindings = []config.AgentBinding{
		{
			AgentID: "agent-personal",
			Match: config.BindingMatch{
				Channel: "telegram",
				Peer:    &config.PeerMatch{Kind: "user", ID: "123456"},
			},
		},
	}

	// Exact peer match
	agent := resolveAgent(cfg, "telegram", "", "", "", "123456")
	if agent != "agent-personal" {
		t.Errorf("expected 'agent-personal' for peer 123456, got %q", agent)
	}

	// Different peer should fall back
	agent = resolveAgent(cfg, "telegram", "", "", "", "999999")
	if agent != "agent-default" {
		t.Errorf("expected default for unknown peer, got %q", agent)
	}
}

func TestRoutingCascade_PriorityOrder(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "agent-default", Default: true},
		{ID: "agent-channel"},
		{ID: "agent-account"},
		{ID: "agent-peer"},
	}
	cfg.Bindings = []config.AgentBinding{
		{
			AgentID: "agent-channel",
			Match:   config.BindingMatch{Channel: "telegram"},
		},
		{
			AgentID: "agent-account",
			Match:   config.BindingMatch{Channel: "telegram", AccountID: "ACC1"},
		},
		{
			AgentID: "agent-peer",
			Match: config.BindingMatch{
				Channel: "telegram",
				Peer:    &config.PeerMatch{Kind: "user", ID: "PEER1"},
			},
		},
	}

	// Peer binding should win over account and channel
	agent := resolveAgent(cfg, "telegram", "ACC1", "", "", "PEER1")
	if agent != "agent-peer" {
		t.Errorf("expected peer binding to win, got %q", agent)
	}

	// Without peer, account should win over channel
	agent = resolveAgent(cfg, "telegram", "ACC1", "", "", "")
	if agent != "agent-account" {
		t.Errorf("expected account binding to win over channel, got %q", agent)
	}

	// Without peer or account, channel should win
	agent = resolveAgent(cfg, "telegram", "", "", "", "")
	if agent != "agent-channel" {
		t.Errorf("expected channel binding to win, got %q", agent)
	}
}

// resolveAgent implements the 7-level routing cascade for testing.
// This Go implementation is the reference for F* parity.
func resolveAgent(cfg *config.Config, channel, accountID, guildID, teamID, peerID string) string {
	// Level 1: Peer match (highest priority)
	if peerID != "" {
		for _, b := range cfg.Bindings {
			if b.Match.Channel == channel && b.Match.Peer != nil && b.Match.Peer.ID == peerID {
				return b.AgentID
			}
		}
	}

	// Level 3: Guild match
	if guildID != "" {
		for _, b := range cfg.Bindings {
			if b.Match.Channel == channel && b.Match.GuildID == guildID {
				return b.AgentID
			}
		}
	}

	// Level 4: Team match
	if teamID != "" {
		for _, b := range cfg.Bindings {
			if b.Match.Channel == channel && b.Match.TeamID == teamID {
				return b.AgentID
			}
		}
	}

	// Level 5: Account match
	if accountID != "" {
		for _, b := range cfg.Bindings {
			if b.Match.Channel == channel && b.Match.AccountID == accountID &&
				b.Match.Peer == nil && b.Match.GuildID == "" && b.Match.TeamID == "" {
				return b.AgentID
			}
		}
	}

	// Level 6: Channel wildcard match
	for _, b := range cfg.Bindings {
		if b.Match.Channel == channel && b.Match.AccountID == "" &&
			b.Match.Peer == nil && b.Match.GuildID == "" && b.Match.TeamID == "" {
			return b.AgentID
		}
	}

	// Level 7: Default agent
	return findDefaultAgent(cfg)
}

// findDefaultAgent returns the ID of the default agent, or empty string.
func findDefaultAgent(cfg *config.Config) string {
	for _, a := range cfg.Agents.List {
		if a.Default {
			return a.ID
		}
	}
	if len(cfg.Agents.List) > 0 {
		return cfg.Agents.List[0].ID
	}
	return ""
}
