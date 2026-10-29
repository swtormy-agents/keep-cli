//go:build unix

package keep

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigWarnsOnWorldReadableFile(t *testing.T) {
	old := configPermWarn
	defer func() { configPermWarn = old }()

	var warn bytes.Buffer
	configPermWarn = func(path string) {
		st, err := os.Stat(path)
		if err != nil {
			return
		}
		if st.Mode().Perm()&0o077 != 0 {
			warn.WriteString("world-readable:")
			warn.WriteString(path)
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"api_key":"dummy"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	getenv := func(k string) string {
		if k == "KEEP_CONFIG" {
			return path
		}
		return ""
	}
	if _, err := LoadConfig(getenv); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn.String(), "world-readable:") {
		t.Fatalf("expected permission warning, got %q", warn.String())
	}
}
