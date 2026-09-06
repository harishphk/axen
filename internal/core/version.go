package core

// Version is the current version of the Axen CLI.
// It is injected at build time by GoReleaser via ldflags (-X github.com/harishphk/axen/internal/core.Version={{.Tag}}).
// During local development, it defaults to "dev".
var Version = "dev"
