package main

import (
	"github.com/harishphk/axen/internal/cli"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"
)

func main() {
	deps := cli.NewDependencies()
	rootCmd := cli.NewRootCmd(deps)
	rootCmd.DisableAutoGenTag = true

	outDir := "./docs/src/content/docs/commands"
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		log.Fatalf("failed to create docs directory: %v", err)
	}

	prepender := func(filename string) string {
		name := filepath.Base(filename)
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		title := strings.ReplaceAll(baseName, "_", " ")
		return fmt.Sprintf("---\ntitle: %s\ndescription: Reference for %s\n---\n\n", title, title)
	}

	linkHandler := func(name string) string {
		return name
	}

	err = doc.GenMarkdownTreeCustom(rootCmd, outDir, prepender, linkHandler)
	if err != nil {
		log.Fatalf("failed to generate markdown documentation: %v", err)
	}

	fmt.Println("Successfully generated CLI documentation in", outDir)
}
