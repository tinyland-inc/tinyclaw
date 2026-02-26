package migrate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tinyland-inc/picoclaw/pkg/migrate"
)

func NewMigrateCommand() *cobra.Command {
	var opts migrate.Options

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate configuration between formats",
		Example: `  picoclaw migrate
  picoclaw migrate --dry-run
  picoclaw migrate --to-dhall
  picoclaw migrate --to-dhall --config /path/to/config.json`,
	}

	// Default: OpenClaw -> PicoClaw migration
	openclawCmd := &cobra.Command{
		Use:   "openclaw",
		Short: "Migrate from OpenClaw to PicoClaw",
		Args:  cobra.NoArgs,
		Example: `  picoclaw migrate openclaw
  picoclaw migrate openclaw --dry-run
  picoclaw migrate openclaw --refresh
  picoclaw migrate openclaw --force`,
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := migrate.Run(opts)
			if err != nil {
				return err
			}
			if !opts.DryRun {
				migrate.PrintSummary(result)
			}
			return nil
		},
	}

	openclawCmd.Flags().BoolVar(&opts.DryRun, "dry-run", false,
		"Show what would be migrated without making changes")
	openclawCmd.Flags().BoolVar(&opts.Refresh, "refresh", false,
		"Re-sync workspace files from OpenClaw (repeatable)")
	openclawCmd.Flags().BoolVar(&opts.ConfigOnly, "config-only", false,
		"Only migrate config, skip workspace files")
	openclawCmd.Flags().BoolVar(&opts.WorkspaceOnly, "workspace-only", false,
		"Only migrate workspace files, skip config")
	openclawCmd.Flags().BoolVar(&opts.Force, "force", false,
		"Skip confirmation prompts")
	openclawCmd.Flags().StringVar(&opts.OpenClawHome, "openclaw-home", "",
		"Override OpenClaw home directory (default: ~/.openclaw)")
	openclawCmd.Flags().StringVar(&opts.PicoClawHome, "picoclaw-home", "",
		"Override PicoClaw home directory (default: ~/.picoclaw)")

	// to-dhall: JSON -> Dhall migration
	var dhallOpts migrate.ToDhallOptions

	toDhallCmd := &cobra.Command{
		Use:   "to-dhall",
		Short: "Convert JSON config to Dhall format",
		Args:  cobra.NoArgs,
		Example: `  picoclaw migrate to-dhall
  picoclaw migrate to-dhall --dry-run
  picoclaw migrate to-dhall --config ~/.picoclaw/config.json
  picoclaw migrate to-dhall --output ~/.picoclaw/config.dhall --force`,
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := migrate.RunToDhall(dhallOpts)
			if err != nil {
				return err
			}
			if !dhallOpts.DryRun {
				fmt.Printf("Dhall config written to %s\n", result.OutputPath)
			}
			if len(result.Warnings) > 0 {
				fmt.Println("\nWarnings:")
				for _, w := range result.Warnings {
					fmt.Printf("  - %s\n", w)
				}
			}
			return nil
		},
	}

	toDhallCmd.Flags().StringVar(&dhallOpts.ConfigPath, "config", "",
		"JSON config file path (default: ~/.picoclaw/config.json)")
	toDhallCmd.Flags().StringVar(&dhallOpts.OutputPath, "output", "",
		"Dhall output file path (default: same dir as input, .dhall extension)")
	toDhallCmd.Flags().BoolVar(&dhallOpts.DryRun, "dry-run", false,
		"Print generated Dhall without writing")
	toDhallCmd.Flags().BoolVar(&dhallOpts.Force, "force", false,
		"Overwrite existing output file")

	// Legacy: bare `picoclaw migrate` still runs OpenClaw migration for compatibility
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		result, err := migrate.Run(opts)
		if err != nil {
			return err
		}
		if !opts.DryRun {
			migrate.PrintSummary(result)
		}
		return nil
	}
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false,
		"Show what would be migrated without making changes")
	cmd.Flags().BoolVar(&opts.Force, "force", false,
		"Skip confirmation prompts")

	cmd.AddCommand(openclawCmd)
	cmd.AddCommand(toDhallCmd)

	return cmd
}
