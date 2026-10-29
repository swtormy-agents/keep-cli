package keep

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectiveAccessEnvWins(t *testing.T) {
	getenv := func(k string) string {
		if k == "KEEP_ACCESS" {
			return "write"
		}
		return ""
	}
	mode, src := EffectiveAccess(getenv)
	if mode != "write" || src != "KEEP_ACCESS" {
		t.Fatalf("got %q %q", mode, src)
	}
}

func TestEffectiveAccessFromAccessFileInConfigDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "access"), []byte("write"), 0o600); err != nil {
		t.Fatal(err)
	}
	getenv := func(k string) string {
		if k == "KEEP_CONFIG_DIR" {
			return dir
		}
		return ""
	}
	mode, src := EffectiveAccess(getenv)
	if mode != "write" || src != "access file" {
		t.Fatalf("got %q %q", mode, src)
	}
}
