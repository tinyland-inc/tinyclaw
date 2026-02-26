// Package aperture provides Tailscale Aperture integration for PicoClaw.
//
// Aperture acts as a proxy for LLM API calls, providing:
// - Centralized API key management (no keys in picoclaw config)
// - Per-request token usage metering via webhooks
// - Request ID correlation with the F* core audit log
package aperture

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tinyland-inc/picoclaw/pkg/logger"
)

// Config holds Aperture proxy configuration.
type Config struct {
	Enabled    bool   `json:"enabled"`
	ProxyURL   string `json:"proxy_url"`   // Aperture proxy endpoint
	WebhookURL string `json:"webhook_url"` // Webhook endpoint for metering events
	WebhookKey string `json:"webhook_key"` // Shared key for webhook authentication
}

// UsageEvent represents a token usage event received from Aperture.
type UsageEvent struct {
	RequestID    string    `json:"request_id"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	Duration     float64   `json:"duration_ms"`
	Timestamp    time.Time `json:"timestamp"`
	Status       int       `json:"status"`
}

// Client provides Aperture proxy and metering integration.
type Client struct {
	config       Config
	proxyURL     *url.URL
	httpClient   *http.Client
	eventHandler func(UsageEvent)
	mu           sync.RWMutex
}

// NewClient creates a new Aperture client.
func NewClient(cfg Config) (*Client, error) {
	c := &Client{
		config:     cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	if cfg.ProxyURL != "" {
		parsed, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid aperture proxy URL: %w", err)
		}
		c.proxyURL = parsed
	}

	return c, nil
}

// ProxyTransport returns an http.RoundTripper that routes requests through
// Aperture. This can be used as the Transport for any http.Client to
// transparently proxy LLM API calls.
func (c *Client) ProxyTransport() http.RoundTripper {
	if !c.config.Enabled || c.proxyURL == nil {
		return http.DefaultTransport
	}
	return &apertureTransport{
		proxyURL: c.proxyURL,
		inner:    http.DefaultTransport,
	}
}

// ProxyURL returns the configured Aperture proxy URL for use in provider
// configuration. Returns empty string if Aperture is not enabled.
func (c *Client) ProxyURL() string {
	if !c.config.Enabled {
		return ""
	}
	return c.config.ProxyURL
}

// SetEventHandler registers a callback for Aperture usage events.
func (c *Client) SetEventHandler(handler func(UsageEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventHandler = handler
}

// WebhookHandler returns an http.Handler for receiving Aperture webhook events.
// Mount this at the configured webhook path in the gateway.
func (c *Client) WebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify webhook key if configured
		if c.config.WebhookKey != "" {
			if r.Header.Get("X-Webhook-Key") != c.config.WebhookKey {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		var event UsageEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		c.mu.RLock()
		handler := c.eventHandler
		c.mu.RUnlock()

		if handler != nil {
			handler(event)
		}

		logger.InfoCF("aperture", "Usage event received", map[string]any{
			"request_id":   event.RequestID,
			"model":        event.Model,
			"total_tokens": event.TotalTokens,
			"duration_ms":  event.Duration,
		})

		w.WriteHeader(http.StatusOK)
	})
}

// IsEnabled returns whether Aperture integration is active.
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// apertureTransport is an http.RoundTripper that proxies requests through Aperture.
type apertureTransport struct {
	proxyURL *url.URL
	inner    http.RoundTripper
}

func (t *apertureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original
	proxyReq := req.Clone(req.Context())

	// Set the original URL as a header for Aperture to route
	proxyReq.Header.Set("X-Aperture-Target", req.URL.String())

	// Rewrite the URL to point to Aperture
	proxyReq.URL.Scheme = t.proxyURL.Scheme
	proxyReq.URL.Host = t.proxyURL.Host
	proxyReq.Host = t.proxyURL.Host

	return t.inner.RoundTrip(proxyReq)
}

// MeterStore provides per-agent, per-session metrics aggregation from
// Aperture usage events. This mirrors the remote-juggler gateway/metering.go
// pattern.
type MeterStore struct {
	mu     sync.RWMutex
	meters map[string]*AgentMeter
}

// AgentMeter tracks per-agent usage metrics.
type AgentMeter struct {
	AgentID      string
	TotalCalls   int64
	TotalTokens  int64
	TotalLatency float64
	Errors       int64
	Sessions     map[string]*SessionMeter
}

// SessionMeter tracks per-session usage metrics.
type SessionMeter struct {
	SessionKey   string
	Calls        int64
	InputTokens  int64
	OutputTokens int64
	ToolCalls    int64
	Duration     float64
	LastActivity time.Time
}

// NewMeterStore creates a new metering store.
func NewMeterStore() *MeterStore {
	return &MeterStore{
		meters: make(map[string]*AgentMeter),
	}
}

// Record adds a usage event to the meter store.
func (s *MeterStore) Record(agentID, sessionKey string, event UsageEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	meter, ok := s.meters[agentID]
	if !ok {
		meter = &AgentMeter{
			AgentID:  agentID,
			Sessions: make(map[string]*SessionMeter),
		}
		s.meters[agentID] = meter
	}

	meter.TotalCalls++
	meter.TotalTokens += int64(event.TotalTokens)
	meter.TotalLatency += event.Duration
	if event.Status >= 400 {
		meter.Errors++
	}

	sess, ok := meter.Sessions[sessionKey]
	if !ok {
		sess = &SessionMeter{SessionKey: sessionKey}
		meter.Sessions[sessionKey] = sess
	}

	sess.Calls++
	sess.InputTokens += int64(event.InputTokens)
	sess.OutputTokens += int64(event.OutputTokens)
	sess.Duration += event.Duration
	sess.LastActivity = event.Timestamp
}

// GetAgentMeter returns metrics for a specific agent.
func (s *MeterStore) GetAgentMeter(agentID string) (*AgentMeter, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.meters[agentID]
	return m, ok
}

// GetAllMeters returns a snapshot of all agent meters.
func (s *MeterStore) GetAllMeters() map[string]*AgentMeter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*AgentMeter, len(s.meters))
	maps.Copy(result, s.meters)
	return result
}

// HandleEvent processes an Aperture usage event into the meter store.
// This is intended to be used as the event handler for Client.SetEventHandler.
func (s *MeterStore) HandleEvent(ctx context.Context, agentID, sessionKey string) func(UsageEvent) {
	return func(event UsageEvent) {
		s.Record(agentID, sessionKey, event)
	}
}
