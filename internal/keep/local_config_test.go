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
	isolateHomeConfig(t, root)
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

	getenv := isolatedGetenv(t, root)
	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "project-key" {
		t.Fatalf("APIKey=%q", cfg.APIKey)
	}
	if cfg.APIURL != DefaultBaseURL {
		t.Fatalf("APIURL=%q want default", cfg.APIURL)
	}
	var sawIgnore, sawKeyUse bool
	for _, w := range warnings {
		if strings.Contains(w, "api_url") {
			sawIgnore = true
		}
		if strings.Contains(w, "using api_key from project-local config") {
			sawKeyUse = true
		}
		if strings.Contains(w, "project-key") {
			t.Fatalf("warning must not contain api key: %q", w)
		}
	}
	if !sawIgnore || !sawKeyUse {
		t.Fatalf("warnings=%v", warnings)
	}
}

func TestLocalConfigMalformedSkipped(t *testing.T) {
	root := t.TempDir()
	isolateHomeConfig(t, root)
	keepDir := filepath.Join(root, ".keep")
	if err := os.MkdirAll(keepDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keepDir, "config.json"), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}

	oldWarn := LocalConfigWarn
	var warnings []string
	LocalConfigWarn = func(msg string) { warnings = append(warnings, msg) }
	defer func() { LocalConfigWarn = oldWarn }()

	origWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	cfg, err := LoadConfig(isolatedGetenv(t, root))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.APIKey != "" {
		t.Fatalf("APIKey=%q", cfg.APIKey)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "skipping unreadable project-local config") {
		t.Fatalf("warnings=%v", warnings)
	}
	if strings.Contains(warnings[0], "not-json") {
		t.Fatalf("warning leaked file contents: %q", warnings[0])
	}
}

func isolateHomeConfig(t *testing.T, root string) {
	t.Helper()
	home := filepath.Join(root, "home")
	configDir := filepath.Join(home, "keep")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KEEP_CONFIG_DIR", configDir)
}

func isolatedGetenv(t *testing.T, root string) func(string) string {
	t.Helper()
	configDir := filepath.Join(root, "home", "keep")
	return func(k string) string {
		if k == "KEEP_CONFIG_DIR" {
			return configDir
		}
		return ""
	}
}
