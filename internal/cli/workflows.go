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

func (a *app) workflowsCmd() *cobra.Command {
	wf := &cobra.Command{
		Use:     "workflows",
		Aliases: []string{"workflow"},
		Short:   "List workflows, run them, and inspect executions",
	}
	wf.AddCommand(
		a.workflowsListCmd(),
		a.workflowsGetCmd(),
		a.workflowsRunCmd(),
		a.workflowsExecutionsCmd(),
		a.workflowsRunsCmd(),
	)
	return wf
}

func (a *app) workflowsListCmd() *cobra.Command {
	var isV2 bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List workflows (GET /workflows)",
		Example: "  keep workflows list --format table",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			if isV2 {
				q.Set("is_v2", "true")
			}
			return a.do(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   "/workflows",
				Query:  q,
			})
		},
	}
	cmd.Flags().BoolVar(&isV2, "v2", false, "request v2 workflows")
	return cmd
}

func (a *app) workflowsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get WORKFLOW_ID",
		Short: "Get a workflow by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := url.PathEscape(args[0])
			return a.do(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/workflows/%s", id),
			})
		},
	}
}

func (a *app) workflowsRunCmd() *cobra.Command {
	var jsonBody, bodyFile string
	cmd := &cobra.Command{
		Use:   "run WORKFLOW_ID",
		Short: "Run a workflow (POST /workflows/{id}/run)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("workflow run"); err != nil {
				return err
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil {
				return err
			}
			if body == nil {
				body = []byte("{}")
			}
			id := url.PathEscape(args[0])
			return a.do(context.Background(), keep.Request{
				Method: http.MethodPost,
				Path:   fmt.Sprintf("/workflows/%s/run", id),
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "optional trigger context JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body")
	return cmd
}

func (a *app) workflowsExecutionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "executions",
		Short: "Workflow executions linked to alerts (GET /workflows/executions)",
		Long:  `The upstream OpenAPI entry has no query parameters; this calls the documented path as-is.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.do(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   "/workflows/executions",
			})
		},
	}
}

func (a *app) workflowsRunsCmd() *cobra.Command {
	var limit, offset, tab int
	var executionID string
	var status, trigger []string
	cmd := &cobra.Command{
		Use:   "runs WORKFLOW_ID",
		Short: "List runs for a workflow (GET /workflows/{id}/runs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := url.PathEscape(args[0])
			q := url.Values{}
			if cmd.Flags().Changed("limit") {
				q.Set("limit", strconv.Itoa(limit))
			}
			if cmd.Flags().Changed("offset") {
				q.Set("offset", strconv.Itoa(offset))
			}
			if cmd.Flags().Changed("tab") {
				q.Set("tab", strconv.Itoa(tab))
			}
			if executionID != "" {
				q.Set("execution_id", executionID)
			}
			for _, s := range status {
				q.Add("status", s)
			}
			for _, t := range trigger {
				q.Add("trigger", t)
			}
			return a.do(context.Background(), keep.Request{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("/workflows/%s/runs", id),
				Query:  q,
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 25, "page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset")
	cmd.Flags().IntVar(&tab, "tab", 1, "UI tab index")
	cmd.Flags().StringVar(&executionID, "execution-id", "", "filter by execution id")
	cmd.Flags().StringSliceVar(&status, "status", nil, "filter by status (repeatable)")
	cmd.Flags().StringSliceVar(&trigger, "trigger", nil, "filter by trigger (repeatable)")
	return cmd
}
