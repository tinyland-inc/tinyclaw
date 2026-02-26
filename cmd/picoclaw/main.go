// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/agent"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/auth"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/cron"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/gateway"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/migrate"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/onboard"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/skills"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/status"
	"github.com/tinyland-inc/picoclaw/cmd/picoclaw/internal/version"
)

func NewPicoclawCommand() *cobra.Command {
	short := fmt.Sprintf("%s picoclaw - Personal AI Assistant v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "picoclaw",
		Short:   short,
		Example: "picoclaw list",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

func main() {
	cmd := NewPicoclawCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
