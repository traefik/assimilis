// Package main provides the Assimilis license report generator CLI entry point.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/traefik/assimilis/pkg/generator"
)

func main() {
	cfg := generator.DefaultConfig()

	version := flag.Bool("version", false, "Display version information.")

	flag.StringVar(&cfg.RepoName, "repo-name", cfg.RepoName, "Name of the repository.")

	flag.StringVar(&cfg.HTMLTemplatePath, "html-template", "", "Override HTML template path (default: embedded).")
	flag.StringVar(&cfg.NoticeTplPath, "notice-template", "", "Override NOTICE template path (default: embedded).")

	flag.StringVar(&cfg.SPDXVersion, "spdx-version", cfg.SPDXVersion, "SPDX license-list-data version/tag")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "assimilis\nUsage:\n %s [flags]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if version != nil && *version {
		displayVersion()
		return
	}

	if strings.TrimSpace(cfg.RepoName) == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\nERROR: -repo-name cannot be empty")
		return
	}

	if err := generator.Run(context.Background(), cfg); err != nil {
		var unknownErr generator.UnknownLicensesError
		if errors.As(err, &unknownErr) {
			fmt.Fprintln(os.Stderr, "ERROR: Unknown license expressions found:")
			for _, id := range unknownErr.IDs {
				fmt.Fprintln(os.Stderr, "-", id)
			}
			fmt.Fprintln(os.Stderr, "Map them to valid SPDX IDs or add custom license texts.")
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
