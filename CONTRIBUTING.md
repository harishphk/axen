# Contributing to Axen

First off, thank you for taking the time to contribute to Axen! Contributions from the community help make Axen the best package manager for AI Agent Skills.

This document provides guidelines and workflows for contributing to the Axen codebase.

---

## 🛠 Local Development Setup

To build and run Axen locally, you need the following prerequisites installed on your system:
*   [Go 1.22+](https://go.dev/doc/install)
*   [Git](https://git-scm.com/)

### Steps:
1.  **Fork and clone** the repository:
    ```bash
    git clone https://github.com/<your-username>/axen.git
    cd axen
    ```
2.  **Build the CLI tool**:
    ```bash
    go build -o bin/axen cmd/axen/main.go
    ```
3.  **Run the CLI locally**:
    ```bash
    ./bin/axen --help
    ```
4.  **Run all unit tests**:
    ```bash
    go test ./...
    ```

---

## 🎨 Adding a New AI Target

Axen comes with built-in path resolution for over 60 AI assistants. If you find an editor, assistant, or tool that is missing, adding it is easy:

1.  Open the [internal/models/config.go](file:///Users/p.kumar/Developer/projects/prod/axen/internal/models/config.go) file.
2.  Add your target to both maps:
    *   `DefaultTargetsWindows` (with Windows-specific paths like `~/AppData/Roaming/...` or `~/.tool/skills/`)
    *   `DefaultTargetsUnix` (with Unix/macOS-specific paths like `~/.tool/skills/`)
3.  Format the file (`go fmt ./internal/models/...`).
4.  Run the docs extraction script in the repository root to automatically regenerate the documentation table:
    ```bash
    go run .github/scripts/extract_targets.go  # or the scratch script path
    ```
5.  Commit the changes and open a pull request!

---

## 🌿 Branch Naming Conventions

To keep our repository organized, please name your branches using the following structured prefixes:

*   `feat/description` for new features or target additions (e.g., `feat/add-new-editor-target`).
*   `fix/description` for bug fixes (e.g., `fix/state-reconciler-panic`).
*   `docs/description` for documentation-only changes.
*   `refactor/description` for code refactoring without behavior changes.

*Note: Please use clean, hyphen-separated description tags and avoid generic branch names like `patch-1` or `my-feature`.*

---

## 📝 Commit Message Guidelines

We follow the **Conventional Commits** specification for commit messages. This helps automate release notes and changelogs.

Format:
```
<type>(<scope>): <short description>
```

### Allowed Types:
*   `feat`: A new feature (e.g., `feat(cli): add dry-run support`).
*   `fix`: A bug fix (e.g., `fix(parser): ignore malformed YAML frontmatter`).
*   `docs`: Documentation changes only.
*   `style`: Code style modifications (formatting, white-space, missing semi-colons).
*   `refactor`: Code changes that neither fix a bug nor add a feature.
*   `test`: Adding missing tests or correcting existing tests.
*   `ci`: Modifications to workflows, configurations, or CI scripts.

---

## 🚀 Pull Request Guidelines

Before submitting your pull request, please ensure you complete the following checklist:

1.  **Format your code**: Run `go fmt ./...` to format all Go source code.
2.  **Run tests**: Ensure `go test ./...` passes. Write test cases for new logic where appropriate.
3.  **Keep PRs focused**: Each PR should address a single concern. Avoid bundling unrelated fixes or features together.
4.  **Fill out the Pull Request Template**: Describe the changes, link relevant issues, and list how you verified the changes.

Thank you again for contributing!
