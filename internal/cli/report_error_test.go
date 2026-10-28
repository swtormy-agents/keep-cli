package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/swtormy-agents/keep-cli/internal/keep"
)

// TestReportErrorRedactsAPIKey verifies stderr JSON redaction in reportError alone
// (APIError.Message is intentionally not pre-redacted by the client layer).
func TestReportErrorRedactsAPIKey(t *testing.T) {
	const secret = "report-layer-secret-key"
	var errOut bytes.Buffer
	a := &app{
		cfg:    keep.Config{APIKey: secret},
		errOut: &errOut,
	}
	apiErr := &keep.APIError{Status: 401, Message: "rejected token " + secret}
	code := a.reportError(&exitError{
		code: exitAuth,
		kind: "auth",
		err:  errors.New("keep API HTTP 401"),
		api:  apiErr,
	})
	if code != exitAuth {
		t.Fatalf("code=%d", code)
	}
	if strings.Contains(errOut.String(), secret) {
		t.Fatalf("reportError leaked key: %s", errOut.String())
	}
	if !strings.Contains(errOut.String(), "<API_KEY>") {
		t.Fatalf("expected redaction placeholder in: %s", errOut.String())
	}
}
