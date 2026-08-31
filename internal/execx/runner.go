package execx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	RunStream(ctx context.Context, name string, args ...string) error
	Attach(ctx context.Context, name string, args ...string) error
	LookPath(file string) (string, error)
}

type CommandRunner struct {
	Verbose bool
	Out     io.Writer
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

// RunStream runs an external command with stdout and stderr passed through
// unchanged and no attached stdin or TTY. A non-zero process exit is returned
// as *ExitError so callers can propagate the exact exit code.
func (r *CommandRunner) RunStream(ctx context.Context, name string, args ...string) error {
	if err := r.log(name, args...); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = r.Out
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = r.Err
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &ExitError{Code: exitErr.ExitCode()}
	}
	return err
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

// ExitError carries the exit code of an external command that ran without a
// TTY. The top-level CLI maps it to the process exit code instead of printing
// it as an internal failure.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

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
