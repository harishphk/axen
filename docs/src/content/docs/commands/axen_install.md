---
title: axen install
description: Reference for axen install
---

## axen install

Install skills from a registered source or local directory

```
axen install [namespace] [flags]
```

### Options

```
  -a, --all                        Install all skills, bypassing interactive prompt
  -c, --conflict-strategy string   Conflict resolution strategy: prompt, overwrite, keep (default "prompt")
  -d, --dry-run                    Preview changes without executing
  -f, --force                      Overwrite existing skills on conflict
  -h, --help                       help for install
  -s, --skills strings             Comma-separated list of specific skills to install
  -t, --targets strings            Comma-separated list of targets to install into
```

### Options inherited from parent commands

```
  -v, --verbose   Enable verbose/debug logging
```

### SEE ALSO

* [axen](axen.md)	 - Axen - A minimal skill manager for AI agents

