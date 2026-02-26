package aperture

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient_Disabled(t *testing.T) {
	c, err := NewClient(Config{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.IsEnabled() {
		t.Error("expected client to be disabled")
	}
	if c.ProxyURL() != "" {
		t.Error("expected empty proxy URL when disabled")
	}
}

func TestNewClient_WithProxyURL(t *testing.T) {
	c, err := NewClient(Config{
		Enabled:  true,
		ProxyURL: "https://aperture.ts.net:8443",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.IsEnabled() {
		t.Error("expected client to be enabled")
	}
	if c.ProxyURL() != "https://aperture.ts.net:8443" {
		t.Errorf("proxy URL: got %q, want %q", c.ProxyURL(), "https://aperture.ts.net:8443")
	}
}

func TestNewClient_InvalidProxyURL(t *testing.T) {
	_, err := NewClient(Config{
		Enabled:  true,
		ProxyURL: "://invalid",
	})
	if err == nil {
		t.Error("expected error for invalid proxy URL")
	}
}

func TestProxyTransport_Disabled(t *testing.T) {
	c, _ := NewClient(Config{Enabled: false})
	transport := c.ProxyTransport()
	if transport != http.DefaultTransport {
		t.Error("expected default transport when disabled")
	}
}

func TestProxyTransport_Enabled(t *testing.T) {
	c, _ := NewClient(Config{
		Enabled:  true,
		ProxyURL: "https://aperture.ts.net:8443",
	})
	transport := c.ProxyTransport()
	if transport == http.DefaultTransport {
		t.Error("expected custom transport when enabled")
	}
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	c, _ := NewClient(Config{Enabled: true})
	handler := c.WebhookHandler()

	req := httptest.NewRequest(http.MethodGet, "/webhook/aperture", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestWebhookHandler_InvalidAuth(t *testing.T) {
	c, _ := NewClient(Config{
		Enabled:    true,
		WebhookKey: "secret-key",
	})
	handler := c.WebhookHandler()

	body := `{"request_id": "req-1"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/aperture", strings.NewReader(body))
	req.Header.Set("X-Webhook-Key", "wrong-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWebhookHandler_ValidEvent(t *testing.T) {
	c, _ := NewClient(Config{
		Enabled:    true,
		WebhookKey: "secret-key",
	})

	var received UsageEvent
	c.SetEventHandler(func(e UsageEvent) {
		received = e
	})

	handler := c.WebhookHandler()

	event := UsageEvent{
		RequestID:    "req-123",
		Model:        "claude-sonnet-4.6",
		InputTokens:  500,
		OutputTokens: 200,
		TotalTokens:  700,
		Duration:     1234.5,
		Timestamp:    time.Now(),
		Status:       200,
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/webhook/aperture", strings.NewReader(string(body)))
	req.Header.Set("X-Webhook-Key", "secret-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if received.RequestID != "req-123" {
		t.Errorf("expected request_id 'req-123', got %q", received.RequestID)
	}
	if received.TotalTokens != 700 {
		t.Errorf("expected 700 total tokens, got %d", received.TotalTokens)
	}
}

func TestMeterStore_Record(t *testing.T) {
	store := NewMeterStore()

	event := UsageEvent{
		RequestID:    "req-1",
		Model:        "gpt-4",
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		Duration:     500,
		Timestamp:    time.Now(),
		Status:       200,
	}

	store.Record("agent-1", "session-1", event)
	store.Record("agent-1", "session-1", event)
	store.Record("agent-1", "session-2", event)

	meter, ok := store.GetAgentMeter("agent-1")
	if !ok {
		t.Fatal("expected meter for agent-1")
	}

	if meter.TotalCalls != 3 {
		t.Errorf("total calls: got %d, want 3", meter.TotalCalls)
	}
	if meter.TotalTokens != 450 {
		t.Errorf("total tokens: got %d, want 450", meter.TotalTokens)
	}

	sess1, ok := meter.Sessions["session-1"]
	if !ok {
		t.Fatal("expected session-1")
	}
	if sess1.Calls != 2 {
		t.Errorf("session-1 calls: got %d, want 2", sess1.Calls)
	}

	sess2, ok := meter.Sessions["session-2"]
	if !ok {
		t.Fatal("expected session-2")
	}
	if sess2.Calls != 1 {
		t.Errorf("session-2 calls: got %d, want 1", sess2.Calls)
	}
}

func TestMeterStore_ErrorTracking(t *testing.T) {
	store := NewMeterStore()

	store.Record("agent-1", "s1", UsageEvent{Status: 200, TotalTokens: 100})
	store.Record("agent-1", "s1", UsageEvent{Status: 429, TotalTokens: 0})
	store.Record("agent-1", "s1", UsageEvent{Status: 500, TotalTokens: 0})

	meter, _ := store.GetAgentMeter("agent-1")
	if meter.Errors != 2 {
		t.Errorf("errors: got %d, want 2", meter.Errors)
	}
}

func TestMeterStore_MultiAgent(t *testing.T) {
	store := NewMeterStore()

	store.Record("agent-1", "s1", UsageEvent{TotalTokens: 100})
	store.Record("agent-2", "s1", UsageEvent{TotalTokens: 200})

	meters := store.GetAllMeters()
	if len(meters) != 2 {
		t.Errorf("expected 2 agents, got %d", len(meters))
	}
}
