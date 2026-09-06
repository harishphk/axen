package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/utils"
	"github.com/pterm/pterm"
)

func ReadUpdateCache() (*models.UpdateCache, error) {
	cachePath := resolvers.GetUpdateCachePath()
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.UpdateCache{Namespaces: make(map[string]models.UpdateCacheEntry)}, nil
		}
		return nil, err
	}

	var cache models.UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return &models.UpdateCache{Namespaces: make(map[string]models.UpdateCacheEntry)}, nil
	}

	if cache.Namespaces == nil {
		cache.Namespaces = make(map[string]models.UpdateCacheEntry)
	}
	return &cache, nil
}

func WriteUpdateCache(cache *models.UpdateCache) error {
	return utils.WriteJson(resolvers.GetUpdateCachePath(), cache)
}

func CheckAndNotifyUpdates() {
	cache, err := ReadUpdateCache()
	if err != nil {
		return
	}

	modified := false

	if len(cache.AutoUpdatedNamespaces) > 0 {
		for _, nsName := range cache.AutoUpdatedNamespaces {
			pterm.Success.Printf("✨ Auto-updated %s in the background.\n", nsName)
		}
		cache.AutoUpdatedNamespaces = nil
		modified = true
	}

	// Notify about skill updates
	for nsName, entry := range cache.Namespaces {
		if entry.UpdateAvailable {
			utils.Info("Updates are available for %s. Run 'axen update %s' to apply.", pterm.Cyan(nsName), nsName)
			delete(cache.Namespaces, nsName)
			modified = true
		}
	}

	// Notify about CLI updates
	if cache.CliUpdate.PendingNotification != "" {
		utils.Info("A new version of Axen (%s) is available! Run 'axen upgrade' to install it.", pterm.Green(cache.CliUpdate.PendingNotification))
		cache.CliUpdate.LastNotifiedVersion = cache.CliUpdate.PendingNotification
		cache.CliUpdate.PendingNotification = ""
		modified = true
	}

	if modified {
		_ = WriteUpdateCache(cache)
	}
}

func shouldCheckDaily(lastChecked string) bool {
	if lastChecked == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, lastChecked)
	if err != nil {
		return true
	}
	return time.Since(t) > 24*time.Hour
}

func shouldCheckWeekly(lastChecked string) bool {
	if lastChecked == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, lastChecked)
	if err != nil {
		return true
	}
	return time.Since(t) > 7*24*time.Hour
}

func TriggerOpportunisticUpdates(ctx context.Context) {
	if os.Getenv("AXEN_TEST_HOME") != "" {
		return
	}

	lockfile, err := ReadLockfile()
	if err != nil {
		return
	}

	needsDetachedCheck := false

	for _, entry := range lockfile.Namespaces {
		policy := entry.UpdatePolicy
		if policy == "" {
			policy = "daily"
		}

		if (policy == "daily" && shouldCheckDaily(entry.LastCheckedAt)) ||
			(policy == "weekly" && shouldCheckWeekly(entry.LastCheckedAt)) {
			needsDetachedCheck = true
			break
		}
	}

	if !needsDetachedCheck {
		config, _ := ReadConfig()
		cache, _ := ReadUpdateCache()
		if config != nil && config.CheckForUpdates && cache != nil && shouldCheckDaily(cache.CliUpdate.LastCheckedAt) {
			needsDetachedCheck = true
		}
	}

	if needsDetachedCheck {
		spawnDetachedCheck()
	}
}

func spawnDetachedCheck() {
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	// #nosec G702
	cmd := exec.Command(exe, "_internal_check_updates")
	setSysProcAttr(cmd)
	_ = cmd.Start()
}

