package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	_, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	repoRoot := filepath.Join(".", "") // assume run from repo root

	configPath := filepath.Join(repoRoot, "internal", "models", "config.go")
	outputPath := filepath.Join(repoRoot, "docs", "src", "content", "docs", "reference", "supported-tools.md")

	contentBytes, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error: Could not find config.go at %s\n", configPath)
		os.Exit(1)
	}
	content := string(contentBytes)

	winRegex := regexp.MustCompile(`(?s)var DefaultTargetsWindows = map\[string\]string\{(.*?)\}`)
	unixRegex := regexp.MustCompile(`(?s)var DefaultTargetsUnix = map\[string\]string\{(.*?)\}`)

	winBlock := winRegex.FindStringSubmatch(content)
	unixBlock := unixRegex.FindStringSubmatch(content)

	if len(winBlock) < 2 || len(unixBlock) < 2 {
		fmt.Println("Error: Could not find DefaultTargetsWindows or DefaultTargetsUnix in config.go")
		os.Exit(1)
	}

	winMap := parseMap(winBlock[1])
	unixMap := parseMap(unixBlock[1])

	targetSet := make(map[string]bool)
	for k := range winMap {
		targetSet[k] = true
	}
	for k := range unixMap {
		targetSet[k] = true
	}

	var allTargets []string
	for k := range targetSet {
		allTargets = append(allTargets, k)
	}
	sort.Strings(allTargets)

	markdown := `---
title: Supported AI Tools & Targets
description: List of the 60+ pre-configured AI Agent targets supported natively by Axen.
---

Axen comes with built-in configurations for over **50+ AI agent development environments** and tools. It automatically detects your operating system and resolves the paths to install skills where they belong.

Below is the complete list of natively supported targets, along with their default installation paths on Unix-like systems and Windows.

| Target ID | Tool Name / Assistant | Default Path (Unix/macOS) | Default Path (Windows) |
| :--- | :--- | :--- | :--- |
`

	nameMappings := map[string]string{
		"claude":      "Claude CLI / Claude Desktop",
		"codex":       "Codex",
		"copilot":     "GitHub Copilot CLI / Extension",
		"cursor":      "Cursor Editor",
		"gemini":      "Gemini CLI / Advanced",
		"goose":       "Block Goose",
		"opencode":    "OpenCode",
		"roo":         "Roo Cline (Roo Code)",
		"trae":        "Trae (ByteDance AI Editor)",
		"windsurf":    "Windsurf Editor (Codeium)",
		"aider":       "Aider AI",
		"openhands":   "OpenHands (formerly All-Hands)",
		"devin":       "Devin CLI",
		"continue":    "Continue.dev",
		"tabnine":     "Tabnine",
		"warp":        "Warp Terminal",
		"antigravity": "Antigravity Agent",
		"deepagents":  "DeepAgents",
		"mcpjam":      "MCP Jam",
		"cortex":      "Snowflake Cortex",
	}

	for _, target := range allTargets {
		name, exists := nameMappings[target]
		if !exists {
			name = strings.ToUpper(target[:1]) + target[1:]
		}
		
		uPath, exists := unixMap[target]
		if !exists {
			uPath = "N/A"
		}
		
		wPath, exists := winMap[target]
		if !exists {
			wPath = "N/A"
		}
		
		markdown += fmt.Sprintf("| `%s` | **%s** | `%s` | `%s` |\n", target, name, uPath, wPath)
	}

	markdown += `
---

## Overriding Default Paths & Adding Custom Targets

If you need to install skills to a different path for a specific target, or want to add a tool that isn't listed here yet, you can configure them in your global configuration file (` + "`~/.config/axen/config.json`" + ` or ` + "`axen-config.json`" + ` depending on configuration).

See the [Configuration Guide](/guides/configuration) for detailed instructions on configuring custom targets.
`

	os.MkdirAll(filepath.Dir(outputPath), 0755)
	err = os.WriteFile(outputPath, []byte(markdown), 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated supported-tools.md successfully at:", outputPath)
}

func parseMap(blockText string) map[string]string {
	data := make(map[string]string)
	lines := strings.Split(blockText, "\n")
	re := regexp.MustCompile(`"([^"]+)":\s*"([^"]+)"`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		match := re.FindStringSubmatch(line)
		if len(match) == 3 {
			data[match[1]] = match[2]
		}
	}
	return data
}
