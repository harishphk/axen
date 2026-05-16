package utils

import "fmt"

type AxenError struct {
	Message string
	Code    string
}

func (e *AxenError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewAxenError(msg, code string) *AxenError {
	return &AxenError{Message: msg, Code: code}
}

func NewManifestError(msg string) *AxenError {
	return &AxenError{Message: msg, Code: "MANIFEST_ERROR"}
}

func NewFrontmatterError(msg, skillPath string) *AxenError {
	return &AxenError{Message: fmt.Sprintf("%s (at %s)", msg, skillPath), Code: "FRONTMATTER_ERROR"}
}

func NewConflictError(skillName, existingNamespace, newNamespace string) *AxenError {
	msg := fmt.Sprintf("Skill %q already installed from %q. New source: %q. Use --force to overwrite.", skillName, existingNamespace, newNamespace)
	return &AxenError{Message: msg, Code: "CONFLICT_ERROR"}
}

func NewSourceError(msg, source string) *AxenError {
	return &AxenError{Message: fmt.Sprintf("%s (source: %s)", msg, source), Code: "SOURCE_ERROR"}
}

func NewFileSystemError(msg, path string) *AxenError {
	return &AxenError{Message: fmt.Sprintf("%s (path: %s)", msg, path), Code: "FS_ERROR"}
}
