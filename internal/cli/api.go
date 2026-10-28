package cli

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/keep-cli/internal/keep"
)

var apiWriteBlocklist = []struct {
	method string
	prefix string
}{
	{http.MethodDelete, "/settings/"},
	{http.MethodDelete, "/auth/"},
	{http.MethodPost, "/providers/install"},
	{http.MethodDelete, "/providers/"},
}

func (a *app) apiCmd() *cobra.Command {
	var jsonBody, bodyFile string
	var query []string
	cmd := &cobra.Command{
		Use:   "api METHOD PATH",
		Short: "Call any Keep API path from the OpenAPI spec",
		Long: `Escape hatch for endpoints not wrapped by first-class commands.
METHOD is GET, POST, PUT, PATCH, or DELETE. PATH must start with /.

Write access and the api blocklist still apply to state-changing calls.`,
		Example: `  keep api GET /healthcheck
  keep api POST /alerts/search --json '{"query":{"sql":"*"},"timeframe":86400}'
  keep api GET /metrics`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			path := args[1]
			if !strings.HasPrefix(path, "/") {
				return usageErr("PATH must start with /")
			}
			switch method {
			case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			default:
				return usageErr("unsupported METHOD %q", method)
			}
			if method != http.MethodGet && method != http.MethodHead {
				if err := a.requireWrite("api " + method); err != nil {
					return err
				}
				if blocked, reason := apiBlocked(method, path); blocked {
					return policyErr(reason)
				}
			}
			body, err := readBodyFlag(jsonBody, bodyFile)
			if err != nil {
				return err
			}
			q, err := parseQueryFlags(query)
			if err != nil {
				return err
			}
			return a.do(context.Background(), keep.Request{
				Method: method,
				Path:   path,
				Query:  q,
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&jsonBody, "json", "", "request body JSON")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON body (@file)")
	cmd.Flags().StringArrayVar(&query, "query", nil, "query param key=value (repeatable)")
	return cmd
}

func parseQueryFlags(pairs []string) (url.Values, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	v := url.Values{}
	for _, p := range pairs {
		k, val, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, usageErr("query flag must be key=value")
		}
		v.Add(k, val)
	}
	return v, nil
}

func apiBlocked(method, path string) (bool, string) {
	for _, b := range apiWriteBlocklist {
		if method == b.method && strings.HasPrefix(path, b.prefix) {
			return true, "blocked by policy: " + method + " " + path + " is permanently blocked; use a narrower command or ask the user"
		}
	}
	return false, ""
}
