package main

import (
	"errors"
	"os"

	"github.com/ppiankov/ancc/internal/cli"
)

var version = "dev"

func main() {
	err := cli.Execute(version)
	if err == nil {
		return
	}

	var exitErr *cli.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.Code)
	}

	os.Exit(1)
}
