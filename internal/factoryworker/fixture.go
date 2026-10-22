package factoryworker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

// Fixture executes an explicitly labelled offline repository change. It does not
// call a model or pretend to be a production coding agent.
func Fixture(ctx context.Context, in io.Reader, out io.Writer, delay time.Duration, fail bool) error {
	var a factory.Assignment
	if e := json.NewDecoder(io.LimitReader(in, 16<<20)).Decode(&a); e != nil {
		return e
	}
	if !a.Fixture || a.Stage != factory.StageImplementation || a.WorkspacePath == "" {
		return errors.New("offline fixture requires a fixture implementation assignment")
	}
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	if fail {
		return errors.New("requested offline fixture failure")
	}
	content := fmt.Sprintf("Offline Waymark fixture\njob=%s\nattempt=%s\nrequirements=%s\n", a.JobID, a.AttemptID, a.RequirementFingerprint)
	if e := os.WriteFile(filepath.Join(a.WorkspacePath, "waymark-fixture.txt"), []byte(content), 0600); e != nil {
		return e
	}
	if _, e := git(a.WorkspacePath, "add", "--", "waymark-fixture.txt"); e != nil {
		return e
	}
	if _, e := git(a.WorkspacePath, "-c", "user.name=Waymark offline fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "Offline Waymark fixture candidate"); e != nil {
		return e
	}
	candidate, e := git(a.WorkspacePath, "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	_, e = fmt.Fprintf(out, "offline fixture produced candidate %s\n", candidate)
	return e
}

// FixtureCheck observes the actual candidate content and Git identity.
func FixtureCheck(in io.Reader, out io.Writer) error {
	var a factory.Assignment
	if e := json.NewDecoder(io.LimitReader(in, 16<<20)).Decode(&a); e != nil {
		return e
	}
	if !a.Fixture || a.Stage != factory.StageVerification {
		return errors.New("offline fixture check requires a fixture verification assignment")
	}
	if e := verifyCandidate(Assignment{Workspace: a.WorkspacePath, Candidate: a.CandidateRevision}); e != nil {
		return e
	}
	b, e := os.ReadFile(filepath.Join(a.WorkspacePath, "waymark-fixture.txt"))
	if e != nil {
		return e
	}
	if len(b) == 0 {
		return errors.New("fixture candidate content is empty")
	}
	_, e = fmt.Fprintf(out, "offline fixture verified candidate %s and %d content bytes\n", a.CandidateRevision, len(b))
	return e
}
