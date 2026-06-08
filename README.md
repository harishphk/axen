# ⚡ Axen

> **A configuration-driven CLI to manage and sync AI Agent Skills across multiple targets and sources.**

[![Go Report Card](https://goreportcard.com/badge/github.com/harishphk/axen)](https://goreportcard.com/report/github.com/harishphk/axen)
[![Documentation](https://img.shields.io/badge/docs-axen.dev-blueviolet)](https://axen.domains.workers.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue)](/LICENSE)
[![CI Status](https://github.com/harishphk/axen/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/harishphk/axen/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/harishphk/axen)](https://github.com/harishphk/axen/releases)

---

Axen allows you to subscribe to multiple skill registries (sources like Git repositories, local directories, or HTTP URLs) and deploy them automatically to the configuration folders of 60+ AI assistants (targets like Cursor, Windsurf, Roo Code, and more).

```bash
# 1. Install Axen (macOS & Linux)
curl -fsSL https://raw.githubusercontent.com/harishphk/axen/main/install.sh | sh

# 2. Add a skill registry
axen source add https://github.com/example/agent-skills.git

# 3. Deploy to all your editors automatically
axen install
```

---

## 💡 Why Axen?

Managing AI Agent Skills across multiple editors and projects is frustrating:

*   **🧩 Target Fragmentation:** Every AI agent target expects skill packages in different directories (e.g., `.cursor/skills/`, `.windsurf/skills/`, or `.roo/skills/`).
*   **✍️ Manual Deployment:** Copying and maintaining skill packages across multiple workspaces, machines, and tools is tedious.
*   **🥀 Stale Skills:** When skill creators release updates, your locally installed skills become outdated because there is no automated sync mechanism.

### The Solution
Axen is a CLI tool that acts as a decentralized package manager for AI Agent Skills. Define your skill sources once, and Axen **automatically syncs and deploys your skills across all your local AI agent targets instantly**.

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

*   **⚡ Multi-Source Support**: Fetch and install skills directly from remote Git repositories, local directories, or HTTP URLs. Add multiple sources to aggregate registries.
*   **⚙️ Config-Based Tracking**: Axen maintains state through a declarative `axen.json` manifest and an `axen-lock.json` lockfile. This ensures repeatable, idempotent installations and easy configuration sharing.
*   **🔍 Intelligent Auto-Detection**: Automatically resolves installation paths across Windows, macOS, and Linux for Cursor, Windsurf, Roo Code, and 60+ other AI agents. No path configuration needed.
*   **🔄 Single-Command Updates**: Keep all your installed skills up to date. Running `axen update` fetches the latest upstream changes, deploys them, and prunes orphaned files.
*   **🛠 Skill Scaffolding**: Use `axen create <name>` to instantly scaffold a new skill package with a standardized layout (`SKILL.md`, `scripts/`, `references/`) ready to share.
*   **🎛 Granular Control & Conflict Resolution**: Choose exactly which skills go to which targets using `--skills` and `--targets` flags, and handle source overlaps gracefully with interactive conflict prompts.
*   **🧪 Safe Previews**: Support for `--dry-run` across all destructive commands (`install`, `remove`, `update`) to let you preview exactly which files will be added or deleted.
*   **🩺 Built-in Diagnostics**: Run `axen doctor` to instantly check folder permissions, lockfile integrity, and identify orphaned skill directories in your environment.

---

## 🎯 Popular Supported Tools

Axen natively maps paths for **60+ AI tools** across Unix/macOS and Windows, sorted by popularity:

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

👉 *For the complete list of all 60+ targets and their exact system paths, see the [Supported AI Tools Documentation](https://axen.domains.workers.dev/reference/supported-tools).*

---

## 🚀 Installation

### macOS & Linux
Install Axen globally using our quick-installation script:
```bash
curl -fsSL https://raw.githubusercontent.com/harishphk/axen/main/install.sh | sh
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/harishphk/axen/main/install.ps1 | iex
```

### Installing from Source (Go 1.26+ required)
```bash
go install github.com/harishphk/axen/cmd/axen@latest
```

---

## 🏁 Quick Start (For Users)

Get started using Axen to download and manage AI agent skills in under 30 seconds.

### 1. Add a Skill Source
Add a remote git repository containing agent skills to your local registry:
```bash
axen source add https://github.com/example/agent-skills.git
```

### 2. Install Skills
Install skills into your detected AI agent targets. Axen will automatically detect which editors (Cursor, Windsurf, etc.) you have installed:
```bash
axen install agent-skills
```
*Tip: Run with `-d` / `--dry-run` first to preview the installation plan safely!*

### 3. List Installed Skills
View all active sources, installed skills, and their corresponding target installation directories:
```bash
axen list
```

### 4. Sync & Update
Keep all your installed prompt libraries up to date with upstream changes:
```bash
axen update
```

---

## 🛠 For Skill Creators (Publishing Skills)

If you want to package and distribute your own prompts, instructions, or rules using Axen:

### 1. Initialize a Skill Repository
Create an `axen.json` manifest in the root of your git repository:
```bash
axen init
```

### 2. Scaffold a New Skill
Scaffold a standardized skill folder structure (adds `SKILL.md`, `scripts/`, and `references/` directories):
```bash
axen create <skill-name>
```

### 3. Define the Manifest (`axen.json`)
List your skills and define installation bundles. Here is an example manifest structure:

```json
{
  "axen_version": "1",
  "name": "my-skills",
  "targets": ["cursor", "windsurf"],
  "skills": {
    "deploy-to-vercel": {
      "path": "skills/deploy-to-vercel/SKILL.md",
      "version": "1.0.0"
    },
    "deploy-to-netlify": {
      "path": "skills/deploy-to-netlify/SKILL.md",
      "version": "1.0.0"
    }
  },
  "bundles": {
    "deploy": {
      "description": "Skills for cloud deployments",
      "skills": ["deploy-to-vercel", "deploy-to-netlify"],
      "is_default": true
    }
  }
}
```

### Skills & Bundles

*   **Skills**: Individual components containing target configurations and paths to a `SKILL.md` file.
*   **Bundles**: Groups of related skills. 
    *   If a bundle has `"is_default": true`, Axen automatically installs it when running `axen install <source>` without flags.
    *   You can install specific bundles using the `-b` / `--bundle` flag: `axen install agent-skills -b deploy`.
    *   You can remove specific bundles using: `axen remove agent-skills -b deploy`.

---

## 📄 Manifest & State Reference

<details>
<summary><b>View lockfile structure (<code>axen-lock.json</code>)</b></summary>

Axen writes a local lockfile to track installation targets and source namespaces, facilitating clean updates and prunes:
```json
{
  "axen_version": "1",
  "namespaces": {
    "agent-skills": {
      "source": "https://github.com/example/agent-skills.git",
      "type": "git",
      "ref": "main",
      "updated_at": "2026-05-30T12:00:00Z",
      "sync_all": true,
      "skills": {
        "installed": {
          "deploy-to-vercel": {
            "version": "1.0.0",
            "targets": [
              "cursor",
              "windsurf"
            ]
          }
        }
      }
    }
  }
}
```
</details>

---

## 📖 Learn More

For deep dives into Axen's architecture and advanced usage:
*   [Full Documentation Site](https://axen.domains.workers.dev/)
*   [Core Concepts Guide](https://axen.domains.workers.dev/guides/core-concepts)
*   [Configuration Schema Reference](https://axen.domains.workers.dev/guides/configuration)

---

## 🛠 Local Development

For developers working on the Axen CLI codebase, a `Makefile` is provided in the root directory to simplify local tasks:

```bash
make build          # Compile the CLI runner to ./bin/axen
make test           # Run Go unit tests
make test-e2e       # Run testscript E2E integration tests
make lint           # Execute golangci-lint
make gosec          # Execute local security scans
make clean          # Remove ./bin/ and coverage files
```

---

## 🤝 Contributing & Community

We welcome contributions! Please review our:
- [Contributing Guidelines](/CONTRIBUTING.md) for local dev setup and commit styles.
- [Code of Conduct](/CODE_OF_CONDUCT.md) to understand community standards.

## 📄 License

This project is licensed under the MIT License. See [LICENSE](/LICENSE) for details.
