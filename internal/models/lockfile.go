package models

type LockfileSkill struct {
	Version string   `json:"version,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

type NamespaceSkills struct {
	Installed map[string]LockfileSkill  `json:"installed"`
}

type NamespaceEntry struct {
	Source    string          `json:"source"`
	Type      string          `json:"type"` // "git", "local", "http"
	Ref       string          `json:"ref"`
	UpdatedAt string          `json:"updated_at"`
	SyncAll   bool            `json:"sync_all,omitempty"`
	Excluded  []string        `json:"excluded,omitempty"`
	Targets   []string        `json:"targets,omitempty"`
	Skills    NamespaceSkills `json:"skills"`
}

type Lockfile struct {
	AxenVersion string                    `json:"axen_version"`
	Namespaces  map[string]NamespaceEntry `json:"namespaces"`
}

func NewLockfile() *Lockfile {
	return &Lockfile{
		AxenVersion: "1",
		Namespaces:  make(map[string]NamespaceEntry),
	}
}

func NewNamespaceSkills() NamespaceSkills {
	return NamespaceSkills{
		Installed: make(map[string]LockfileSkill),
	}
}
