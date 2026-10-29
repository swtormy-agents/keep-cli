package keep

import (
	"os"
	"path/filepath"
)

// EffectiveAccess resolves write policy: KEEP_ACCESS env, then ~/.config/keep/access
// (or $KEEP_CONFIG_DIR/access), then "access" in config.json (first matching file in search order).
func EffectiveAccess(getenv func(string) string) (mode, source string) {
	if v := getenv("KEEP_ACCESS"); v != "" {
		return v, "KEEP_ACCESS"
	}
	dir, err := configDirFrom(getenv)
	if err == nil {
		b, err := os.ReadFile(filepath.Join(dir, "access"))
		if err == nil {
			m := string(bytesTrimSpace(b))
			if m != "" {
				return m, "access file"
			}
		}
	}
	for _, path := range configPaths(getenv) {
		f, err := readConfigFile(path)
		if err == nil && f.Access != "" {
			return f.Access, path
		}
	}
	return "read", "default"
}
