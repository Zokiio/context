package factorycli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	cli "github.com/urfave/cli/v3"
)

func TestSoloJudgeConfiguration(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	config := filepath.Join(t.TempDir(), "judge.json")
	if err := os.WriteFile(config, []byte(`["custom-judge","--read-only"]`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		args, want []string
	}{
		{name: "default", want: []string{"factory", "judge", "--model", "sonnet", "--max-budget-usd", "0.5"}},
		{name: "disabled", args: []string{"--no-judge"}},
		{name: "explicit command", args: []string{"--judge-command-file", config}, want: []string{"custom-judge", "--read-only"}},
		{name: "disabled overrides command", args: []string{"--no-judge", "--judge-command-file", config}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := soloCommand(io.Discard, io.Discard)
			command.Action = func(_ context.Context, c *cli.Command) error {
				argv, err := soloJudge(c)
				if tc.name == "default" && len(argv) > 0 {
					argv = argv[1:]
				}
				if !slices.Equal(argv, tc.want) {
					t.Errorf("judge command = %q, want %q", argv, tc.want)
				}
				return err
			}
			if err := command.Run(context.Background(), append([]string{"solo", "--data-dir", "unused"}, tc.args...)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
