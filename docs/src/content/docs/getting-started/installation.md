---
title: Installation
description: How to install Axen on your machine.
---

## Prerequisites

- **Go 1.22+** (if compiling from source or using `go install`)

## Using `go install` (Recommended)

The easiest way to install Axen is via Go:

```bash
go install github.com/harishphk/axen/cmd/axen@latest
```

Ensure your `$(go env GOPATH)/bin` directory is in your system's `$PATH`.

## Verifying Installation

Once installed, verify the installation by checking the version and running the `doctor` command:

```bash
axen --help
axen doctor
```
