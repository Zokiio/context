package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Zokiio/context/internal/discovery"
	urfave "github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func defaultTerminal() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

func setupCommand(stdout, stderr io.Writer, environment Environment) *urfave.Command {
	if environment.IsTerminal == nil {
		environment.IsTerminal = defaultTerminal
	}
	return &urfave.Command{
		Name: "setup", Usage: "Connect an existing Project bundle to a directory",
		Writer: stderr, ErrWriter: stderr, DisableSliceFlagSeparator: true,
		ExitErrHandler: func(context.Context, *urfave.Command, error) {},
		OnUsageError:   func(_ context.Context, _ *urfave.Command, err error, _ bool) error { return err },
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "records", Usage: "Existing Project bundle directory"},
			&urfave.StringFlag{Name: "directory", Usage: "Binding directory; defaults to cwd"},
			&urfave.BoolFlag{Name: "personal", Usage: "Write a personal project registration"},
			&urfave.StringFlag{Name: "alias", Usage: "Personal project alias; requires --personal"},
			&urfave.StringSliceFlag{Name: "allow-source", Usage: "Supporting-document directory to authorize; repeat for multiple directories"},
			&urfave.BoolFlag{Name: "dry-run", Usage: "Print the exact proposed configuration without writing"},
			&urfave.BoolFlag{Name: "replace", Usage: "Update a conflicting binding in the destination file"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() != 0 {
				return errors.New("setup does not accept positional arguments")
			}
			for _, name := range []string{"records", "directory", "personal", "alias", "dry-run", "replace"} {
				if cmd.Count(name) > 1 {
					return fmt.Errorf("--%s may only be specified once", name)
				}
			}
			for _, name := range []string{"records", "directory", "alias"} {
				if cmd.IsSet(name) && strings.TrimSpace(cmd.String(name)) == "" {
					return fmt.Errorf("--%s must be nonempty", name)
				}
			}
			if cmd.IsSet("alias") && !cmd.Bool("personal") {
				return errors.New("--alias requires --personal")
			}
			cwd, err := environment.WorkingDirectory()
			if err != nil {
				return fmt.Errorf("read setup working directory: %w", err)
			}
			home, err := environment.HomeDirectory()
			if err != nil {
				return fmt.Errorf("read setup home directory: %w", err)
			}
			request := discovery.SetupRequest{Cwd: cwd, Home: home, Records: cmd.String("records"), Directory: cmd.String("directory"),
				Personal: cmd.Bool("personal"), Alias: cmd.String("alias"), AliasSet: cmd.IsSet("alias"), AllowSources: cmd.StringSlice("allow-source"), Replace: cmd.Bool("replace")}
			if !cmd.IsSet("records") {
				if !environment.IsTerminal() {
					return errors.New("setup requires --records PATH when input is not an interactive terminal")
				}
				if err := guideSetup(environment.Input, stderr, cmd, &request); err != nil {
					return err
				}
			}
			plan, err := discovery.PrepareSetup(ctx, request)
			if err != nil {
				return err
			}
			if err := printSetupSummary(stdout, plan.Summary()); err != nil {
				return err
			}
			if cmd.Bool("dry-run") {
				return writeSetupText(stdout, "\nProposed configuration:\n"+plan.Document())
			}
			return plan.Apply(ctx)
		},
	}
}

func guideSetup(input io.Reader, output io.Writer, cmd *urfave.Command, request *discovery.SetupRequest) error {
	reader := bufio.NewReader(input)
	prompt := func(label, fallback string) (string, error) {
		if fallback != "" {
			label += " [" + fallback + "]"
		}
		if err := writeSetupText(output, label+": "); err != nil {
			return "", err
		}
		answer, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read setup answer: %w; supply --records and other flags for noninteractive setup", err)
		}
		answer = strings.TrimSuffix(strings.TrimSuffix(answer, "\n"), "\r")
		if answer == "" {
			answer = fallback
		}
		return answer, nil
	}
	var err error
	if request.Records, err = prompt("Existing records directory", ""); err != nil {
		return err
	}
	if !cmd.IsSet("directory") {
		if request.Directory, err = prompt("Binding directory", request.Cwd); err != nil {
			return err
		}
	}
	if !cmd.IsSet("personal") {
		answer, err := prompt("Save in personal configuration? yes/no", "no")
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "yes", "y":
			request.Personal = true
		case "no", "n":
		default:
			return errors.New("personal configuration answer must be yes or no")
		}
	}
	if request.Personal && !cmd.IsSet("alias") {
		if request.Alias, err = prompt("Personal alias, or Enter to retain the current alias", ""); err != nil {
			return err
		}
		request.AliasSet = request.Alias != ""
	}
	if !cmd.IsSet("allow-source") {
		for {
			root, err := prompt("Allowed source directory, or Enter to finish", "")
			if err != nil {
				return err
			}
			if root == "" {
				break
			}
			request.AllowSources = append(request.AllowSources, root)
		}
	}
	return nil
}

func printSetupSummary(output io.Writer, summary discovery.SetupSummary) error {
	var text bytes.Buffer
	fmt.Fprintf(&text, "Project binding: %s\nBinding directory: %s\nRecords: %s\nDestination: %s\n", summary.Change, summary.Directory, summary.Records, summary.Destination)
	if summary.Alias != "" {
		fmt.Fprintf(&text, "Personal alias: %s\n", summary.Alias)
	}
	if len(summary.AllowSources) == 0 {
		fmt.Fprintln(&text, "Allowed sources: (none)")
	} else {
		fmt.Fprintln(&text, "Allowed sources:")
		for _, path := range summary.AllowSources {
			fmt.Fprintf(&text, "  %s\n", path)
		}
	}
	if summary.PreviousRecords != "" {
		fmt.Fprintf(&text, "Replaces records: %s\n", summary.PreviousRecords)
	}
	if summary.PreviousAlias != "" {
		fmt.Fprintf(&text, "Removes alias: %s\n", summary.PreviousAlias)
	}
	for _, path := range summary.RemovedSources {
		fmt.Fprintf(&text, "Removes allowed source: %s\n", path)
	}
	return writeSetupText(output, text.String())
}

func writeSetupText(output io.Writer, text string) error {
	n, err := io.WriteString(output, text)
	if err == nil && n != len(text) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("write setup output: %w", err)
	}
	return nil
}
