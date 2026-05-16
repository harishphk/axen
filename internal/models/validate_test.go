package models

import (
	"strings"
	"testing"
)

func TestValidateFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		fm      Frontmatter
		wantErr string
	}{
		{
			name:    "valid frontmatter",
			fm:      Frontmatter{Name: "my-skill", Description: "A great skill"},
			wantErr: "",
		},
		{
			name:    "empty name",
			fm:      Frontmatter{Name: "", Description: "desc"},
			wantErr: "name is required",
		},
		{
			name:    "name too long",
			fm:      Frontmatter{Name: strings.Repeat("a", 65), Description: "desc"},
			wantErr: "at most 64 characters",
		},
		{
			name:    "name with spaces",
			fm:      Frontmatter{Name: "my skill", Description: "desc"},
			wantErr: "lowercase kebab-case",
		},
		{
			name:    "name with uppercase",
			fm:      Frontmatter{Name: "MySkill", Description: "desc"},
			wantErr: "lowercase kebab-case",
		},
		{
			name:    "name with consecutive hyphens",
			fm:      Frontmatter{Name: "my--skill", Description: "desc"},
			wantErr: "consecutive hyphens",
		},
		{
			name:    "name starting with hyphen",
			fm:      Frontmatter{Name: "-skill", Description: "desc"},
			wantErr: "lowercase kebab-case",
		},
		{
			name:    "name ending with hyphen",
			fm:      Frontmatter{Name: "skill-", Description: "desc"},
			wantErr: "lowercase kebab-case",
		},
		{
			name:    "empty description",
			fm:      Frontmatter{Name: "skill", Description: ""},
			wantErr: "description is required",
		},
		{
			name:    "description too long",
			fm:      Frontmatter{Name: "skill", Description: strings.Repeat("a", 1025)},
			wantErr: "at most 1024 characters",
		},
		{
			name:    "compatibility too long",
			fm:      Frontmatter{Name: "skill", Description: "desc", Compatibility: strings.Repeat("a", 501)},
			wantErr: "at most 500 characters",
		},
		{
			name:    "single char name",
			fm:      Frontmatter{Name: "a", Description: "desc"},
			wantErr: "",
		},
		{
			name:    "numeric name",
			fm:      Frontmatter{Name: "s3", Description: "desc"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFrontmatter(&tt.fm)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name    string
		m       Manifest
		wantErr string
	}{
		{
			name:    "valid manifest",
			m:       Manifest{AxenVersion: "1", Name: "my-skills", Skills: make(map[string]SkillEntry)},
			wantErr: "",
		},
		{
			name:    "empty axen_version",
			m:       Manifest{AxenVersion: "", Name: "my-skills"},
			wantErr: "axen_version is required",
		},
		{
			name:    "unsupported version",
			m:       Manifest{AxenVersion: "2", Name: "my-skills"},
			wantErr: "unsupported axen_version",
		},
		{
			name:    "empty name",
			m:       Manifest{AxenVersion: "1", Name: ""},
			wantErr: "name is required",
		},
		{
			name: "multiple default bundles",
			m: Manifest{
				AxenVersion: "1",
				Name:        "my-skills",
				Skills: map[string]SkillEntry{
					"s1": {Path: "."},
				},
				Bundles: map[string]Bundle{
					"b1": {Skills: []string{"s1"}, IsDefault: true},
					"b2": {Skills: []string{"s1"}, IsDefault: true},
				},
			},
			wantErr: "multiple default bundles defined",
		},
		{
			name: "missing skill in bundle",
			m: Manifest{
				AxenVersion: "1",
				Name:        "my-skills",
				Skills: map[string]SkillEntry{
					"s1": {Path: "."},
				},
				Bundles: map[string]Bundle{
					"b1": {Skills: []string{"s1", "s2"}, IsDefault: false},
				},
			},
			wantErr: "references missing skill \"s2\"",
		},
		{
			name: "valid bundles",
			m: Manifest{
				AxenVersion: "1",
				Name:        "my-skills",
				Skills: map[string]SkillEntry{
					"s1": {Path: "."},
					"s2": {Path: "."},
				},
				Bundles: map[string]Bundle{
					"b1": {Skills: []string{"s1", "s2"}, IsDefault: true},
					"b2": {Skills: []string{"s1"}, IsDefault: false},
				},
			},
			wantErr: "",
		},
		{
			name: "invalid skill name in manifest keys (path traversal)",
			m: Manifest{
				AxenVersion: "1",
				Name:        "my-skills",
				Skills: map[string]SkillEntry{
					"s1":          {Path: "."},
					"../../s2":    {Path: "."},
				},
			},
			wantErr: "lowercase kebab-case",
		},
		{
			name: "invalid skill name in manifest keys (uppercase)",
			m: Manifest{
				AxenVersion: "1",
				Name:        "my-skills",
				Skills: map[string]SkillEntry{
					"InvalidName": {Path: "."},
				},
			},
			wantErr: "lowercase kebab-case",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateManifest(&tt.m)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestValidateLockfile(t *testing.T) {
	tests := []struct {
		name    string
		l       Lockfile
		wantErr string
	}{
		{
			name: "valid lockfile",
			l: Lockfile{
				AxenVersion: "1",
				Namespaces: map[string]NamespaceEntry{
					"test": {Source: "https://github.com/test/repo.git", Type: "git"},
				},
			},
			wantErr: "",
		},
		{
			name:    "empty axen_version",
			l:       Lockfile{AxenVersion: "", Namespaces: make(map[string]NamespaceEntry)},
			wantErr: "axen_version is required",
		},
		{
			name:    "unsupported version",
			l:       Lockfile{AxenVersion: "99", Namespaces: make(map[string]NamespaceEntry)},
			wantErr: "unsupported axen_version",
		},
		{
			name: "invalid namespace type",
			l: Lockfile{
				AxenVersion: "1",
				Namespaces: map[string]NamespaceEntry{
					"test": {Source: "ftp://bad", Type: "ftp"},
				},
			},
			wantErr: "invalid type",
		},
		{
			name: "empty source",
			l: Lockfile{
				AxenVersion: "1",
				Namespaces: map[string]NamespaceEntry{
					"test": {Source: "", Type: "git"},
				},
			},
			wantErr: "empty source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLockfile(&tt.l)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}
