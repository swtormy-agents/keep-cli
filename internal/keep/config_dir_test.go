package keep

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := WriteConfigFile(cfgPath, File{APIKey: "from-dir", APIURL: "https://example.test"}); err != nil {
		t.Fatal(err)
	}
	getenv := func(k string) string {
		if k == "KEEP_CONFIG_DIR" {
			return dir
		}
		return ""
	}
	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "from-dir" || cfg.APIURL != "https://example.test" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestConfigDirEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KEEP_CONFIG_DIR", dir)
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("ConfigDir() = %q, want %q", got, dir)
	}
}
