---
title: Installation
description: How to install Axen on your machine.
---

## Prerequisites

- **Go 1.22+** (if compiling from source or using `go install`)

## Quick Installation

### macOS & Linux
Install Axen globally using our quick-installation script:
```bash
curl -fsSL https://raw.githubusercontent.com/harishphk/axen/main/install.sh | sh
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/harishphk/axen/main/install.ps1 | iex
```

## Installing from Source (Go 1.26+ required)

If you prefer to compile from source or use Go:

```bash
go install github.com/harishphk/axen/cmd/axen@latest
```

Ensure your Go bin directory is in your system's `$PATH`.

## Verifying Installation

Once installed, verify the installation by checking the version and running the `doctor` command:

```bash
axen --help
axen doctor
```
