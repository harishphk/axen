---
title: Quick Start
description: Get up and running with Axen in minutes.
---

## 1. Add a Skill Source

A source is a Git repository or local directory containing a collection of skills. Add your first source:

```bash
axen source add https://github.com/example/my-skills.git
```

This registers the namespace (e.g. `my-skills`) in your local registry.

## 2. Install Skills

Install the skills from the source into your AI agent's targets (Cursor, Claude, Copilot, etc.):

```bash
# Interactive mode
axen install my-skills

# Unattended/All skills
axen install my-skills --all
```

## 3. Manage Your Skills

You can view your installed skills across all targets:

```bash
axen list
```

## 4. Keep Them Updated

When the upstream repository changes, easily fetch the latest updates:

```bash
axen update my-skills
```
