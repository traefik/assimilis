// Package version exposes build-time metadata injected via ldflags.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the release version.
	Version = "dev"
	// Commit is the git commit SHA.
	Commit = "I don't remember exactly"
	// Date is the build date.
	Date = "I don't remember exactly"
)

// Print writes version information to stdout.
func Print() {
	fmt.Printf(`Myrmica Assimilis:
 version     : %s
 commit      : %s
 build date  : %s
 go version  : %s
 go compiler : %s
 platform    : %s/%s
`, Version, Commit, Date, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
}
