package main

import (
	"context"

	"github.com/traefik/assimilis/pkg/version"
	"github.com/urfave/cli/v3"
)

func displayVersion(_ context.Context, _ *cli.Command) error {
	version.Print()

	return nil
}
