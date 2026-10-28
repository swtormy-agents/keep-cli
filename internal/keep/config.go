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
	APIURL string
	APIKey string
	Access string
	Source string
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
	if v := getenv("KEEP_ACCESS"); v != "" {
		cfg.Access = v
	}
	for _, path := range configPaths(getenv) {
		f, err := readConfigFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Config{}, err
		}
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
		if cfg.Access == "read" && f.Access != "" {
			cfg.Access = f.Access
		}
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultBaseURL
	}
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
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "keep", "config.json"))
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

func ReadAccessMode(getenv func(string) string) string {
	if v := getenv("KEEP_ACCESS"); v != "" {
		return v
	}
	dir, err := ConfigDir()
	if err != nil {
		return "read"
	}
	b, err := os.ReadFile(filepath.Join(dir, "access"))
	if err != nil {
		return "read"
	}
	m := string(bytesTrimSpace(b))
	if m == "" {
		return "read"
	}
	return m
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
