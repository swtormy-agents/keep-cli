package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/keep-cli/internal/keep"
)

func (a *app) alertsCmd() *cobra.Command {
	alerts := &cobra.Command{
		Use:     "alerts",
		Aliases: []string{"alert"},
		Short:   "List, search, and manage alerts",
	}
	alerts.AddCommand(
		a.alertsListCmd(),
		a.alertsGetCmd(),
		a.alertsSearchCmd(),
		a.alertsEnrichCmd(),
		a.alertsAssignCmd(),
		a.alertsDismissCmd(),
	)
	return alerts
}

func (a *app) alertsListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List recent alerts (GET /alerts)",
		Example: "  keep alerts list --limit 50\n  keep alerts list --format table",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			if cmd.Flags().Changed("limit") {
				q.Set("limit", strconv.Itoa(limit))
			}
			return a.doUntrusted(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   "/alerts",
				Query:  q,
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 1000, "max alerts to return")
	return cmd
}

func (a *app) alertsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get FINGERPRINT",
		Short:   "Get one alert by fingerprint",
		Args:    cobra.ExactArgs(1),
		Example: "  keep alerts get abc123def",
		RunE: func(cmd *cobra.Command, args []string) error {
			fp := url.PathEscape(args[0])
			return a.doUntrusted(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/alerts/%s", fp),
			})
		},
	}
}

func (a *app) alertsSearchCmd() *cobra.Command {
	var jsonBody, bodyFile string
	var timeframe int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search alerts (POST /alerts/search)",
		Long:  `Requires a SearchAlertsRequest JSON body with "query" and "timeframe" (seconds).`,
		Example: `  keep alerts search --json '{"query":{"sql":"status = 'firing'"},"timeframe":3600}'
  keep alerts search --body @search.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil {
				return err
			}
			if body == nil {
				if timeframe == 0 {
					return usageErr("pass --json or --body with a SearchAlertsRequest")
				}
				body = []byte(fmt.Sprintf(`{"query":{"sql":"*"},"timeframe":%d}`, timeframe))
			}
			return a.doUntrusted(context.Background(), keep.Request{
				Method: http.MethodPost,
				Path:   "/alerts/search",
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "request body JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body (@file.json)")
	cmd.Flags().IntVar(&timeframe, "timeframe", 0, "shortcut: search with query sql * and this timeframe")
	return cmd
}

func (a *app) alertsEnrichCmd() *cobra.Command {
	var jsonBody, bodyFile string
	var dispose bool
	cmd := &cobra.Command{
		Use:   "enrich",
		Short: "Add enrichments to an alert (POST /alerts/enrich)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("alert enrich"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil || body == nil {
				return usageErr("pass --json with enrichments and fingerprint")
			}
			q := url.Values{}
			if dispose {
				q.Set("dispose_on_new_alert", "true")
			}
			return a.do(context.Background(), keep.Request{
				Method: http.MethodPost,
				Path:   "/alerts/enrich",
				Query:  q,
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "EnrichAlertRequestBody JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body")
	cmd.Flags().BoolVar(&dispose, "dispose-on-new-alert", false, "dispose enrichments on new alert")
	return cmd
}

func (a *app) alertsAssignCmd() *cobra.Command {
	var unassign bool
	cmd := &cobra.Command{
		Use:   "assign FINGERPRINT LAST_RECEIVED",
		Short: "Assign the alert to the current API user (POST .../assign/...)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("alert assign"); err != nil {
				return err
			}
			fp := url.PathEscape(args[0])
			lr := url.PathEscape(args[1])
			q := url.Values{}
			if unassign {
				q.Set("unassign", "true")
			}
			path := fmt.Sprintf("/alerts/%s/assign/%s", fp, lr)
			return a.do(context.Background(), keep.Request{
				Method: http.MethodPost,
				Path:   path,
				Query:  q,
			})
		},
	}
	cmd.Flags().BoolVar(&unassign, "unassign", false, "unassign instead of assign")
	return cmd
}

func (a *app) alertsDismissCmd() *cobra.Command {
	var jsonBody, bodyFile, fingerprint, lastReceived string
	cmd := &cobra.Command{
		Use:     "dismiss",
		Short:   "Dismiss (delete) an alert occurrence (DELETE /alerts)",
		Aliases: []string{"delete"},
		Long:    `Maps to DELETE /alerts with fingerprint and lastReceived.`,
		Example: `  keep alerts dismiss --fingerprint FP --last-received "2024-01-01T00:00:00Z"
  keep alerts dismiss --json '{"fingerprint":"fp","lastReceived":"..."}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("alert dismiss"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil {
				return err
			}
			if body == nil {
				if fingerprint == "" || lastReceived == "" {
					return usageErr("pass --json or both --fingerprint and --last-received")
				}
				body = []byte(fmt.Sprintf(`{"fingerprint":%q,"lastReceived":%q}`, fingerprint, lastReceived))
			}
			return a.do(context.Background(), keep.Request{
				Method: http.MethodDelete,
				Path:   "/alerts",
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "DeleteRequestBody JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body")
	cmd.Flags().StringVar(&fingerprint, "fingerprint", "", "alert fingerprint")
	cmd.Flags().StringVar(&lastReceived, "last-received", "", "alert lastReceived timestamp")
	return cmd
}
