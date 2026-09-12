package models

type SkillEntry struct {
	Version      string   `json:"version,omitempty"`
	Path         string   `json:"path"`
	Targets      []string `json:"targets,omitempty"`
	PathOverride string   `json:"path_override,omitempty"`
}

type Bundle struct {
	Description string   `json:"description,omitempty"`
	Skills      []string `json:"skills"`
	IsDefault   bool     `json:"is_default,omitempty"`
}

type Manifest struct {
	AxenVersion string                `json:"axen_version"`
	Name        string                `json:"name"`
	Targets     []string              `json:"targets,omitempty"`
	Skills      map[string]SkillEntry `json:"skills"`
	Bundles     map[string]Bundle     `json:"bundles,omitempty"`
}

func NewManifest(name string) *Manifest {
	return &Manifest{
		AxenVersion: "1",
		Name:        name,
		Skills:      make(map[string]SkillEntry),
		Bundles:     make(map[string]Bundle),
	}
}

// HasDefaultBundle reports whether any bundle in the manifest is marked as default.
func (m *Manifest) HasDefaultBundle() bool {
	if m == nil {
		return false
	}
	for _, b := range m.Bundles {
		if b.IsDefault {
			return true
		}
	}
	return false
}
