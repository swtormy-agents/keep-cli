package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed SKILL.md
var skillMD string

var skills = map[string]*string{"keep": &skillMD}

func skillNames() []string { return []string{"keep"} }

var skillDirs = map[string]string{
	"claude": ".claude/skills",
	"cursor": ".cursor/skills",
	"grok":   ".grok/skills",
	"agents": ".agents/skills",
}

func (a *app) skillCmd() *cobra.Command {
	skill := &cobra.Command{
		Use:   "skill",
		Short: "Print or install the agent skill bundled in this binary",
	}
	skill.AddCommand(&cobra.Command{
		Use:   "print [keep]",
		Short: "Print SKILL.md to stdout (default: keep)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _, err := pickSkill(args)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(a.out, body)
			return err
		},
	})

	var agent, dir string
	var global bool
	install := &cobra.Command{
		Use:   "install [keep]",
		Short: "Write SKILL.md for an agent (project-local by default)",
		Example: `  keep skill install
  keep skill install --global
  keep skill install --agent cursor --global
  keep skill install --dir ./my/skills/keep`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, name, err := pickSkill(args)
			if err != nil {
				return err
			}
			target := dir
			if target == "" {
				sub, ok := skillDirs[agent]
				if !ok {
					return usageErr("--agent must be one of claude, cursor, grok, agents")
				}
				base, err := os.Getwd()
				if global {
					base, err = os.UserHomeDir()
				}
				if err != nil {
					return runtimeErr(err)
				}
				target = filepath.Join(base, sub, name)
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return runtimeErr(err)
			}
			file := filepath.Join(target, "SKILL.md")
			if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
				return runtimeErr(err)
			}
			fmt.Fprintln(a.out, file)
			return nil
		},
	}
	install.Flags().StringVar(&agent, "agent", "claude", "claude, cursor, grok or agents")
	install.Flags().BoolVar(&global, "global", false, "install into the home directory")
	install.Flags().StringVar(&dir, "dir", "", "exact target directory")
	skill.AddCommand(install)
	return skill
}

func pickSkill(args []string) (body, name string, err error) {
	name = "keep"
	if len(args) == 1 {
		name = args[0]
	}
	p, ok := skills[name]
	if !ok {
		return "", "", usageErr("unknown skill %q (available: %s)", name, strings.Join(skillNames(), ", "))
	}
	return *p, name, nil
}
