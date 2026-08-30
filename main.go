package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/hocicopollo02/sandbox/cmd"
	"github.com/hocicopollo02/sandbox/internal/execx"
)

func main() {
	if err := cmd.Execute(); err != nil {
		var exitErr *execx.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
