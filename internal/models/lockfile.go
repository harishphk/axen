package models

type LockfileSkill struct {
	Version string   `json:"version,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

type NamespaceEntry struct {
	Source    string                   `json:"source"`
	Type      string                   `json:"type"` // "git", "local", "http"
	Ref       string                   `json:"ref"`
	UpdatedAt string                   `json:"updated_at"`
	Targets   []string                 `json:"targets,omitempty"`
	Skills    map[string]LockfileSkill `json:"skills"`
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
