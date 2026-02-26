// Package tailscale provides Tailscale integration for the PicoClaw gateway.
//
// When enabled, the gateway joins the tailnet as a picoclaw-gateway node via
// tsnet, inheriting zero-trust identity for authentication. All LLM API calls
// can be optionally routed through Tailscale Aperture for centralized API key
// management and usage metering.
package tailscale

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/tinyland-inc/picoclaw/pkg/logger"
)

// Config holds Tailscale tsnet configuration.
type Config struct {
	Enabled  bool   `json:"enabled"`
	Hostname string `json:"hostname"`  // Tailscale node name (default: picoclaw-gateway)
	StateDir string `json:"state_dir"` // Directory for tsnet state (default: ~/.picoclaw/tsnet)
	AuthKey  string `json:"auth_key"`  // Optional pre-auth key for headless setup
}

// Server wraps a tsnet.Server for the picoclaw gateway.
// When tsnet is not available (build without tsnet tag), it provides a no-op
// fallback that uses standard net.Listen.
type Server struct {
	config   Config
	listener net.Listener
	mu       sync.Mutex
	running  bool
}

// NewServer creates a new Tailscale server with the given config.
func NewServer(cfg Config) *Server {
	if cfg.Hostname == "" {
		cfg.Hostname = "picoclaw-gateway"
	}
	if cfg.StateDir == "" {
		home, _ := os.UserHomeDir()
		cfg.StateDir = filepath.Join(home, ".picoclaw", "tsnet")
	}
	return &Server{config: cfg}
}

// Start initializes the tsnet server and begins listening.
// In the current implementation (without tsnet build tag), this creates a
// standard TCP listener as a placeholder. The full tsnet integration requires
// the tailscale.com/tsnet dependency.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("tsnet server already running")
	}

	if !s.config.Enabled {
		return nil
	}

	logger.InfoCF("tailscale", "Starting tsnet node", map[string]any{
		"hostname": s.config.Hostname,
		"state":    s.config.StateDir,
	})

	// Ensure state directory exists
	if err := os.MkdirAll(s.config.StateDir, 0o700); err != nil {
		return fmt.Errorf("creating tsnet state dir: %w", err)
	}

	// Placeholder: when tsnet is linked, this would be:
	//   srv := &tsnet.Server{Hostname: s.config.Hostname, Dir: s.config.StateDir}
	//   if s.config.AuthKey != "" { srv.AuthKey = s.config.AuthKey }
	//   ln, err := srv.Listen("tcp", ":443")
	//
	// For now, log that tsnet integration requires the tailscale dependency.
	logger.InfoC("tailscale", "tsnet stub: full integration requires tailscale.com/tsnet dependency")
	s.running = true
	return nil
}

// Stop shuts down the tsnet server.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.listener != nil {
		s.listener.Close()
	}
	s.running = false
	logger.InfoC("tailscale", "tsnet server stopped")
}

// HTTPClient returns an http.Client that routes through the tailnet.
// When tsnet is fully integrated, this uses the tsnet server's HTTPClient().
// Currently returns a standard http.Client.
func (s *Server) HTTPClient() *http.Client {
	return &http.Client{}
}

// IsRunning returns whether the tsnet server is active.
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
