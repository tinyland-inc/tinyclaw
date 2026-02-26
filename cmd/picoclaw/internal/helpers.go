package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tinyland-inc/picoclaw/pkg/config"
)

const Logo = "🦞"

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

func GetDhallConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.dhall")
}

func LoadConfig() (*config.Config, error) {
	// Try Dhall config first (opt-in: only if .dhall file exists)
	dhallPath := GetDhallConfigPath()
	if _, err := os.Stat(dhallPath); err == nil {
		cfg, err := config.LoadDhallConfig(dhallPath)
		if err != nil {
			return nil, fmt.Errorf("error loading dhall config: %w", err)
		}
		if cfg != nil {
			return cfg, nil
		}
		// cfg == nil means dhall-to-json not installed, fall through to JSON
	}

	return config.LoadConfig(GetConfigPath())
}

// FormatVersion returns the version string with optional git commit
func FormatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// FormatBuildInfo returns build time and go version info
func FormatBuildInfo() (string, string) {
	build := buildTime
	goVer := goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return build, goVer
}

// GetVersion returns the version string
func GetVersion() string {
	return version
}
