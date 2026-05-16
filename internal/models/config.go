package models

import "runtime"

var DefaultTargetsWindows = map[string]string{
	"claude":   "~/AppData/Roaming/Claude/skills/",
	"codex":    "~/.codex/skills/",
	"copilot":  "~/.copilot/skills/",
	"cursor":   "~/.cursor/skills/",
	"gemini":   "~/.gemini/skills/",
	"goose":    "~/AppData/Roaming/goose/skills/",
	"opencode": "~/AppData/Roaming/opencode/skills/",
	"roo":      "~/.roo/skills/",
	"trae":     "~/.trae/skills/",
	"windsurf": "~/.windsurf/skills/",
	"amp":      "~/.amp/skills/",
	"agents":   "~/.agents/skills/",
}

var DefaultTargetsUnix = map[string]string{
	"claude":   "~/.claude/skills/",
	"codex":    "~/.codex/skills/",
	"copilot":  "~/.copilot/skills/",
	"cursor":   "~/.cursor/skills/",
	"gemini":   "~/.gemini/skills/",
	"goose":    "~/.config/goose/skills/",
	"opencode": "~/.config/opencode/skills/",
	"roo":      "~/.roo/skills/",
	"trae":     "~/.trae/skills/",
	"windsurf": "~/.windsurf/skills/",
	"amp":      "~/.amp/skills/",
	"agents":   "~/.agents/skills/",
}

func GetDefaultTargets() map[string]string {
	if runtime.GOOS == "windows" {
		return DefaultTargetsWindows
	}
	return DefaultTargetsUnix
}

type Config struct {
	Targets         map[string]string `json:"targets"`
	DefaultTemplate *string           `json:"default_template"`
}
