package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra/doc"
	"github.com/stefanjarina/ginit/cmd"
)

func main() {
	// Parse command-line flags
	outputDir := flag.String("output", "docs", "Output directory for generated documentation")
	flag.Parse()

	// Get root command from main application
	rootCmd := cmd.GetRootCommand()

	// Create docs directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating docs directory: %v\n", err)
		os.Exit(1)
	}

	// Generate markdown docs
	markdownDir := filepath.Join(*outputDir, "cli")
	if err := os.MkdirAll(markdownDir, 0755); err != nil {
		fmt.Printf("Error creating markdown directory: %v\n", err)
		os.Exit(1)
	}
	if err := doc.GenMarkdownTree(rootCmd, markdownDir); err != nil {
		fmt.Printf("Error generating markdown: %v\n", err)
		os.Exit(1)
	}

	// Generate man pages
	manDir := filepath.Join(*outputDir, "man")
	if err := os.MkdirAll(manDir, 0755); err != nil {
		fmt.Printf("Error creating man directory: %v\n", err)
		os.Exit(1)
	}
	header := &doc.GenManHeader{
		Title:   "GINIT",
		Section: "1",
	}
	if err := doc.GenManTree(rootCmd, header, manDir); err != nil {
		fmt.Printf("Error generating man pages: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Documentation generated successfully")
	fmt.Printf("  - CLI reference: %s\n", markdownDir)
	fmt.Printf("  - Man pages: %s\n", manDir)
}
