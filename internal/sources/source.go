package sources

import (
	"axen/internal/utils"
	"strings"
)

type SourceType string

const (
	SourceTypeGit   SourceType = "git"
	SourceTypeLocal SourceType = "local"
	SourceTypeHttp  SourceType = "http"
)

type FetchResult struct {
	LocalPath string
	Ref       string
	Type      SourceType
}

func DetectSourceType(source string) SourceType {
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "git@") || strings.HasSuffix(source, ".git") {
		return SourceTypeGit
	}
	return SourceTypeLocal
}

func FetchSource(source string, namespaceName string) (*FetchResult, error) {
	srcType := DetectSourceType(source)

	switch srcType {
	case SourceTypeGit:
		localPath, ref, err := FetchGit(source, namespaceName)
		if err != nil {
			return nil, err
		}
		return &FetchResult{LocalPath: localPath, Ref: ref, Type: SourceTypeGit}, nil
	case SourceTypeLocal:
		localPath, ref, err := FetchLocal(source)
		if err != nil {
			return nil, err
		}
		return &FetchResult{LocalPath: localPath, Ref: ref, Type: SourceTypeLocal}, nil
	case SourceTypeHttp:
		return nil, utils.NewSourceError("HTTP/ZIP sources are not yet supported (coming in v1.1)", source)
	default:
		return nil, utils.NewSourceError("Unknown source type", source)
	}
}
