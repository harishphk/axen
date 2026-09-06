package sources

import (
	"github.com/harishphk/axen/internal/utils"
	"context"
	"path/filepath"
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
	LocalPath      string
	Ref            string
	Type           SourceType
	ResolvedSource string
}

func DetectSourceType(source string) SourceType {
	if strings.HasPrefix(source, "ext::") {
		return SourceTypeUnsupported
	}
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "git://") || strings.HasPrefix(source, "git@") || strings.HasSuffix(source, ".git") {
		return SourceTypeGit
	}
	if strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") || filepath.IsAbs(source) {
		return SourceTypeLocal
	}
	return SourceTypeUnsupported
}

func IsGitHubShorthand(source string) bool {
	if strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") || strings.HasPrefix(source, "http") || strings.HasPrefix(source, "git") || filepath.IsAbs(source) {
		return false
	}

	cleanSource := strings.TrimSuffix(source, "/")
	parts := strings.Split(cleanSource, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" && !strings.Contains(cleanSource, " ") {
		return true
	}

	return false
}

func FetchSource(ctx context.Context, source string, namespaceName string) (*FetchResult, error) {
	originalSource := source
	
	cleanSource := strings.TrimSuffix(source, "/")
	isShorthand := IsGitHubShorthand(cleanSource)
	if isShorthand {
		source = "https://github.com/" + cleanSource + ".git"
	}

	srcType := DetectSourceType(source)

	switch srcType {
	case SourceTypeGit:
		localPath, ref, err := FetchGit(ctx, source, namespaceName)
		if err != nil {
			if isShorthand {
				return nil, utils.NewSourceError("Failed to fetch shorthand from GitHub. If this is not a GitHub repository, please provide the full URL. Original error: "+err.Error(), originalSource)
			}
			return nil, err
		}
		return &FetchResult{LocalPath: localPath, Ref: ref, Type: SourceTypeGit, ResolvedSource: source}, nil
	case SourceTypeLocal:
		localPath, ref, err := FetchLocal(source)
		if err != nil {
			return nil, err
		}
		return &FetchResult{LocalPath: localPath, Ref: ref, Type: SourceTypeLocal, ResolvedSource: localPath}, nil
	case SourceTypeHttp:
		return nil, utils.NewSourceError("HTTP/ZIP sources are not yet supported (coming in v1.1)", source)
	case SourceTypeUnsupported:
		return nil, utils.NewSourceError("Invalid source format. For GitHub, use 'owner/repo' or a full URL. For local directories, use a path starting with './', '/', or '~/'.", originalSource)
	default:
		return nil, utils.NewSourceError("Unknown source type", source)
	}
}
