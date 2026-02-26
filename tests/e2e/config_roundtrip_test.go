package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tinyland-inc/picoclaw/pkg/config"
	"github.com/tinyland-inc/picoclaw/pkg/migrate"
)

// TestConfigRoundtrip verifies that JSON -> Dhall -> JSON produces equivalent output.
// Requires dhall-to-json to be installed.
func TestConfigRoundtrip(t *testing.T) {
	if _, err := exec.LookPath("dhall-to-json"); err != nil {
		t.Skip("dhall-to-json not installed, skipping roundtrip test")
	}

	// The migration redacts API keys to env:PICOCLAW_API_KEY references;
	// set it so dhall-to-json can resolve it.
	t.Setenv("PICOCLAW_API_KEY", "test-key")

	tmpDir := t.TempDir()

	// Step 1: Save default config as JSON
	cfg := config.DefaultConfig()
	jsonPath := filepath.Join(tmpDir, "config.json")
	if err := config.SaveConfig(jsonPath, cfg); err != nil {
		t.Fatalf("saving JSON config: %v", err)
	}

	// Step 2: Convert JSON -> Dhall
	dhallPath := filepath.Join(tmpDir, "config.dhall")
	result, err := migrate.RunToDhall(migrate.ToDhallOptions{
		ConfigPath: jsonPath,
		OutputPath: dhallPath,
	})
	if err != nil {
		t.Fatalf("converting to dhall: %v", err)
	}
	t.Logf("Dhall output: %s", result.OutputPath)

	// Step 3: Convert Dhall -> JSON via dhall-to-json
	dhallCfg, err := config.LoadDhallConfig(dhallPath)
	if err != nil {
		t.Fatalf("loading dhall config: %v", err)
	}
	if dhallCfg == nil {
		t.Fatal("LoadDhallConfig returned nil (dhall-to-json should be available)")
	}

	// Step 4: Compare key structural fields
	// We compare the essential structure, not exact JSON equality, because:
	// - Model list API keys are redacted during Dhall generation
	// - Optional fields may have different zero values
	if dhallCfg.Gateway.Host != cfg.Gateway.Host {
		t.Errorf("gateway.host: got %s, want %s", dhallCfg.Gateway.Host, cfg.Gateway.Host)
	}
	if dhallCfg.Gateway.Port != cfg.Gateway.Port {
		t.Errorf("gateway.port: got %d, want %d", dhallCfg.Gateway.Port, cfg.Gateway.Port)
	}
	if dhallCfg.Agents.Defaults.MaxTokens != cfg.Agents.Defaults.MaxTokens {
		t.Errorf("agents.defaults.max_tokens: got %d, want %d",
			dhallCfg.Agents.Defaults.MaxTokens, cfg.Agents.Defaults.MaxTokens)
	}
	if dhallCfg.Agents.Defaults.MaxToolIterations != cfg.Agents.Defaults.MaxToolIterations {
		t.Errorf("agents.defaults.max_tool_iterations: got %d, want %d",
			dhallCfg.Agents.Defaults.MaxToolIterations, cfg.Agents.Defaults.MaxToolIterations)
	}
	if dhallCfg.Heartbeat.Enabled != cfg.Heartbeat.Enabled {
		t.Errorf("heartbeat.enabled: got %v, want %v", dhallCfg.Heartbeat.Enabled, cfg.Heartbeat.Enabled)
	}
	if dhallCfg.Heartbeat.Interval != cfg.Heartbeat.Interval {
		t.Errorf("heartbeat.interval: got %d, want %d", dhallCfg.Heartbeat.Interval, cfg.Heartbeat.Interval)
	}
}

// TestDhallExampleRoundtrip tests that the checked-in Dhall examples render to valid JSON
// and that the JSON can be loaded as a valid config.
func TestDhallExampleRoundtrip(t *testing.T) {
	if _, err := exec.LookPath("dhall-to-json"); err != nil {
		t.Skip("dhall-to-json not installed, skipping example roundtrip test")
	}

	examples, err := filepath.Glob("../../dhall/examples/*.dhall")
	if err != nil {
		t.Fatalf("globbing dhall examples: %v", err)
	}
	if len(examples) == 0 {
		t.Skip("no dhall examples found")
	}

	for _, example := range examples {
		t.Run(filepath.Base(example), func(t *testing.T) {
			cmd := exec.Command("dhall-to-json", "--file", example)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("dhall-to-json failed for %s: %v", example, err)
			}

			// Verify it's valid JSON
			var raw map[string]any
			if err := json.Unmarshal(out, &raw); err != nil {
				t.Fatalf("invalid JSON output from %s: %v", example, err)
			}

			// Verify essential config fields exist
			if _, ok := raw["agents"]; !ok {
				t.Errorf("missing 'agents' key in %s output", example)
			}
			if _, ok := raw["channels"]; !ok {
				t.Errorf("missing 'channels' key in %s output", example)
			}
			if _, ok := raw["gateway"]; !ok {
				t.Errorf("missing 'gateway' key in %s output", example)
			}
		})
	}
}

// TestDefaultConfigJSON verifies the default config marshals to valid JSON.
func TestDefaultConfigJSON(t *testing.T) {
	cfg := config.DefaultConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshaling default config: %v", err)
	}

	var roundtrip config.Config
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("unmarshaling default config: %v", err)
	}

	if roundtrip.Gateway.Host != cfg.Gateway.Host {
		t.Errorf("gateway.host roundtrip: got %s, want %s", roundtrip.Gateway.Host, cfg.Gateway.Host)
	}
}

// TestConfigLoadAndSaveRoundtrip tests JSON load -> save -> load roundtrip.
func TestConfigLoadAndSaveRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	// Save
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "10.0.0.1"
	cfg.Gateway.Port = 9999
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Load
	loaded, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	if loaded.Gateway.Host != "10.0.0.1" {
		t.Errorf("gateway.host: got %s, want 10.0.0.1", loaded.Gateway.Host)
	}
	if loaded.Gateway.Port != 9999 {
		t.Errorf("gateway.port: got %d, want 9999", loaded.Gateway.Port)
	}
}

// TestDhallConfigLoaderFallback tests that LoadDhallConfig returns nil when
// dhall-to-json is not found (simulated by clearing PATH).
func TestDhallConfigLoaderFallback(t *testing.T) {
	tmpDir := t.TempDir()
	dhallPath := filepath.Join(tmpDir, "config.dhall")
	os.WriteFile(dhallPath, []byte("{ agents = {}}"), 0o600)

	// Save original PATH and set to empty
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir) // dir with no dhall-to-json
	defer os.Setenv("PATH", origPath)

	cfg, err := config.LoadDhallConfig(dhallPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config when dhall-to-json not available")
	}
}
