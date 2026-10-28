package cli

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/keep-cli/internal/keep"
)

func (a *app) statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check API connectivity and current access mode",
		Long:  `Calls GET /whoami when an API key is present, and reports read/write access mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.loadConfig(); err != nil {
				return err
			}
			out := map[string]any{
				"ok":       true,
				"api_url":  a.cfg.APIURL,
				"access":   a.cfg.Access,
				"access_via": a.cfg.AccessSource,
				"has_key":  a.cfg.APIKey != "",
				"key_from": a.cfg.Source,
			}
			if a.cfg.APIKey == "" {
				return a.emitValue(out)
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			raw, err := c.Do(context.Background(), keep.Request{Method: http.MethodGet, Path: "/whoami"})
			if err != nil {
				return a.wrapErr(err)
			}
			var who any
			_ = json.Unmarshal(raw, &who)
			out["whoami"] = who
			return a.emitValue(out)
		},
	}
}

func (a *app) accessCmd() *cobra.Command {
	access := &cobra.Command{
		Use:   "access",
		Short: "View or change the local write-access policy",
	}
	access.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print current access mode (read or write)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.loadConfig(); err != nil {
				return err
			}
			return a.emitValue(map[string]any{"ok": true, "access": a.cfg.Access, "access_via": a.cfg.AccessSource})
		},
	})
	var confirm string
	write := &cobra.Command{
		Use:   "write",
		Short: "Enable write access for state-changing commands (requires a TTY)",
		Long:  `Write access is stored in ~/.config/keep/access (mode 0600). Agents cannot run this; the user must.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !a.isTTY() {
				return policyErr("keep access write requires an interactive terminal")
			}
			if confirm != "write" {
				return usageErr(`re-run with --confirm write (type the word "write")`)
			}
			if err := keep.WriteAccessMode("write"); err != nil {
				return runtimeErr(err)
			}
			return a.emitValue(map[string]any{"ok": true, "access": "write"})
		},
	}
	write.Flags().StringVar(&confirm, "confirm", "", `must be the word "write"`)
	access.AddCommand(write)

	read := &cobra.Command{
		Use:   "read",
		Short: "Return to read-only mode (requires a TTY)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !a.isTTY() {
				return policyErr("keep access read requires an interactive terminal")
			}
			if err := keep.WriteAccessMode("read"); err != nil {
				return runtimeErr(err)
			}
			return a.emitValue(map[string]any{"ok": true, "access": "read"})
		},
	}
	access.AddCommand(read)
	return access
}
