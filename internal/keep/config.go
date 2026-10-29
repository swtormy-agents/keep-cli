package keep

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// File holds non-secret defaults; api_key may be stored with mode 0600.
type File struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
	Access string `json:"access"` // read (default) or write
}

type Config struct {
	APIURL       string
	APIKey       string
	Access       string
	AccessSource string
	Source       string
}

func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{Access: "read"}
	if v := getenv("KEEP_API_URL"); v != "" {
		cfg.APIURL = v
		cfg.Source = "KEEP_API_URL"
	}
	if v := getenv("KEEP_API_KEY"); v != "" {
		cfg.APIKey = v
		if cfg.Source == "" {
			cfg.Source = "KEEP_API_KEY"
		}
	}
	for _, path := range configPaths(getenv) {
		f, err := readConfigFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Config{}, err
		}
		auditConfigFilePerm(path)
		if cfg.APIURL == "" && f.APIURL != "" {
			cfg.APIURL = f.APIURL
			cfg.Source = path
		}
		if cfg.APIKey == "" && f.APIKey != "" {
			cfg.APIKey = f.APIKey
			if cfg.Source == "" {
				cfg.Source = path
			}
		}
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultBaseURL
	}
	mode, src := EffectiveAccess(getenv)
	cfg.Access = mode
	cfg.AccessSource = src
	if cfg.Access == "" {
		cfg.Access = "read"
	}
	return cfg, nil
}

func configPaths(getenv func(string) string) []string {
	var paths []string
	if p := getenv("KEEP_CONFIG"); p != "" {
		paths = append(paths, p)
	}
	if dir, err := configDirFrom(getenv); err == nil {
		paths = append(paths, filepath.Join(dir, "config.json"))
	}
	paths = append(paths, ".keep/config.json")
	return paths
}

func readConfigFile(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

func ConfigDir() (string, error) {
	return configDirFrom(os.Getenv)
}

func configDirFrom(getenv func(string) string) (string, error) {
	if v := getenv("KEEP_CONFIG_DIR"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "keep"), nil
}

func WriteAccessMode(mode string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "access")
	return os.WriteFile(path, []byte(mode), 0o600)
}

// WriteConfigFile writes a config JSON with mode 0600 (parent dir 0700).
func WriteConfigFile(path string, f File) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}

func bytesTrimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\n' || b[0] == '\r' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 {
		c := b[len(b)-1]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			break
		}
		b = b[:len(b)-1]
	}
	return b
}
