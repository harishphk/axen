package models

import (
	"fmt"
	"regexp"
	"strings"
)

// skillNameRegex matches kebab-case names: lowercase alphanumeric with single hyphens.
var skillNameRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// ValidateFrontmatter validates a Frontmatter struct against the agentskills.io spec.
func ValidateFrontmatter(f *Frontmatter) error {
	if f.Name == "" {
		return fmt.Errorf("frontmatter: name is required")
	}
	if len(f.Name) > 64 {
		return fmt.Errorf("frontmatter: name must be at most 64 characters, got %d", len(f.Name))
	}
	if !skillNameRegex.MatchString(f.Name) {
		return fmt.Errorf("frontmatter: name %q must be lowercase kebab-case (a-z, 0-9, hyphens)", f.Name)
	}
	if strings.Contains(f.Name, "--") {
		return fmt.Errorf("frontmatter: name %q must not contain consecutive hyphens", f.Name)
	}
	if f.Description == "" {
		return fmt.Errorf("frontmatter: description is required")
	}
	if len(f.Description) > 1024 {
		return fmt.Errorf("frontmatter: description must be at most 1024 characters, got %d", len(f.Description))
	}
	if f.Compatibility != "" && len(f.Compatibility) > 500 {
		return fmt.Errorf("frontmatter: compatibility must be at most 500 characters, got %d", len(f.Compatibility))
	}
	return nil
}

// ValidateManifest validates a Manifest struct.
func ValidateManifest(m *Manifest) error {
	if m.AxenVersion == "" {
		return fmt.Errorf("manifest: axen_version is required")
	}
	if m.AxenVersion != "1" {
		return fmt.Errorf("manifest: unsupported axen_version %q (expected \"1\")", m.AxenVersion)
	}
	if m.Name == "" {
		return fmt.Errorf("manifest: name is required")
	}
	return nil
}

// ValidateLockfile validates a Lockfile struct.
func ValidateLockfile(l *Lockfile) error {
	if l.AxenVersion == "" {
		return fmt.Errorf("lockfile: axen_version is required")
	}
	if l.AxenVersion != "1" {
		return fmt.Errorf("lockfile: unsupported axen_version %q (expected \"1\")", l.AxenVersion)
	}
	validTypes := map[string]bool{"git": true, "local": true, "http": true}
	for name, entry := range l.Namespaces {
		if !validTypes[entry.Type] {
			return fmt.Errorf("lockfile: namespace %q has invalid type %q (expected git, local, or http)", name, entry.Type)
		}
		if entry.Source == "" {
			return fmt.Errorf("lockfile: namespace %q has empty source", name)
		}
	}
	return nil
}
