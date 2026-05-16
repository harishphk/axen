---
title: Configuration
description: How to customize your Axen installation.
---

Axen allows you to customize its behavior and add new agent targets using its global configuration file.

## `axen-config.json`

The global configuration file is located at `~/.axen/axen-config.json`. 

You can use it to override default target paths or add entirely new custom AI agent targets that aren't natively supported yet.

### Example Configuration

```json
{
  "custom_targets": {
    "my-new-agent": {
      "path": "~/.config/my-new-agent/skills",
      "os": "unix"
    }
  }
}
```
