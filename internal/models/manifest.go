package models

type SkillEntry struct {
	Version      string   `json:"version,omitempty"`
	Path         string   `json:"path"`
	Targets      []string `json:"targets,omitempty"`
	PathOverride string   `json:"path_override,omitempty"`
}

type Manifest struct {
	AxenVersion string                `json:"axen_version"`
	Name        string                `json:"name"`
	Targets     []string              `json:"targets,omitempty"`
	Skills      map[string]SkillEntry `json:"skills"`
}

func NewManifest(name string) *Manifest {
	return &Manifest{
		AxenVersion: "1",
		Name:        name,
		Skills:      make(map[string]SkillEntry),
	}
}
