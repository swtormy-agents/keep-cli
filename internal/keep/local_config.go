package keep

import "fmt"

const localConfigRelPath = ".keep/config.json"

// LocalConfigWarn is called when project-local .keep/config.json sets fields that are ignored.
var LocalConfigWarn func(message string)

func isLocalProjectConfig(path string) bool {
	return path == localConfigRelPath
}

func applyLocalConfigFile(cfg *Config, path string, f File) {
	var ignored []string
	if f.APIURL != "" {
		ignored = append(ignored, "api_url")
	}
	if f.Access != "" {
		ignored = append(ignored, "access")
	}
	if len(ignored) > 0 && LocalConfigWarn != nil {
		LocalConfigWarn(fmt.Sprintf(
			"keep: ignoring %v in %s (project-local config cannot set api_url or access)",
			ignored, path,
		))
	}
	if cfg.APIKey == "" && f.APIKey != "" {
		cfg.APIKey = f.APIKey
		if cfg.Source == "" {
			cfg.Source = path
		}
		if LocalConfigWarn != nil {
			LocalConfigWarn(fmt.Sprintf(
				"keep: using api_key from project-local config %s",
				path,
			))
		}
	}
}

func warnLocalConfigUnreadable(path string) {
	if LocalConfigWarn != nil {
		LocalConfigWarn(fmt.Sprintf("keep: skipping unreadable project-local config %s", path))
	}
}
