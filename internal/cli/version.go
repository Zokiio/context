package cli

import (
	"context"
	"fmt"
	"io"
	"runtime/debug"

	urfave "github.com/urfave/cli/v3"
)

func versionCommand(stdout, stderr io.Writer) *urfave.Command {
	return &urfave.Command{
		Name: "version", Usage: "Print this binary's embedded version and VCS state",
		Writer: stderr, ErrWriter: stderr,
		Action: func(_ context.Context, cmd *urfave.Command) error {
			if cmd.NArg() != 0 {
				return fmt.Errorf("version does not accept positional arguments")
			}
			info, _ := debug.ReadBuildInfo()
			return renderVersion(stdout, info)
		},
	}
}

func renderVersion(w io.Writer, info *debug.BuildInfo) error {
	version, revision, modified := "unknown", "unknown", "unknown"
	if info != nil {
		if info.Main.Version != "" {
			version = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if setting.Value != "" {
					revision = setting.Value
				}
			case "vcs.modified":
				if setting.Value == "true" || setting.Value == "false" {
					modified = setting.Value
				}
			}
		}
	}
	_, err := fmt.Fprintf(w, "ctx %s\nrevision: %s\nmodified: %s\n", version, revision, modified)
	return err
}
