package cli

import (
	"context"
	"fmt"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
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
			updatePolicy, _ := cmd.Flags().GetString("update-policy")
			if !cmd.Flags().Changed("update-policy") {
				cfg, _ := core.ReadConfig()
				if cfg.DefaultUpdatePolicy != "" {
					updatePolicy = cfg.DefaultUpdatePolicy
				}
			}
			if !models.IsValidUpdatePolicy(updatePolicy) {
				return fmt.Errorf("invalid update policy %q (must be daily, weekly, or manual)", updatePolicy)
			}

			return runSourceAdd(cmd.Context(), deps, args[0], installFlag, name, updatePolicy)
		},
	}
	addCmd.Flags().StringP("name", "n", "", "Custom name for the source namespace")
	addCmd.Flags().BoolP("install", "i", false, "Install all skills immediately after adding the source")
	addCmd.Flags().StringP("update-policy", "u", "daily", "Set the auto-update policy (daily, weekly, manual)")
	cmd.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered skill sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourceList(cmd.Context(), deps)
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

	cmd.AddCommand(&cobra.Command{
		Use:   "policy <name> <daily|weekly|manual>",
		Short: "Set the auto-update policy for a source",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			return runSourcePolicy(cmd.Context(), deps, args[0], args[1])
		},
	})

	return cmd
}

func runSourceAdd(ctx context.Context, deps *Dependencies, source string, installFlag bool, customName string, updatePolicy string) error {
	namespaceName := customName
	if namespaceName == "" {
		namespaceName = resolvers.DeriveNamespace(source)
	}

	var spinner *pterm.SpinnerPrinter
	inspectOpts := core.InspectOptions{
		OnFetchStart: func(ns string) {
			spinner, _ = utils.StartSpinner("Fetching " + source + "...")
		},
		OnFetchDone: func(ns string, err error) {
			if err != nil && spinner != nil {
				spinner.Fail("Failed to fetch")
			}
		},
	}

	sourceManifest, err := deps.Engine.Inspect(ctx, source, namespaceName, inspectOpts)
	if err != nil {
		return err
	}
	if spinner != nil {
		spinner.Success(fmt.Sprintf("Fetched %s (%s)", namespaceName, sourceManifest.FetchResult.ResolvedSource))
	}

	err = deps.Engine.AddSource(ctx, core.SourceAddRequest{
		NamespaceName: namespaceName,
		SourceURL:     sourceManifest.FetchResult.ResolvedSource,
		SourceType:    string(sourceManifest.FetchResult.Type),
		Ref:           sourceManifest.FetchResult.Ref,
		UpdatePolicy:  updatePolicy,
		Targets:       sourceManifest.Manifest.Targets,
		Manifest:      sourceManifest.Manifest,
	})
	if err != nil {
		return err
	}

	utils.Success("Successfully added source %s!", pterm.Cyan(namespaceName))

	// Chain into the interactive installer with pre-inspected manifest
	opts := RunInstallOptions{
		AllSkills:        installFlag,
		SkipAutoDetect:   false,
		SkipFetchSpinner: true,
		CachedManifest:   sourceManifest,
	}
	return runInstall(ctx, deps, namespaceName, opts)
}

func runSourcePolicy(ctx context.Context, deps *Dependencies, namespaceName string, policy string) error {
	if !models.IsValidUpdatePolicy(policy) {
		return utils.NewAxenError(fmt.Sprintf("invalid policy %q (must be daily, weekly, or manual)", policy), "INVALID_POLICY")
	}

	err := deps.Engine.SetSourcePolicy(ctx, namespaceName, policy)
	if err != nil {
		return err
	}

	utils.Success("Set auto-update policy for %s to %s", pterm.Cyan(namespaceName), pterm.Green(policy))
	return nil
}

func runSourceList(ctx context.Context, deps *Dependencies) error {
	sources, err := deps.Engine.ListSources(ctx)
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		utils.Warn("No sources registered. Use `axen source add <url>` to add one.")
		return nil
	}

	tableData := pterm.TableData{
		{"Namespace", "Source", "Type", "Installed Skills"},
	}

	for _, s := range sources {
		tableData = append(tableData, []string{
			s.Namespace,
			s.Source,
			s.Type,
			fmt.Sprintf("%d", s.InstalledSkillsCount),
		})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	return nil
}
