package cli

import (
	"axen/internal/core"
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/utils"
	"context"
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdSource(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage skill repositories (sources)",
	}

	addCmd := &cobra.Command{
		Use:   "add [url|path]",
		Short: "Add a skill repository to your local registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			installFlag, _ := cmd.Flags().GetBool("install")
			nameFlag, _ := cmd.Flags().GetString("name")
			return runSourceAdd(cmd.Context(), deps, args[0], installFlag, nameFlag)
		},
	}
	addCmd.Flags().BoolP("install", "i", false, "Install all skills immediately after adding the source")
	addCmd.Flags().StringP("name", "n", "", "Override the auto-derived namespace name")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered skill sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourceList()
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove [namespace]",
		Short: "Remove a source repository and all its skills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			ns := ""
			if len(args) > 0 {
				ns = args[0]
			}
			opts := RunRemoveOptions{
				All:            true,
				IsSourceRemove: true,
			}
			return runRemove(cmd.Context(), deps, ns, opts)
		},
	}

	cmd.AddCommand(addCmd, listCmd, removeCmd)
	return cmd
}

func runSourceList() error {
	lockfile, err := core.ReadLockfile()
	if err != nil {
		return err
	}

	if len(lockfile.Namespaces) == 0 {
		utils.Warn("No sources registered. Use `axen source add <url>` to add one.")
		return nil
	}

	tableData := pterm.TableData{
		{"Namespace", "Source", "Type", "Installed Skills"},
	}

	for name, entry := range lockfile.Namespaces {
		installedCount := len(entry.Skills.Installed)
		tableData = append(tableData, []string{
			name,
			entry.Source,
			entry.Type,
			fmt.Sprintf("%d", installedCount),
		})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	return nil
}

func runSourceAdd(ctx context.Context, deps *Dependencies, source string, installFlag bool, customName string) error {
	namespaceName := customName
	if namespaceName == "" {
		namespaceName = resolvers.DeriveNamespace(source)
	}

	spinner, _ := utils.StartSpinner("Fetching " + source + "...")
	fetchResult, manifest, err := core.FetchAndResolve(ctx, source, namespaceName)
	if err != nil {
		spinner.Fail(err.Error())
		return err
	}
	spinner.Success("Fetched " + namespaceName)

	lockfile, _ := core.ReadLockfile()
	if _, exists := lockfile.Namespaces[namespaceName]; !exists {
		lockfile.Namespaces[namespaceName] = models.NamespaceEntry{
			Type:      string(fetchResult.Type),
			Source:    source,
			Ref:       fetchResult.Ref,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			Targets:   manifest.Targets,
			Skills: models.NamespaceSkills{
				Installed: make(map[string]models.LockfileSkill),
			},
		}
		_ = core.WriteLockfile(lockfile)
	}

	// Update sources cache
	cache, _ := core.ReadSourcesIndex()
	cacheNs := models.CacheNamespace{Available: make(map[string]models.AvailableSkill)}
	for skillName, entry := range manifest.Skills {
		cacheNs.Available[skillName] = models.AvailableSkill{Path: entry.Path, Version: entry.Version}
	}
	cache.Namespaces[namespaceName] = cacheNs
	_ = core.WriteSourcesIndex(cache)

	utils.Success("Successfully added source %s!", pterm.Cyan(namespaceName))

	// Chain into the interactive installer
	opts := RunInstallOptions{AllSkills: installFlag, SkipAutoDetect: !installFlag}
	return runInstall(ctx, deps, namespaceName, opts)
}
