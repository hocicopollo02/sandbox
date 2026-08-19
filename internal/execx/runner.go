package execx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	Attach(ctx context.Context, name string, args ...string) error
	LookPath(file string) (string, error)
}

type CommandRunner struct {
	Verbose bool
	Err     io.Writer
}

func (r *CommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if err := r.log(name, args...); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.Bytes(), err
}

func (r *CommandRunner) Attach(ctx context.Context, name string, args ...string) error {
	if err := r.log(name, args...); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *CommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (r *CommandRunner) log(name string, args ...string) error {
	if !r.Verbose {
		return nil
	}
	if r.Err == nil {
		r.Err = os.Stderr
	}
	_, err := fmt.Fprintf(r.Err, "[%s] %s %s\n", name, name, strings.Join(args, " "))
	return err
}
