package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/hocicopollo02/sandbox/internal/config"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
)

type CreateAnswers struct {
	Name         string
	Distribution string
	Persistence  string
	HomeMode     string
	AutoEnter    bool
}

func PromptCreate(defaults config.Config, in io.Reader, out io.Writer) (CreateAnswers, error) {
	answers := CreateAnswers{
		Distribution: defaults.DefaultDistro,
		Persistence:  string(defaults.DefaultPersistence),
		HomeMode:     string(defaults.DefaultHome),
		AutoEnter:    defaults.AutoEnter,
	}

	distroOptions := make([]huh.Option[string], 0)
	for _, distro := range sandbox.Distributions() {
		distroOptions = append(distroOptions, huh.NewOption(distro.Name, distro.ID))
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Name").Value(&answers.Name).Validate(func(value string) error {
				_, err := sandbox.ValidateName(value)
				return err
			}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Distribution").Options(distroOptions...).Value(&answers.Distribution),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Persistence").Options(
				huh.NewOption("Disposable", string(sandbox.Disposable)),
				huh.NewOption("Persistent", string(sandbox.Persistent)),
			).Value(&answers.Persistence),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Home").Options(
				huh.NewOption("Isolated", string(sandbox.IsolatedHome)),
			).Value(&answers.HomeMode),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Enter sandbox now?").Affirmative("Yes").Negative("No").Value(&answers.AutoEnter),
		),
	).WithInput(in).WithOutput(out)
	if err := form.Run(); err != nil {
		return CreateAnswers{}, fmt.Errorf("interactive prompt: %w", err)
	}
	return answers, nil
}

func ConfirmCreate(name string, in io.Reader, out io.Writer) (bool, error) {
	confirmed := true
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(fmt.Sprintf("Create sandbox %q?", name)).Affirmative("Yes").Negative("No").Value(&confirmed),
	)).WithInput(in).WithOutput(out)
	if err := form.Run(); err != nil {
		return false, fmt.Errorf("creation confirmation: %w", err)
	}
	return confirmed, nil
}

func ConfirmDelete(name string, in io.Reader, out io.Writer) (bool, error) {
	confirmed := false
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(fmt.Sprintf("Delete sandbox %q?", name)).Affirmative("Yes").Negative("No").Value(&confirmed),
	)).WithInput(in).WithOutput(out)
	if err := form.Run(); err != nil {
		return false, fmt.Errorf("delete confirmation: %w", err)
	}
	return confirmed, nil
}

func ConfirmDeleteHome(in io.Reader, out io.Writer) (bool, error) {
	confirmed := true
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Also delete its isolated home?").Affirmative("Yes").Negative("No").Value(&confirmed),
	)).WithInput(in).WithOutput(out)
	if err := form.Run(); err != nil {
		return false, fmt.Errorf("home deletion confirmation: %w", err)
	}
	return confirmed, nil
}

func SelectSandbox(names []string, in io.Reader, out io.Writer) (string, error) {
	if len(names) == 0 {
		return "", fmt.Errorf("no persistent sandboxes found")
	}
	if len(names) == 1 {
		return names[0], nil
	}
	options := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		options = append(options, huh.NewOption(name, name))
	}
	selected := names[0]
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Select sandbox").Options(options...).Value(&selected),
	)).WithInput(in).WithOutput(out)
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("sandbox selection: %w", err)
	}
	return selected, nil
}
