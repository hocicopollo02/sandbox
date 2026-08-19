package ui

import (
	"fmt"
	"io"
	"os"
)

type Printer struct {
	Out   io.Writer
	Err   io.Writer
	Color bool
}

func New(out, errOut io.Writer) Printer {
	return Printer{Out: out, Err: errOut, Color: os.Getenv("NO_COLOR") == ""}
}

func (p Printer) Header(text string) {
	fmt.Fprintln(p.Out, p.paint("1;36", text))
}

func (p Printer) Success(text string) {
	fmt.Fprintf(p.Out, "%s %s\n", p.paint("32", "✓"), text)
}

func (p Printer) Failure(text string) {
	fmt.Fprintf(p.Out, "%s %s\n", p.paint("31", "✗"), text)
}

func (p Printer) Warning(text string) {
	fmt.Fprintf(p.Out, "%s %s\n", p.paint("33", "!"), text)
}

func (p Printer) paint(code, text string) string {
	if !p.Color {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
