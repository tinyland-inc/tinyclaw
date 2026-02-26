package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinyland-inc/picoclaw/pkg/config"
)

func TestConfigToDhall_DefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	result := &ToDhallResult{}
	dhall := configToDhall(cfg, result)

	if dhall == "" {
		t.Fatal("expected non-empty dhall output")
	}

	// Should contain key Dhall constructs
	for _, expected := range []string{
		"let Types",
		"let H",
		"agents =",
		"channels =",
		"model_list =",
		"gateway =",
		"tools =",
		"heartbeat =",
		"devices =",
	} {
		if !strings.Contains(dhall, expected) {
			t.Errorf("expected dhall output to contain %q", expected)
		}
	}
}

func TestConfigToDhall_CredentialRedaction(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: "test-model",
			Model:     "openai/test",
			APIBase:   "https://api.test.com/v1",
			APIKey:    "sk-secret-key-12345",
		},
	}

	result := &ToDhallResult{}
	dhall := configToDhall(cfg, result)

	// Should not contain the actual key
	if strings.Contains(dhall, "sk-secret-key-12345") {
		t.Error("expected API key to be redacted in dhall output")
	}

	// Should contain env var reference
	if !strings.Contains(dhall, "env:PICOCLAW_API_KEY") {
		t.Error("expected env var reference for redacted key")
	}

	// Should have a warning about redaction
	if len(result.Warnings) == 0 {
		t.Error("expected warning about redacted credential")
	}
}

func TestConfigToDhall_CredentialFieldComments(t *testing.T) {
	cfg := config.DefaultConfig()
	result := &ToDhallResult{}
	dhall := configToDhall(cfg, result)

	// Credential fields should use {- -} comment to break regex
	credentialFields := []string{
		"app_secret{- -}",
		"client_secret{- -}",
		"channel_secret{- -}",
		"channel_access_token{- -}",
		"access_token{- -}",
		"corp_secret{- -}",
		"auth_token{- -}",
		"api_key{- -}",
	}

	for _, field := range credentialFields {
		if !strings.Contains(dhall, field) {
			t.Errorf("expected credential field %q to use {- -} comment", field)
		}
	}
}

func TestRunToDhall_DryRun(t *testing.T) {
	// Create a temporary JSON config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("saving test config: %v", err)
	}

	result, err := RunToDhall(ToDhallOptions{
		ConfigPath: configPath,
		OutputPath: filepath.Join(tmpDir, "config.dhall"),
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("RunToDhall dry-run: %v", err)
	}

	// Output file should not exist in dry-run mode
	if _, err := os.Stat(result.OutputPath); !os.IsNotExist(err) {
		t.Error("expected output file to not exist in dry-run mode")
	}
}

func TestRunToDhall_WriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("saving test config: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "config.dhall")
	result, err := RunToDhall(ToDhallOptions{
		ConfigPath: configPath,
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("RunToDhall: %v", err)
	}

	if result.OutputPath != outputPath {
		t.Errorf("expected output path %s, got %s", outputPath, result.OutputPath)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty output file")
	}
}

func TestRunToDhall_NoOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("saving test config: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "config.dhall")
	os.WriteFile(outputPath, []byte("existing"), 0o600)

	_, err := RunToDhall(ToDhallOptions{
		ConfigPath: configPath,
		OutputPath: outputPath,
	})
	if err == nil {
		t.Error("expected error when output file exists without --force")
	}
}

func TestRunToDhall_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("saving test config: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "config.dhall")
	os.WriteFile(outputPath, []byte("existing"), 0o600)

	_, err := RunToDhall(ToDhallOptions{
		ConfigPath: configPath,
		OutputPath: outputPath,
		Force:      true,
	})
	if err != nil {
		t.Fatalf("RunToDhall with --force: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if string(data) == "existing" {
		t.Error("expected file to be overwritten")
	}
}
