package sources

import (
	"axen/internal/utils"
	"context"
	"strings"
)

type SourceType string

const (
	SourceTypeGit         SourceType = "git"
	SourceTypeLocal       SourceType = "local"
	SourceTypeHttp        SourceType = "http"
	SourceTypeUnsupported SourceType = "unsupported"
)

type FetchResult struct {
	LocalPath string
	Ref       string
	Type      SourceType
}

func DetectSourceType(source string) SourceType {
	if strings.HasPrefix(source, "ext::") {
		return SourceTypeUnsupported
	}
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "git://") || strings.HasPrefix(source, "git@") || strings.HasSuffix(source, ".git") {
		return SourceTypeGit
	}
	return SourceTypeLocal
}

func FetchSource(ctx context.Context, source string, namespaceName string) (*FetchResult, error) {
	srcType := DetectSourceType(source)

	switch srcType {
	case SourceTypeGit:
		localPath, ref, err := FetchGit(ctx, source, namespaceName)
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
	case SourceTypeUnsupported:
		return nil, utils.NewSourceError("Unsupported source format", source)
	default:
		return nil, utils.NewSourceError("Unknown source type", source)
	}
}
