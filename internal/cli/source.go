package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/services"
	"github.com/harishphk/axen/internal/utils"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdSource(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage skill sources (add, remove, list)",
	}

	addCmd := &cobra.Command{
		Use:   "add <source>",
		Short: "Add a new source (GitHub repo or local path)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			name, _ := cmd.Flags().GetString("name")
			installFlag, _ := cmd.Flags().GetBool("install")

			return runSourceAdd(cmd.Context(), deps, args[0], installFlag, name, "")
		},
	}
	addCmd.Flags().StringP("name", "n", "", "Custom name for the source namespace")
	addCmd.Flags().BoolP("install", "i", false, "Install all skills immediately after adding the source")
	cmd.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered skill sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourceList()
		},
	}
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a source and all its installed skills",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			opts := RunRemoveOptions{
				All:            true,
				IsSourceRemove: true,
			}
			return runRemove(cmd.Context(), deps, args[0], opts)
		},
	})

	return cmd
}

func runSourceAdd(ctx context.Context, deps *Dependencies, source string, installFlag bool, customName string, updatePolicy string) error {
	namespaceName := customName
	if namespaceName == "" {
		namespaceName = resolvers.DeriveNamespace(source)
	}

	installSvc := &services.InstallService{}
	
	var spinner *pterm.SpinnerPrinter
	fetchOpts := services.FetchOptions{
		OnFetchStart: func(ns string) {
			spinner, _ = utils.StartSpinner("Fetching " + source + "...")
		},
		OnFetchDone: func(ns string, err error) {
			if err != nil {
				spinner.Fail("Failed to fetch")
			}
		},
	}

	fetchResult, manifest, err := installSvc.FetchManifest(ctx, source, namespaceName, fetchOpts)
	if err != nil {
		return err
	}
	if spinner != nil {
		spinner.Success(fmt.Sprintf("Fetched %s (%s)", namespaceName, fetchResult.ResolvedSource))
	}

	sourceSvc := &services.SourceService{}
	err = sourceSvc.Add(services.SourceAddRequest{
		NamespaceName: namespaceName,
		SourceURL:     fetchResult.ResolvedSource,
		SourceType:    string(fetchResult.Type),
		Ref:           fetchResult.Ref,
		UpdatePolicy:  updatePolicy,
		Targets:       manifest.Targets,
		Manifest:      manifest,
	})
	if err != nil {
		return err
	}

	utils.Success("Successfully added source %s!", pterm.Cyan(namespaceName))

	// Chain into the interactive installer
	opts := RunInstallOptions{
		AllSkills:        installFlag,
		SkipAutoDetect:   false,
		SkipFetchSpinner: true,
	}
	return runInstall(ctx, deps, namespaceName, opts)
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

	var names []string
	for name := range lockfile.Namespaces {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := lockfile.Namespaces[name]
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
