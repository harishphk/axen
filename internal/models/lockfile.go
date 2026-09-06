package models

type LockfileSkill struct {
	Version string   `json:"version,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

type NamespaceSkills struct {
	Installed map[string]LockfileSkill `json:"installed"`
}

type NamespaceEntry struct {
	Source         string          `json:"source"`
	Type           string          `json:"type"` // "git", "local", "http"
	Ref            string          `json:"ref"`
	UpdatedAt      string          `json:"updated_at"`
	UpdatePolicy   string          `json:"update_policy,omitempty"`
	LastCheckedAt  string          `json:"last_checked_at,omitempty"`
	SyncAll        bool            `json:"sync_all,omitempty"`
	Excluded       []string        `json:"excluded,omitempty"`
	Targets        []string        `json:"targets,omitempty"`
	Bundles        []string        `json:"bundles,omitempty"`
	ExplicitSkills []string        `json:"explicit_skills,omitempty"`
	Skills         NamespaceSkills `json:"skills"`
}

type Lockfile struct {
	AxenVersion string                    `json:"axen_version"`
	Namespaces  map[string]NamespaceEntry `json:"namespaces"`
}

type UpdateCache struct {
	Namespaces              map[string]UpdateCacheEntry `json:"namespaces"`
	CliUpdate               CliUpdateCache              `json:"cli_update"`
	AutoUpdatedNamespaces   []string                    `json:"auto_updated_namespaces,omitempty"`
}

type UpdateCacheEntry struct {
	UpdateAvailable bool `json:"update_available"`
}

type CliUpdateCache struct {
	LastCheckedAt       string `json:"last_checked_at,omitempty"`
	LastNotifiedVersion string `json:"last_notified_version,omitempty"`
	PendingNotification string `json:"pending_notification,omitempty"`
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
