package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReportAuthoredObservation(t *testing.T) {
	input, err := os.ReadFile("testdata/observed.json")
	if err != nil {
		t.Fatal(err)
	}

	command := exec.Command(buildFixture(t))
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("fixture did not exit zero: %v\nstdout: %q\nstderr: %q", err, stdout.String(), stderr.String())
	}

	// CP1-OBS-001 uses the literal row from the specification's observed example.
	const expected = "C1\tPreserve authored mapping\tpass\tchecks/observed.txt\tfixture-rev-1\n"
	if stdout.String() != expected {
		t.Fatalf("stdout = %q, want %q", stdout.String(), expected)
	}
}

func buildFixture(t *testing.T) string {
	t.Helper()
	executable := filepath.Join(t.TempDir(), "fixture")
	command := exec.Command("go", "build", "-o", executable, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, output)
	}
	return executable
}
