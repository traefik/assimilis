package main

import (
	"context"
	"fmt"
	"runtime"

	"github.com/urfave/cli/v3"
)

var (
	version = "dev"
	commit  = "I don't remember exactly"
	date    = "I don't remember exactly"
)

// displayVersion DisplayVersion version.
func displayVersion(_ context.Context, _ *cli.Command) error {
	fmt.Printf(`Myrmica Assimilis:
 version     : %s
 commit      : %s
 build date  : %s
 go version  : %s
 go compiler : %s
 platform    : %s/%s
`, version, commit, date, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)

	return nil
}
