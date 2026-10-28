package cli

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/keep-cli/internal/keep"
)

func (a *app) providersCmd() *cobra.Command {
	p := &cobra.Command{
		Use:     "providers",
		Aliases: []string{"provider"},
		Short:   "List available and installed providers",
	}
	p.AddCommand(
		&cobra.Command{
			Use:     "list",
			Short:   "List provider types (GET /providers)",
			Example: "  keep providers list",
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.do(context.Background(), keep.Request{
					Method: http.MethodGet,
					Path:   "/providers",
				})
			},
		},
		&cobra.Command{
			Use:   "installed",
			Short: "Export installed providers (GET /providers/export)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.do(context.Background(), keep.Request{
					Method: http.MethodGet,
					Path:   "/providers/export",
				})
			},
		},
	)
	return p
}
