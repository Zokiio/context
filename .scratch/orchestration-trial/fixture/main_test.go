package main

import (
	"bytes"
	"errors"
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

func TestPreserveAuthoredFailure(t *testing.T) {
	// CP1-OBS-002 preserves the authored result, nonexistent source, and revision.
	const expected = "F7\tKeep authored failure visible\tfail\tsynthetic/does-not-exist.txt\tauthored-revision-not-HEAD\n"
	assertFixtureOutput(t, "testdata/authored-failure.json", expected)
}

func TestPreserveCriterionOrder(t *testing.T) {
	// CP1-OBS-003 uses literal rows in the input's authored order.
	const expected = "Z9\tFirst authored row\tfail\tchecks/z.txt\trevision-z\nA1\tSecond authored row\tpass\tchecks/a.txt\trevision-a\n"
	assertFixtureOutput(t, "testdata/observed-order.json", expected)
}

func TestRejectMalformedJSON(t *testing.T) {
	// CP1-DECODE-004 requires rejection and a diagnostic, without fixed wording.
	input, err := os.ReadFile("testdata/malformed.json")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(buildFixture(t))
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	exitStatus := -1
	if command.ProcessState != nil {
		exitStatus = command.ProcessState.ExitCode()
	}
	t.Logf("fixture exit status: %d\nstdout: %q\nstderr: %q", exitStatus, stdout.String(), stderr.String())
	if err == nil {
		t.Fatal("fixture accepted malformed JSON with exit status zero")
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("fixture failed to launch, not a JSON rejection: %v", err)
	}
	if exitError.ExitCode() <= 0 {
		t.Fatalf("fixture did not reject JSON with a nonzero exit status: %v", err)
	}
	if stderr.Len() == 0 {
		t.Fatal("fixture rejected malformed JSON without a stderr diagnostic")
	}
}

func assertFixtureOutput(t *testing.T, inputFile, expected string) {
	t.Helper()
	input, err := os.ReadFile(inputFile)
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
	if stdout.String() != expected {
		t.Fatalf("stdout = %q, want %q", stdout.String(), expected)
	}
}

func TestReportMissingObservation(t *testing.T) {
	// CP2-MISSING-001 uses the literal row from the specification's missing example.
	const expected = "C2\tVerification remains missing\tmissing\t-\t-\n"
	assertFixtureOutput(t, "testdata/missing.json", expected)
}

func TestReportNullObservation(t *testing.T) {
	// CP2-MISSING-002 gives JSON-null the independently authored missing markers.
	const expected = "N4\tNull verification remains missing\tmissing\t-\t-\n"
	assertFixtureOutput(t, "testdata/null-observation.json", expected)
}

func TestPreserveMixedObservationRows(t *testing.T) {
	// CP2-MIXED-003 preserves the independent literals for all four row kinds.
	const expected = "C1\tPreserve authored mapping\tpass\tchecks/observed.txt\tfixture-rev-1\n" +
		"C2\tVerification remains missing\tmissing\t-\t-\n" +
		"F7\tKeep authored failure visible\tfail\tsynthetic/does-not-exist.txt\tauthored-revision-not-HEAD\n" +
		"N4\tNull verification remains missing\tmissing\t-\t-\n"
	assertFixtureOutput(t, "testdata/mixed-observations.json", expected)
}
