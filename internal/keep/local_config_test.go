package keep

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLocalConfig(t *testing.T, dir string, f File) {
	t.Helper()
	path := filepath.Join(dir, ".keep", "config.json")
	if err := WriteConfigFile(path, f); err != nil {
		t.Fatal(err)
	}
}

func TestLocalConfigIgnoredWithEnvAPIKey(t *testing.T) {
	root := t.TempDir()
	writeLocalConfig(t, root, File{
		APIURL: "https://evil.example",
		Access: "write",
		APIKey: "local-key",
	})

	oldWarn := LocalConfigWarn
	var warnings []string
	LocalConfigWarn = func(msg string) { warnings = append(warnings, msg) }
	defer func() { LocalConfigWarn = oldWarn }()

	origWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	getenv := func(k string) string {
		switch k {
		case "KEEP_API_KEY":
			return "env-key"
		case "KEEP_API_URL":
			return "https://trusted.example"
		default:
			return ""
		}
	}
	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "env-key" {
		t.Fatalf("APIKey=%q", cfg.APIKey)
	}
	if cfg.APIURL != "https://trusted.example" {
		t.Fatalf("APIURL=%q", cfg.APIURL)
	}
	mode, _ := EffectiveAccess(getenv)
	if mode != "read" {
		t.Fatalf("access=%q", mode)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning about ignored local fields")
	}
	if !strings.Contains(warnings[0], "api_url") || !strings.Contains(warnings[0], "access") {
		t.Fatalf("warning=%q", warnings[0])
	}
}

func TestLocalConfigIgnoredWithHomeConfigAPIKey(t *testing.T) {
	root := t.TempDir()
	homeDir := filepath.Join(root, "home")
	configDir := filepath.Join(homeDir, "keep")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteConfigFile(filepath.Join(configDir, "config.json"), File{
		APIKey: "home-key",
		APIURL: "https://home.example",
	}); err != nil {
		t.Fatal(err)
	}
	writeLocalConfig(t, root, File{
		APIURL: "https://evil.example",
		Access: "write",
	})

	oldWarn := LocalConfigWarn
	var warnings []string
	LocalConfigWarn = func(msg string) { warnings = append(warnings, msg) }
	defer func() { LocalConfigWarn = oldWarn }()

	origWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	getenv := func(k string) string {
		if k == "KEEP_CONFIG_DIR" {
			return configDir
		}
		return ""
	}
	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "home-key" || cfg.APIURL != "https://home.example" {
		t.Fatalf("cfg=%+v", cfg)
	}
	mode, _ := EffectiveAccess(getenv)
	if mode != "read" {
		t.Fatalf("access=%q", mode)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning")
	}
}

func TestLocalConfigAPIKeyWhenNoTrustedKey(t *testing.T) {
	root := t.TempDir()
	writeLocalConfig(t, root, File{
		APIKey: "project-key",
		APIURL: "https://evil.example",
	})

	oldWarn := LocalConfigWarn
	var warnings []string
	LocalConfigWarn = func(msg string) { warnings = append(warnings, msg) }
	defer func() { LocalConfigWarn = oldWarn }()

	origWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	cfg, err := LoadConfig(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "project-key" {
		t.Fatalf("APIKey=%q", cfg.APIKey)
	}
	if cfg.APIURL != DefaultBaseURL {
		t.Fatalf("APIURL=%q want default", cfg.APIURL)
	}
	if len(warnings) == 0 || !strings.Contains(warnings[0], "api_url") {
		t.Fatalf("warnings=%v", warnings)
	}
}
