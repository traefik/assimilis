// Package main provides the Assimilis license report generator CLI entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/traefik/assimilis/pkg/generator"
	"github.com/traefik/assimilis/pkg/logger"
	"github.com/urfave/cli/v3"
)

func main() {
	cfg := generator.DefaultConfig()
	app := &cli.Command{
		Name:  "assimilis",
		Usage: "Generate OSS attribution files",
		Commands: []*cli.Command{
			{
				Name:   "version",
				Usage:  "Display version information",
				Action: displayVersion,
			},
		},
		Flags:  buildFlags(&cfg),
		Action: func(ctx context.Context, _ *cli.Command) error { return run(ctx, cfg) },
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		var unknownErr generator.UnknownLicensesError
		if errors.As(err, &unknownErr) {
			log.Fatal().
				Err(unknownErr).
				Strs("unknown_licenses", unknownErr.IDs).
				Msg("Unknown license expressions found.")
		}

		log.Fatal().Err(err).Msg("Application error")
	}
}

func buildFlags(cfg *generator.Config) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "repo-name",
			Usage:       "Name of the repository",
			Destination: &cfg.RepoName,
		},
		&cli.StringFlag{
			Name:  "output-dir",
			Usage: "Base output directory",
			Value: cfg.OutDir,
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				cfg.OutDir = v
				cfg.OutLicensesDir = filepath.Join(v, "licenses")
				cfg.SBOMPath = filepath.Join(v, "sbom")

				return nil
			},
		},
		&cli.StringFlag{
			Name:        "html-template",
			Usage:       "Override HTML template path (default: embedded)",
			Destination: &cfg.HTMLTemplatePath,
		},
		&cli.StringFlag{
			Name:        "notice-template",
			Usage:       "Override NOTICE template path (default: embedded)",
			Destination: &cfg.NoticeTplPath,
		},
		&cli.StringFlag{
			Name:        "spdx-version",
			Usage:       "SPDX license-list-data version/tag",
			Value:       cfg.SPDXVersion,
			Destination: &cfg.SPDXVersion,
		},
		&cli.StringFlag{
			Name:        "html-filename",
			Usage:       "Output HTML filename",
			Value:       cfg.HTMLFileName,
			Destination: &cfg.HTMLFileName,
		},
		&cli.StringFlag{
			Name:        "notice-filename",
			Usage:       "Output NOTICE filename",
			Value:       cfg.NoticeFileName,
			Destination: &cfg.NoticeFileName,
		},
		&cli.StringFlag{
			Name:        "license-map",
			Usage:       "Path to external license-map JSON (default: embedded)",
			Destination: &cfg.LicenseMapPath,
		},
		&cli.StringFlag{
			Name:        "license-corrections",
			Usage:       "Path to external license-corrections JSON (default: embedded)",
			Destination: &cfg.LicenseCorrectionsPath,
		},
		&cli.StringFlag{
			Name:        "filters",
			Usage:       "Path to external filters JSON (default: embedded)",
			Destination: &cfg.FiltersPath,
		},
		&cli.StringFlag{
			Name:        "node-modules-dir",
			Usage:       "Path to node_modules directory for npm copyright extraction (default: auto-detect)",
			Destination: &cfg.NodeModulesDir,
		},
		&cli.StringFlag{
			Name:        "python-site-packages-dir",
			Usage:       "Path to Python site-packages directory for PyPI copyright extraction",
			Destination: &cfg.PythonSitePackagesDir,
		},
	}
}

func validate(cfg generator.Config) error {
	if strings.TrimSpace(cfg.RepoName) == "" {
		return fmt.Errorf("--repo-name cannot be empty")
	}

	return nil
}

func run(ctx context.Context, cfg generator.Config) error {
	logger.Setup("info")

	err := validate(cfg)
	if err != nil {
		return err
	}

	if err := generator.Run(ctx, cfg); err != nil {
		return fmt.Errorf("failed to run generator: %w", err)
	}

	return nil
}
