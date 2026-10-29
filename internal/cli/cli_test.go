package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testEnv(key string) string {
	switch key {
	case "KEEP_API_KEY":
		return "test-key-12345"
	case "KEEP_API_URL":
		return ""
	case "KEEP_ACCESS":
		return "write"
	default:
		return ""
	}
}

func runCLI(t *testing.T, srvURL string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	getenv := testEnv
	if srvURL != "" {
		prev := getenv
		getenv = func(k string) string {
			if k == "KEEP_API_URL" {
				return srvURL
			}
			return prev(k)
		}
	}
	var out, errOut bytes.Buffer
	code = Execute(args, strings.NewReader(""), &out, &errOut, getenv)
	return out.String(), errOut.String(), code
}

func TestAlertsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key-12345" {
			t.Fatalf("bad key")
		}
		if r.URL.Path != "/alerts" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"fingerprint":"fp1","name":"CPU high"}]`))
	}))
	defer srv.Close()

	out, errOut, code := runCLI(t, srv.URL, "alerts", "list", "--limit", "5")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	var resp struct {
		Untrusted bool `json:"untrusted"`
		Result    []map[string]any
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("stdout: %s err %v", out, err)
	}
	if !resp.Untrusted || resp.Result[0]["fingerprint"] != "fp1" {
		t.Fatalf("got %s", out)
	}
}

func TestStatusWithoutKey(t *testing.T) {
	getenv := func(k string) string {
		if k == "KEEP_API_URL" {
			return "https://example.test"
		}
		return ""
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"status"}, strings.NewReader(""), &out, &errOut, getenv)
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, errOut.String())
	}
	var v map[string]any
	if err := json.Unmarshal(out.Bytes(), &v); err != nil || v["has_key"] != false {
		t.Fatalf("%s", out.String())
	}
}

func TestWriteBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not hit server")
	}))
	defer srv.Close()

	getenv := func(k string) string {
		switch k {
		case "KEEP_API_KEY":
			return "test-key-12345"
		case "KEEP_API_URL":
			return srv.URL
		case "KEEP_ACCESS":
			return "read"
		default:
			return ""
		}
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"alerts", "dismiss", "--fingerprint", "a", "--last-received", "t"}, strings.NewReader(""), &out, &errOut, getenv)
	if code != exitPolicy {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	var errJSON struct {
		OK    bool
		Error map[string]any
	}
	_ = json.Unmarshal(errOut.Bytes(), &errJSON)
	if errJSON.Error["type"] != "policy" {
		t.Fatalf("%s", errOut.String())
	}
}

func TestDryRun(t *testing.T) {
	out, errOut, code := runCLI(t, "", "alerts", "list", "--dry-run")
	if code != 0 {
		t.Fatalf("stderr %s", errOut)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil || v["method"] != "GET" {
		t.Fatalf("%s", out)
	}
}

func TestSkillPrint(t *testing.T) {
	out, _, code := runCLI(t, "", "skill", "print")
	if code != 0 || !strings.Contains(out, "name: keep") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestAPIHealthcheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthcheck" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	out, errOut, code := runCLI(t, srv.URL, "api", "GET", "/healthcheck")
	if code != 0 {
		t.Fatalf("stderr %s", errOut)
	}
	if !strings.Contains(out, `"status"`) {
		t.Fatalf("%s", out)
	}
}

func TestUsageError(t *testing.T) {
	_, errOut, code := runCLI(t, "", "alerts", "get")
	if code != exitUsage {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(errOut, `"ok":false`) {
		t.Fatalf("%s", errOut)
	}
}

func TestAuthErrorRedactsAPIKey(t *testing.T) {
	const secret = "test-key-12345"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"invalid key ` + secret + `"}`))
	}))
	defer srv.Close()

	getenv := func(k string) string {
		switch k {
		case "KEEP_API_KEY":
			return secret
		case "KEEP_API_URL":
			return srv.URL
		default:
			return ""
		}
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"status"}, strings.NewReader(""), &out, &errOut, getenv)
	if code != exitAuth {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
	if strings.Contains(errOut.String(), secret) {
		t.Fatalf("stderr leaked key: %s", errOut.String())
	}
}

func TestKeepAccessWriteNotice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	getenv := func(k string) string {
		switch k {
		case "KEEP_API_KEY":
			return "test-key-12345"
		case "KEEP_API_URL":
			return srv.URL
		case "KEEP_ACCESS":
			return "write"
		default:
			return ""
		}
	}
	var out, errOut bytes.Buffer
	code := Execute([]string{"alerts", "dismiss", "--fingerprint", "a", "--last-received", "t", "--dry-run"}, strings.NewReader(""), &out, &errOut, getenv)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "KEEP_ACCESS") {
		t.Fatalf("expected KEEP_ACCESS notice, stderr=%q", errOut.String())
	}
}
