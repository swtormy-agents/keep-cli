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

func (a *app) incidentsCmd() *cobra.Command {
	inc := &cobra.Command{
		Use:     "incidents",
		Aliases: []string{"incident"},
		Short:   "List and manage incidents",
	}
	inc.AddCommand(
		a.incidentsListCmd(),
		a.incidentsGetCmd(),
		a.incidentsCreateCmd(),
		a.incidentsUpdateCmd(),
		a.incidentsMergeCmd(),
		a.incidentsAlertsCmd(),
	)
	return inc
}

func (a *app) incidentsListCmd() *cobra.Command {
	var limit, offset int
	var confirmed bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List incidents (GET /incidents)",
		Example: "  keep incidents list --limit 10 --format table",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			q.Set("confirmed", strconv.FormatBool(confirmed))
			if cmd.Flags().Changed("limit") {
				q.Set("limit", strconv.Itoa(limit))
			}
			if cmd.Flags().Changed("offset") {
				q.Set("offset", strconv.Itoa(offset))
			}
			return a.doUntrusted(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   "/incidents",
				Query:  q,
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 25, "page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset")
	cmd.Flags().BoolVar(&confirmed, "confirmed", true, "filter confirmed incidents")
	return cmd
}

func (a *app) incidentsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get INCIDENT_ID",
		Short: "Get an incident by UUID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := url.PathEscape(args[0])
			return a.doUntrusted(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/incidents/%s", id),
			})
		},
	}
}

func (a *app) incidentsCreateCmd() *cobra.Command {
	var jsonBody, bodyFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an incident (POST /incidents)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("incident create"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil || body == nil {
				return usageErr("pass --json or --body with IncidentDtoIn")
			}
			return a.do(context.Background(), keep.Request{
				Method: http.MethodPost,
				Path:   "/incidents",
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "IncidentDtoIn JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body")
	return cmd
}

func (a *app) incidentsUpdateCmd() *cobra.Command {
	var jsonBody, bodyFile string
	var generatedByAI bool
	cmd := &cobra.Command{
		Use:   "update INCIDENT_ID",
		Short: "Update an incident (PUT /incidents/{id})",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("incident update"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil || body == nil {
				return usageErr("pass --json or --body with IncidentDtoIn")
			}
			id := url.PathEscape(args[0])
			q := url.Values{}
			if generatedByAI {
				q.Set("generatedByAi", "true")
			}
			return a.do(context.Background(), keep.Request{
				Method: http.MethodPut,
				Path:   fmt.Sprintf("/incidents/%s", id),
				Query:  q,
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "IncidentDtoIn JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body")
	cmd.Flags().BoolVar(&generatedByAI, "generated-by-ai", false, "set generatedByAi query flag")
	return cmd
}

func (a *app) incidentsMergeCmd() *cobra.Command {
	var jsonBody, bodyFile string
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge incidents (POST /incidents/merge)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("incident merge"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil || body == nil {
				return usageErr("pass --json or --body with merge payload")
			}
			return a.do(context.Background(), keep.Request{
				Method: http.MethodPost,
				Path:   "/incidents/merge",
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "merge request JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body")
	return cmd
}

func (a *app) incidentsAlertsCmd() *cobra.Command {
	var jsonBody, bodyFile string
	alerts := &cobra.Command{
		Use:   "alerts",
		Short: "Alerts linked to an incident",
	}
	alerts.AddCommand(&cobra.Command{
		Use:   "list INCIDENT_ID",
		Short: "List alerts in the incident (GET)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := url.PathEscape(args[0])
			return a.doUntrusted(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/incidents/%s/alerts", id),
			})
		},
	})
	alerts.AddCommand(&cobra.Command{
		Use:   "add INCIDENT_ID",
		Short: "Add alerts to the incident (POST)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("incident alerts add"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil || body == nil {
				return usageErr("pass --json or --body")
			}
			id := url.PathEscape(args[0])
			return a.do(context.Background(), keep.Request{
				Method: http.MethodPost,
				Path:   fmt.Sprintf("/incidents/%s/alerts", id),
				Body:   body,
			})
		},
	})
	alerts.AddCommand(&cobra.Command{
		Use:   "remove INCIDENT_ID",
		Short: "Remove alerts from the incident (DELETE)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("incident alerts remove"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil || body == nil {
				return usageErr("pass --json or --body")
			}
			id := url.PathEscape(args[0])
			return a.do(context.Background(), keep.Request{
				Method: http.MethodDelete,
				Path:   fmt.Sprintf("/incidents/%s/alerts", id),
				Body:   body,
			})
		},
	})
	alerts.PersistentFlags().StringVar(&jsonBody, "json", "", "request JSON")
	alerts.PersistentFlags().StringVar(&bodyFile, "body", "", "path to JSON body")
	return alerts
}
