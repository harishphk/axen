package core

import (
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/sources"
	"context"
)

// FetchAndResolve handles the common flow of fetching a source repository,
// checking for an existing manifest, and generating one if it doesn't exist.
func FetchAndResolve(ctx context.Context, sourceURL, namespace string) (*sources.FetchResult, *models.Manifest, error) {
	fetchResult, err := sources.FetchSource(ctx, sourceURL, namespace)
	if err != nil {
		return nil, nil, err
	}

	if HasManifest(fetchResult.LocalPath) {
		manifest, err := ReadManifest(fetchResult.LocalPath)
		return fetchResult, manifest, err
	}

	scanned, err := ScanSkills(fetchResult.LocalPath)
	if err != nil {
		return nil, nil, err
	}

	manifest := GenerateManifest(namespace, scanned, nil)
	return fetchResult, manifest, nil
}
