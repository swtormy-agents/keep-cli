package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/keep-cli/internal/keep"
)

const (
	exitRuntime = 1
	exitUsage   = 2
	exitAuth    = 3
	exitPolicy  = 4
)

type exitError struct {
	code int
	kind string
	err  error
	api  *keep.APIError
}

func (e *exitError) Error() string { return e.err.Error() }

func usageErr(format string, a ...any) error {
	return &exitError{code: exitUsage, kind: "usage", err: fmt.Errorf(format, a...)}
}

func runtimeErr(err error) error {
	return &exitError{code: exitRuntime, kind: "runtime", err: err}
}

func authErr(err error) error {
	return &exitError{code: exitAuth, kind: "auth", err: err}
}

func policyErr(msg string) error {
	return &exitError{code: exitPolicy, kind: "policy", err: errors.New(msg)}
}

var Version = "dev"

func versionString() string {
	v := Version
	if v != "dev" {
		return v
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		rev, dirty := "", ""
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				if s.Value == "true" {
					dirty = "+dirty"
				}
			}
		}
		if len(rev) > 7 {
			rev = rev[:7]
		}
		if rev != "" {
			return "dev-" + rev + dirty
		}
	}
	return v
}

type app struct {
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
	getenv  func(string) string
	isTTY   func() bool

	cfg     keep.Config
	dryRun  bool
	jq      string
	pretty  bool
	format  string
	apiURL  string
	timeout time.Duration

	accessWarnOnce sync.Once
}

func Execute(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	a := &app{
		in: stdin, out: stdout, errOut: stderr, getenv: getenv,
		isTTY: defaultIsTTY, format: "json",
	}
	root := a.rootCmd()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.Execute(); err != nil {
		return a.reportError(err)
	}
	return 0
}

func (a *app) reportError(err error) int {
	ee, ok := err.(*exitError)
	if !ok {
		ee = &exitError{code: exitUsage, kind: "usage", err: err}
	}
	detail := map[string]any{"type": ee.kind, "message": a.redact(ee.err.Error())}
	if ee.api != nil {
		detail["status"] = ee.api.Status
		if ee.api.Message != "" {
			detail["message"] = a.redact(ee.api.Message)
		}
	}
	b, _ := marshal(struct {
		OK    bool           `json:"ok"`
		Error map[string]any `json:"error"`
	}{false, detail})
	fmt.Fprintln(a.errOut, string(b))
	return ee.code
}

func (a *app) redact(s string) string {
	return (&keep.Client{APIKey: a.cfg.APIKey}).Redact(s)
}

func (a *app) loadConfig() error {
	cfg, err := keep.LoadConfig(a.getenv)
	if err != nil {
		return runtimeErr(err)
	}
	if a.apiURL != "" {
		cfg.APIURL = a.apiURL
	}
	a.cfg = cfg
	return nil
}

func (a *app) client() (*keep.Client, error) {
	if err := a.loadConfig(); err != nil {
		return nil, err
	}
	if a.cfg.APIKey == "" && !a.dryRun {
		return nil, authErr(errors.New("KEEP_API_KEY is not set (put the key in the environment or ~/.config/keep/config.json with chmod 600)"))
	}
	return &keep.Client{
		APIKey:  a.cfg.APIKey,
		BaseURL: a.cfg.APIURL,
		HTTP:    &http.Client{Timeout: a.timeout},
	}, nil
}

func (a *app) do(ctx context.Context, r keep.Request) error {
	if a.dryRun {
		if err := a.loadConfig(); err != nil {
			return err
		}
		return a.emitValue(keep.DryRun(a.cfg.APIURL, r))
	}
	c, err := a.client()
	if err != nil {
		return err
	}
	raw, err := c.Do(ctx, r)
	if err != nil {
		return a.wrapErr(err)
	}
	return a.emit(raw)
}

func (a *app) doUntrusted(ctx context.Context, r keep.Request) error {
	if a.dryRun {
		if err := a.loadConfig(); err != nil {
			return err
		}
		return a.emitValue(keep.DryRun(a.cfg.APIURL, r))
	}
	c, err := a.client()
	if err != nil {
		return err
	}
	raw, err := c.Do(ctx, r)
	if err != nil {
		return a.wrapErr(err)
	}
	return a.emitUntrusted(raw)
}

func (a *app) wrapErr(err error) error {
	var api *keep.APIError
	if errors.As(err, &api) {
		code, kind := exitRuntime, "api"
		if api.IsAuth() {
			code, kind = exitAuth, "auth"
		}
		return &exitError{code: code, kind: kind, err: api, api: api}
	}
	return runtimeErr(err)
}

func (a *app) requireWrite(action string) error {
	if err := a.loadConfig(); err != nil {
		return err
	}
	if strings.EqualFold(a.cfg.Access, "write") {
		a.warnWriteAccessSource()
		return nil
	}
	return policyErr(fmt.Sprintf("%s is blocked while access is read-only: ask the user to run `keep access write` in their own terminal", action))
}

func (a *app) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "keep",
		Short:         "Keep AIOps API from the shell, built for AI agents",
		Long:          rootLong,
		Version:       versionString(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if a.format != "json" && a.format != "table" {
				return usageErr("--format must be json or table")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	pf := root.PersistentFlags()
	pf.BoolVar(&a.dryRun, "dry-run", false, "print the HTTP request instead of sending it")
	pf.StringVar(&a.jq, "jq", "", "filter JSON output with a jq expression")
	pf.BoolVar(&a.pretty, "pretty", false, "indent JSON output")
	pf.StringVar(&a.format, "format", "json", "output format: json (default) or table")
	pf.StringVar(&a.apiURL, "api-url", "", "Keep API base URL (default https://api.keephq.dev or $KEEP_API_URL)")
	pf.DurationVar(&a.timeout, "http-timeout", 60*time.Second, "HTTP timeout per request")

	root.AddCommand(
		a.statusCmd(),
		a.accessCmd(),
		a.alertsCmd(),
		a.incidentsCmd(),
		a.workflowsCmd(),
		a.providersCmd(),
		a.apiCmd(),
		a.skillCmd(),
	)
	return root
}

const rootLong = `keep calls the Keep REST API from the shell. JSON on stdout, errors as JSON on stderr.

Auth
  export KEEP_API_KEY=...                    API key (header X-API-KEY)
  export KEEP_API_URL=https://api.keephq.dev optional base URL

Safety
  State-changing commands require write access. Default is read-only.
  The user enables writes with: keep access write  (interactive, needs a TTY)
  Non-interactive opt-in (CI/agents): export KEEP_ACCESS=write (see README)

Exit codes: 0 ok, 1 API/runtime, 2 usage, 3 auth, 4 blocked by access policy`

func (a *app) warnWriteAccessSource() {
	a.accessWarnOnce.Do(func() {
		if !strings.EqualFold(a.cfg.Access, "write") {
			return
		}
		var msg string
		switch a.cfg.AccessSource {
		case "KEEP_ACCESS":
			msg = "keep: write access enabled via KEEP_ACCESS (non-interactive opt-in; see README)"
		case "access file":
			msg = "keep: write access enabled via ~/.config/keep/access (non-interactive opt-in; see README)"
		case "default":
			return
		default:
			msg = fmt.Sprintf("keep: write access enabled via %q access field in config (non-interactive opt-in; see README)", a.cfg.AccessSource)
		}
		if msg != "" {
			fmt.Fprintln(a.errOut, msg)
		}
	})
}

func defaultIsTTY() bool {
	f, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (f.Mode() & os.ModeCharDevice) != 0
}
