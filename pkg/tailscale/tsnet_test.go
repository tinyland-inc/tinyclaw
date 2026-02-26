package tailscale

import (
	"context"
	"testing"
)

func TestNewServer_Defaults(t *testing.T) {
	s := NewServer(Config{})

	if s.config.Hostname != "picoclaw-gateway" {
		t.Errorf("hostname: got %q, want %q", s.config.Hostname, "picoclaw-gateway")
	}
	if s.config.StateDir == "" {
		t.Error("expected non-empty state dir")
	}
	if s.IsRunning() {
		t.Error("expected not running initially")
	}
}

func TestNewServer_CustomHostname(t *testing.T) {
	s := NewServer(Config{Hostname: "my-agent"})
	if s.config.Hostname != "my-agent" {
		t.Errorf("hostname: got %q, want %q", s.config.Hostname, "my-agent")
	}
}

func TestServer_StartDisabled(t *testing.T) {
	s := NewServer(Config{Enabled: false})
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start disabled: %v", err)
	}
	if s.IsRunning() {
		t.Error("expected not running when disabled")
	}
}

func TestServer_StartEnabled(t *testing.T) {
	s := NewServer(Config{
		Enabled:  true,
		StateDir: t.TempDir(),
	})
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start enabled: %v", err)
	}
	if !s.IsRunning() {
		t.Error("expected running after start")
	}
	s.Stop()
	if s.IsRunning() {
		t.Error("expected not running after stop")
	}
}

func TestServer_DoubleStart(t *testing.T) {
	s := NewServer(Config{
		Enabled:  true,
		StateDir: t.TempDir(),
	})
	s.Start(context.Background())
	defer s.Stop()

	err := s.Start(context.Background())
	if err == nil {
		t.Error("expected error on double start")
	}
}

func TestServer_HTTPClient(t *testing.T) {
	s := NewServer(Config{})
	client := s.HTTPClient()
	if client == nil {
		t.Error("expected non-nil http client")
	}
}

func TestSetecClient_Disabled(t *testing.T) {
	c := NewSetecClient(SetecConfig{Enabled: false})
	_, err := c.Get(context.Background(), "test")
	if err == nil {
		t.Error("expected error when disabled")
	}
}

func TestSetecClient_DefaultPrefix(t *testing.T) {
	c := NewSetecClient(SetecConfig{})
	if c.config.Prefix != "picoclaw/" {
		t.Errorf("prefix: got %q, want %q", c.config.Prefix, "picoclaw/")
	}
}

func TestSetecClient_StubGet(t *testing.T) {
	c := NewSetecClient(SetecConfig{
		Enabled: true,
		BaseURL: "http://setec.ts.net",
	})
	_, err := c.Get(context.Background(), "api-key")
	if err == nil {
		t.Error("expected error from stub implementation")
	}
}

func TestSetecClient_Invalidate(t *testing.T) {
	c := NewSetecClient(SetecConfig{Enabled: true})
	// Should not panic
	c.Invalidate("test-key")
}

func TestSetecClient_IsEnabled(t *testing.T) {
	c := NewSetecClient(SetecConfig{Enabled: true})
	if !c.IsEnabled() {
		t.Error("expected enabled")
	}
	c2 := NewSetecClient(SetecConfig{Enabled: false})
	if c2.IsEnabled() {
		t.Error("expected disabled")
	}
}