func PerformBackgroundUpdateChecks() {
	lockfile, err := ReadLockfile()
	if err != nil {
		return
	}
	cache, err := ReadUpdateCache()
	if err != nil {
		return
	}

	modified := false
	ctx := context.Background()

	for nsName, entry := range lockfile.Namespaces {
		policy := entry.UpdatePolicy
		if policy == "" {
			policy = "daily"
		}

		// Manual sources are never updated or checked in the background.
		// They are only updated when the user explicitly runs `axen update`.
		if policy == "manual" {
			continue
		}

		if (policy == "daily" && shouldCheckDaily(entry.LastCheckedAt)) || (policy == "weekly" && shouldCheckWeekly(entry.LastCheckedAt)) {
			fetchResult, manifest, err := FetchAndResolve(ctx, entry.Source, nsName)
			if err != nil {
				continue
			}

			entry.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)
			lockfile.Namespaces[nsName] = entry
			_ = WriteLockfile(lockfile)

			if fetchResult.Ref != entry.Ref {
				intent := GetIntent(lockfile, nsName)
				result, err := Reconcile(ctx, lockfile, nsName, entry.Source, fetchResult, manifest, intent, ReconcileOpts{
					Force:            true,
					UpdateMode:       true,
					ConflictStrategy: "keep",
				})
				if err == nil && len(result.Installed) > 0 {
					cache.AutoUpdatedNamespaces = append(cache.AutoUpdatedNamespaces, nsName)
					modified = true
				}
			}
		}
	}

	// Check for CLI updates (throttled daily)
	config, _ := ReadConfig()
	if config.CheckForUpdates {
		if cliModified := checkForCliUpdate(cache); cliModified {
			modified = true
		}
	}

	if modified {
		_ = WriteUpdateCache(cache)
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// FetchLatestCliReleaseTag retrieves the latest release tag from GitHub,
// falling back to inspecting the redirect header if API rate-limited.
func FetchLatestCliReleaseTag(ctx context.Context) (string, error) {
	// 1. Try standard GitHub API
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/harishphk/axen/releases/latest", nil)
	if err == nil {
		req.Header.Set("User-Agent", "axen-cli/"+Version)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode == http.StatusOK {
				var release githubRelease
				if err := json.NewDecoder(resp.Body).Decode(&release); err == nil && release.TagName != "" {
					return release.TagName, nil
				}
			}
		}
	}

	// 2. Fallback: HEAD request to releases/latest to follow Location redirect without API rate-limiting
	headReq, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com/harishphk/axen/releases/latest", nil)
	if err != nil {
		return "", err
	}
	headReq.Header.Set("User-Agent", "axen-cli/"+Version)
	noRedirectClient := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	headResp, err := noRedirectClient.Do(headReq)
	if err != nil {
		return "", err
	}
	defer func() { _ = headResp.Body.Close() }()

	loc := headResp.Header.Get("Location")
	if loc != "" {
		parts := strings.Split(strings.TrimRight(loc, "/"), "/")
		tag := parts[len(parts)-1]
		if strings.HasPrefix(tag, "v") {
			return tag, nil
		}
	}

	return "", fmt.Errorf("could not determine latest release tag")
}

func checkForCliUpdate(cache *models.UpdateCache) bool {
	if Version == "dev" {
		return false
	}

	if !shouldCheckDaily(cache.CliUpdate.LastCheckedAt) {
		return false
	}

	tag, err := FetchLatestCliReleaseTag(context.Background())
	if err != nil {
		return false
	}

	cache.CliUpdate.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)

	latestVersion := tag
	currentVersion := Version
	notifiedVersion := cache.CliUpdate.LastNotifiedVersion

	if IsNewerVersion(latestVersion, currentVersion) && IsNewerVersion(latestVersion, notifiedVersion) {
		cache.CliUpdate.PendingNotification = tag
	}

	return true
}

// IsNewerVersion compares two semver strings (with or without 'v' prefix)
// and returns true if v1 is newer than v2.
func IsNewerVersion(v1, v2 string) bool {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")
	if v1 == "" {
		return false
	}
	if v2 == "" || v2 == "dev" {
		return true
	}

	p1 := strings.Split(v1, ".")
	p2 := strings.Split(v2, ".")

	maxLen := len(p1)
	if len(p2) > maxLen {
		maxLen = len(p2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(p1) {
			seg := strings.Split(p1[i], "-")[0]
			for _, ch := range seg {
				if ch >= '0' && ch <= '9' {
					n1 = n1*10 + int(ch-'0')
				}
			}
		}
		if i < len(p2) {
			seg := strings.Split(p2[i], "-")[0]
			for _, ch := range seg {
				if ch >= '0' && ch <= '9' {
					n2 = n2*10 + int(ch-'0')
				}
			}
		}
		if n1 > n2 {
			return true
		}
		if n1 < n2 {
			return false
		}
	}
	return false
}
