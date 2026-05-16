---
title: Core Concepts
description: Understand how Axen works under the hood.
---

## Skills

A **Skill** is the fundamental unit in Axen. It is a directory containing a `SKILL.md` file (which holds the instructions for the AI) and optional subdirectories like `scripts/`, `references/`, and `assets/`.

## Bundles

A **Bundle** is a named collection of skills defined in the manifest (`axen.json`). Instead of installing skills individually, you can subscribe to a bundle to install all of them at once. Manifests can also designate a bundle as `is_default`, meaning it will be auto-installed when the source is registered.

## Sources

A **Source** is a Git repository (`https://`, `git@`) or a local directory that contains a collection of skills. 

## Namespaces

A **Namespace** is the logical identifier for a source. For example, pulling from `https://github.com/org/go-skills.git` automatically assigns it the namespace `go-skills`.

## Targets

**Targets** are the destination directories where different AI agents expect their skills to reside. 
Axen comes with over **50+ built-in targets** covering both Windows and Unix conventions, including:
- Cursor (`.cursor/rules`)
- Claude (`~/Library/Application Support/Claude/skills`)
- GitHub Copilot
- Roo
- Windsurf
- Aider
- OpenHands

## Manifest (`axen.json`)

The manifest sits in the root of a skill source directory, defining the namespace, skills available, and optionally enforced targets.

## Lockfile (`axen-lock.json`)

Axen maintains a lockfile in `~/.axen/axen-lock.json`. It tracks all registered namespaces, installed skills, Git refs, updated timestamps, and their respective target installation paths. This ensures Axen knows exactly what is installed and where.
