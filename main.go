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
		if line, ok := cmd.RenderError(err, cmd.ErrorFormat); ok {
			fmt.Fprintln(os.Stderr, line)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
