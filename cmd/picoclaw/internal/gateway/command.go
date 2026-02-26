package gateway

import (
	"github.com/spf13/cobra"
)

func NewGatewayCommand() *cobra.Command {
	var debug bool
	var verified bool
	var legacy bool

	cmd := &cobra.Command{
		Use:     "gateway",
		Aliases: []string{"g"},
		Short:   "Start picoclaw gateway",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			mode := GatewayModeVerified // Default to verified mode
			if legacy {
				mode = GatewayModeLegacy
			}
			return gatewayCmd(debug, mode)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().BoolVar(&verified, "verified", false, "Use F*-verified core for message processing (default)")
	cmd.Flags().BoolVar(&legacy, "legacy", false, "Use legacy Go agent loop")

	return cmd
}
