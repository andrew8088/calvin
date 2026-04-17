package main

import (
	"errors"
	"os"

	"github.com/andrew8088/calvin/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cli.SetVersion(version, commit)
	if err := cli.Execute(); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode)
		}
		os.Exit(1)
	}
}
