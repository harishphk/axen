# Axen

> The Universal, Configuration-Driven Package Manager for AI Agent Skills.

[![Go Report Card](https://goreportcard.com/badge/github.com/axen/axen)](https://goreportcard.com/report/github.com/axen/axen)
[![Documentation](https://img.shields.io/badge/docs-axen.dev-blueviolet)](https://axen.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](/LICENSE)
[![CI Status](https://img.shields.io/github/actions/workflow/status/axen/axen/ci.yml?branch=main)](https://github.com/axen/axen/actions)
[![Latest Release](https://img.shields.io/github/v/release/axen/axen)](https://github.com/axen/axen/releases)

---

## 💡 The Problem Axen Solves

The AI agent ecosystem is highly fragmented. Developers share instructions (`.cursorrules`, system prompts, agent tools, custom instructions) by copying and pasting markdown files or scripts. When you use Cursor at work, Windsurf at home, and Roo Cline in your terminal, you are forced to manage multiple proprietary folder structures manually. When prompt creators release updates, you miss them.

**Axen** solves this fragmentation by acting as a decentralized package manager for AI Agent Skills. It acts as a bridge between git-based/local skill repositories and the specific configuration paths where 60+ different AI assistants expect them. Write/maintain your skills once, install and keep them synced across all your tools instantly.

```text
┌─────────────────────────┐
│  Decentralized Sources  │ ── (Git Repos, Local Folders, HTTP URLs)
└────────────┬────────────┘
             │
             │ axen source add <url|path>
             ▼
      ┌─────────────┐
      │  AXEN CLI   │ ◄─── Config-Driven Manifest (axen.json)
      └──────┬──────┘
             │
             │ axen install
             ▼
┌─────────────────────────┐
│  60+ Universal Targets  │ ── (Cursor, Windsurf, Copilot, Roo, etc.)
└─────────────────────────┘
```

---

## ✨ Features

*   **Multi-Source Support**: Fetch and install skills directly from remote Git repositories or local directories. Add multiple sources to aggregate instructions from different teams or authors.
*   **Config-Based Tracking**: Axen maintains state through a declarative `axen.json` manifest and an `axen-lock.json` lockfile. This ensures repeatable, idempotent installations and easy configuration sharing across teams.
*   **Intelligent Auto-Detection for 60+ Tools**: Automatically resolves installation paths across Windows, macOS, and Linux for Cursor, Windsurf, Copilot, Roo Cline, and dozens of other AI editors. You don't need to configure where skills go—Axen already knows.
*   **Single-Command Updates**: Keep your entire instruction library up to date across all tools. Running `axen update` fetches the latest upstream commits from all sources, deploys new changes, and intelligently prunes removed files.
*   **Skill Scaffolding**: Use `axen create <name>` to instantly scaffold a new skill package with a standardized layout (`SKILL.md`, `scripts/`, `references/`) ready to be shared.
*   **Granular Control & Conflict Resolution**: Choose exactly which skills go to which editors using `--skills` and `--targets` flags. Handle source overlaps gracefully with interactive or automated conflict resolution.
*   **Safe Previews**: Support for `--dry-run` across all destructive commands (`install`, `remove`, `update`) to let you preview exactly which files will be added or deleted.
*   **Built-in Diagnostics**: Run `axen doctor` to instantly check folder permissions, lockfile integrity, and identify orphaned skill directories in your environment.

---

## 🎯 Popular Supported Tools

Axen natively maps paths for 60+ AI tools across Unix/macOS and Windows, sorted by popularity:

*   **GitHub Copilot** (`.copilot/skills/`)
*   **Cursor Editor** (`.cursor/skills/`)
*   **Windsurf Editor** (`.windsurf/skills/`)
*   **Continue.dev** (`.continue/skills/`)
*   **Claude CLI / Desktop** (`.claude/skills/`)
*   **Aider AI** (`.aider-desk/skills/`)
*   **Roo Cline (Roo Code)** (`.roo/skills/`)
*   **Devin** (`.config/devin/skills/`)
*   **OpenHands** (`.openhands/skills/`)
*   **Block Goose** (`.config/goose/skills/`)

👉 *For the complete list of all 60+ targets and their exact system paths, see the [Supported AI Tools Documentation](https://axen.dev/reference/supported-tools).*

---

## 🚀 Installation

### macOS & Linux
Install Axen globally using our quick-installation script:
```bash
curl -fsSL https://axen.dev/install.sh | sh
```

### Windows (PowerShell)
```powershell
irm https://axen.dev/install.ps1 | iex
```

### Pre-Built Binaries
You can also download the pre-compiled binary for your operating system and architecture directly from our [GitHub Releases](https://github.com/axen/axen/releases) page and place it inside your system path.

### Installing from Source (Go 1.22+ required)
If you prefer to compile the tool yourself from source:
```bash
go install github.com/axen/axen/cmd/axen@latest
```

---

## 🛠 Quick Start

### 1. Initialize Axen configuration
Generate an empty project configuration manifest `axen.json` in your repository:
```bash
axen init
```

### 2. Add a Skill Source
Add a remote git repository containing agent skills to your local registry:
```bash
axen source add https://github.com/example/agent-skills.git
```

### 3. Install Skills
Install skills into your detected AI agent targets interactively (or specify them via flags):
```bash
axen install agent-skills
```
*Tip: Run with `-d` / `--dry-run` first to preview the installation plan safely!*

### 4. List Installed Skills
View all installed skills and their corresponding target installations:
```bash
axen list
```

### 5. Update Skills
Keep your skills up-to-date with upstream changes:
```bash
axen update agent-skills
```

---

## 📄 Manifest & State Reference

### Manifest (`axen.json`)
The manifest file defines the skills available in a source registry:
```json
{
  "axen_version": "1",
  "name": "my-skills",
  "targets": ["cursor", "windsurf"],
  "skills": {
    "deploy-to-vercel": {
      "path": "skills/deploy-to-vercel/SKILL.md",
      "version": "1.0.0"
    }
  }
}
```

### Lockfile (`axen-lock.json`)
Axen writes a lockfile to track installation targets, hashes, and source versions, facilitating clean updates and prunes:
```json
{
  "version": 1,
  "skills": {
    "deploy-to-vercel": {
      "source": "https://github.com/example/agent-skills.git",
      "sourceType": "git",
      "skillPath": "skills/deploy-to-vercel/SKILL.md",
      "computedHash": "03e0eaaa9bf13ba1e7ffa387f5893de6f324c0868c"
    }
  }
}
```

---

## 📖 Learn More
*   [Full Documentation Site](https://axen.dev)
*   [Core Concepts Guide](https://axen.dev/guides/core-concepts)
*   [Configuration Schema Reference](https://axen.dev/guides/configuration)

---

## 🤝 Contributing & Community
We welcome contributions! Please review our:
- [Contributing Guidelines](/CONTRIBUTING.md) for local dev setup and commit styles.
- [Code of Conduct](/CODE_OF_CONDUCT.md) to understand community standards.

## 📄 License
This project is licensed under the MIT License. See [LICENSE](/LICENSE) for details.
