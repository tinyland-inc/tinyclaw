package tailscale

import (
	"context"
	"fmt"
	"sync"

	"github.com/tinyland-inc/picoclaw/pkg/logger"
)

// SetecConfig holds Tailscale Setec secret management configuration.
type SetecConfig struct {
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"base_url"` // Setec server URL on the tailnet
	Prefix  string `json:"prefix"`   // Secret name prefix (default: picoclaw/)
}

// SetecClient provides access to secrets stored in Tailscale Setec.
// Secrets are stored/retrieved over the tailnet, so no credentials leave the
// secure network boundary.
type SetecClient struct {
	config SetecConfig
	cache  map[string]string
	mu     sync.RWMutex
}

// NewSetecClient creates a new Setec client.
func NewSetecClient(cfg SetecConfig) *SetecClient {
	if cfg.Prefix == "" {
		cfg.Prefix = "picoclaw/"
	}
	return &SetecClient{
		config: cfg,
		cache:  make(map[string]string),
	}
}

// Get retrieves a secret by name from Setec.
// The name is automatically prefixed with the configured prefix.
func (c *SetecClient) Get(ctx context.Context, name string) (string, error) {
	if !c.config.Enabled {
		return "", fmt.Errorf("setec not enabled")
	}

	fullName := c.config.Prefix + name

	// Check cache first
	c.mu.RLock()
	if val, ok := c.cache[fullName]; ok {
		c.mu.RUnlock()
		return val, nil
	}
	c.mu.RUnlock()

	// Placeholder: when Setec client is linked, this would be:
	//   client := setec.NewClient(c.config.BaseURL)
	//   secret, err := client.Get(ctx, fullName)
	//   if err != nil { return "", fmt.Errorf("setec get %q: %w", fullName, err) }
	//   value := string(secret)
	//
	// For now, return an error indicating Setec is not yet connected.
	logger.InfoCF("setec", "Secret lookup", map[string]any{"name": fullName})
	return "", fmt.Errorf(
		"setec client not connected: secret %q unavailable (requires tailscale.com/setec dependency)",
		fullName,
	)
}

// Put stores a secret in Setec.
func (c *SetecClient) Put(ctx context.Context, name, value string) error {
	if !c.config.Enabled {
		return fmt.Errorf("setec not enabled")
	}

	fullName := c.config.Prefix + name

	// Placeholder for Setec write
	logger.InfoCF("setec", "Secret store", map[string]any{"name": fullName})

	// Cache locally
	c.mu.Lock()
	c.cache[fullName] = value
	c.mu.Unlock()

	return fmt.Errorf("setec client not connected: cannot store %q (requires tailscale.com/setec dependency)", fullName)
}

// Invalidate removes a cached secret, forcing a fresh fetch on next Get.
func (c *SetecClient) Invalidate(name string) {
	fullName := c.config.Prefix + name
	c.mu.Lock()
	delete(c.cache, fullName)
	c.mu.Unlock()
}

// IsEnabled returns whether Setec integration is configured.
func (c *SetecClient) IsEnabled() bool {
	return c.config.Enabled
}
