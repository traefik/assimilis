// Package main provides the Assimilis license report generator CLI entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "repo-name",
				Usage:       "Name of the repository",
				Destination: &cfg.RepoName,
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
		},
		Action: func(ctx context.Context, _ *cli.Command) error {
			return run(ctx, cfg)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		var unknownErr generator.UnknownLicensesError
		if errors.As(err, &unknownErr) {
			log.Fatal().
				Err(unknownErr).
				Strs("unknown_licenses", unknownErr.IDs).
				Msg("Unknown license expressions found. Map them to valid SPDX IDs or add custom license texts.")
		}

		log.Fatal().Err(err).Msg("Application error")
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
