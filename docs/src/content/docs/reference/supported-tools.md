---
title: Supported AI Tools & Targets
description: List of the 60+ pre-configured AI Agent targets supported natively by Axen.
---

Axen comes with built-in configurations for over **50+ AI agent development environments** and tools. It automatically detects your operating system and resolves the paths to install skills where they belong.

Below is the complete list of natively supported targets, along with their default installation paths on Unix-like systems and Windows.

| Target ID | Tool Name / Assistant | Default Path (Unix/macOS) | Default Path (Windows) |
| :--- | :--- | :--- | :--- |
| `adal` | **Adal** | `~/.adal/skills/` | `~/.adal/skills/` |
| `agents` | **Agents** | `~/.agents/skills/` | `~/.agents/skills/` |
| `aider` | **Aider AI** | `~/.aider-desk/skills/` | `~/.aider-desk/skills/` |
| `amp` | **Amp** | `~/.amp/skills/` | `~/.amp/skills/` |
| `antigravity` | **Antigravity Agent** | `~/.gemini/antigravity/skills/` | `~/.gemini/antigravity/skills/` |
| `augment` | **Augment** | `~/.augment/skills/` | `~/.augment/skills/` |
| `bob` | **Bob** | `~/.bob/skills/` | `~/.bob/skills/` |
| `claude` | **Claude CLI / Claude Desktop** | `~/.claude/skills/` | `~/AppData/Roaming/Claude/skills/` |
| `codearts` | **Codearts** | `~/.codeartsdoer/skills/` | `~/.codeartsdoer/skills/` |
| `codebuddy` | **Codebuddy** | `~/.codebuddy/skills/` | `~/.codebuddy/skills/` |
| `codemaker` | **Codemaker** | `~/.codemaker/skills/` | `~/.codemaker/skills/` |
| `codestudio` | **Codestudio** | `~/.codestudio/skills/` | `~/.codestudio/skills/` |
| `codex` | **Codex** | `~/.codex/skills/` | `~/.codex/skills/` |
| `commandcode` | **Commandcode** | `~/.commandcode/skills/` | `~/.commandcode/skills/` |
| `continue` | **Continue.dev** | `~/.continue/skills/` | `~/.continue/skills/` |
| `copilot` | **GitHub Copilot CLI / Extension** | `~/.copilot/skills/` | `~/.copilot/skills/` |
| `cortex` | **Snowflake Cortex** | `~/.snowflake/cortex/skills/` | `~/.snowflake/cortex/skills/` |
| `crush` | **Crush** | `~/.config/crush/skills/` | `~/AppData/Roaming/crush/skills/` |
| `cursor` | **Cursor Editor** | `~/.cursor/skills/` | `~/.cursor/skills/` |
| `deepagents` | **DeepAgents** | `~/.deepagents/agent/skills/` | `~/.deepagents/agent/skills/` |
| `devin` | **Devin CLI** | `~/.config/devin/skills/` | `~/AppData/Roaming/devin/skills/` |
| `dexto` | **Dexto** | `~/.dexto/` | `~/.dexto/` |
| `droid` | **Droid** | `~/.factory/skills/` | `~/.factory/skills/` |
| `firebender` | **Firebender** | `~/.firebender/skills/` | `~/.firebender/skills/` |
| `forgecode` | **Forgecode** | `~/.forge/skills/` | `~/.forge/skills/` |
| `gemini` | **Gemini CLI / Advanced** | `~/.gemini/skills/` | `~/.gemini/skills/` |
| `goose` | **Block Goose** | `~/.config/goose/skills/` | `~/AppData/Roaming/goose/skills/` |
| `hermes` | **Hermes** | `~/.hermes/skills/` | `~/.hermes/skills/` |
| `iflow` | **Iflow** | `~/.iflow/skills/` | `~/.iflow/skills/` |
| `junie` | **Junie** | `~/.junie/skills/` | `~/.junie/skills/` |
| `kilo` | **Kilo** | `~/.kilocode/skills/` | `~/.kilocode/skills/` |
| `kimi` | **Kimi** | `~/.config/agents/skills/` | `~/AppData/Roaming/agents/skills/` |
| `kiro` | **Kiro** | `~/.kiro/skills/` | `~/.kiro/skills/` |
| `kode` | **Kode** | `~/.kode/skills/` | `~/.kode/skills/` |
| `mcpjam` | **MCP Jam** | `~/.mcpjam/skills/` | `~/.mcpjam/skills/` |
| `mux` | **Mux** | `~/.mux/skills/` | `~/.mux/skills/` |
| `neovate` | **Neovate** | `~/.neovate/skills/` | `~/.neovate/skills/` |
| `opencode` | **OpenCode** | `~/.config/opencode/skills/` | `~/AppData/Roaming/opencode/skills/` |
| `openhands` | **OpenHands (formerly All-Hands)** | `~/.openhands/skills/` | `~/.openhands/skills/` |
| `pi` | **Pi** | `~/.pi/agent/skills/` | `~/.pi/agent/skills/` |
| `pochi` | **Pochi** | `~/.pochi/skills/` | `~/.pochi/skills/` |
| `qoder` | **Qoder** | `~/.qoder/skills/` | `~/.qoder/skills/` |
| `qwen` | **Qwen** | `~/.qwen/skills/` | `~/.qwen/skills/` |
| `roo` | **Roo Cline (Roo Code)** | `~/.roo/skills/` | `~/.roo/skills/` |
| `rovodev` | **Rovodev** | `~/.rovodev/skills/` | `~/.rovodev/skills/` |
| `tabnine` | **Tabnine** | `~/.tabnine/agent/skills/` | `~/.tabnine/agent/skills/` |
| `trae` | **Trae (ByteDance AI Editor)** | `~/.trae/skills/` | `~/.trae/skills/` |
| `trae-cn` | **Trae-cn** | `~/.trae-cn/skills/` | `~/.trae-cn/skills/` |
| `vibe` | **Vibe** | `~/.vibe/skills/` | `~/.vibe/skills/` |
| `warp` | **Warp Terminal** | `~/.warp/` | `~/.warp/` |
| `windsurf` | **Windsurf Editor (Codeium)** | `~/.windsurf/skills/` | `~/.windsurf/skills/` |
| `zencoder` | **Zencoder** | `~/.zencoder/skills/` | `~/.zencoder/skills/` |

---

## Overriding Default Paths & Adding Custom Targets

If you need to install skills to a different path for a specific target, or want to add a tool that isn't listed here yet, you can configure them in your global configuration file (`~/.config/axen/config.json` or `axen-config.json` depending on configuration).

See the [Configuration Guide](/guides/configuration) for detailed instructions on configuring custom targets.
