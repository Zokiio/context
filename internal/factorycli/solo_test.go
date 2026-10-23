package factorycli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

func TestSoloBuilderRunsGoChecksWithAPinnedModel(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx := context.Background()
	service, err := factory.Open(t.TempDir(), factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	operator, _, err := soloOperator(ctx, service, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	builder := func() []string {
		t.Helper()
		profiles, err := service.ListExecutors(ctx, operator)
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range profiles {
			if profile.Key == "claude-agent" {
				return profile.Command[1:]
			}
		}
		t.Fatalf("no claude-agent in %+v", profiles)
		return nil
	}

	var stderr bytes.Buffer
	registerAgentExecutors(ctx, service, operator, &stderr, "sonnet")
	want := []string{"factory", "agent", "--harness", "claude", "--model", "sonnet", "--max-budget-usd", "2", "--allow-tool", "Bash(go test:*)", "--allow-tool", "Bash(go vet:*)"}
	if got := builder(); !slices.Equal(got, want) || stderr.Len() != 0 {
		t.Fatalf("builder = %q, stderr %q", got, stderr.String())
	}

	// Profiles never change, so an installation with another builder command is told how to switch.
	registerAgentExecutors(ctx, service, operator, &stderr, "opus")
	if got := builder(); !slices.Equal(got, want) {
		t.Fatalf("registered profile changed to %q", got)
	}
	if warning := stderr.String(); !strings.Contains(warning, "claude-agent") || !strings.Contains(warning, `"opus"`) || !strings.Contains(warning, "new key") {
		t.Fatalf("warning = %q", warning)
	}
}
