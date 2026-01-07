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
			err := validate(cfg)
			if err != nil {
				return err
			}
			return run(cfg, ctx)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application error")
	}
}

func validate(cfg generator.Config) error {
	if strings.TrimSpace(cfg.RepoName) == "" {
		return errors.New("-repo-name cannot be empty")
	}
	return nil
}

func run(cfg generator.Config, ctx context.Context) error {
	err := validate(cfg)
	if err != nil {
		return err
	}

	if err := generator.Run(ctx, cfg); err != nil {
		var unknownErr generator.UnknownLicensesError
		if errors.As(err, &unknownErr) {
			fmt.Fprintln(os.Stderr, "ERROR: Unknown license expressions found:")
			for _, id := range unknownErr.IDs {
				fmt.Fprintln(os.Stderr, "-", id)
			}
			return fmt.Errorf("Map them to valid SPDX IDs or add custom license texts.")
		}
		return fmt.Errorf("ERROR: %v\n", err)
	}
	return nil
}
