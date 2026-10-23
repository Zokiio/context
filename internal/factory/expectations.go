package factory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/taskcontext"
)

// Holdout expectations are tests written from a work item's criteria by an author that
// never implements the work, kept where the implementer cannot change them. An author
// command writes them only under the project's protected paths, a person accepts the
// commit, and a job then starts from it, so verification runs tests the implementer
// could not weaken. The commit reaches a branch only together with the implementation,
// through the verified merge, so no branch carries tests for work that does not exist.

type ExpectationStatus string

const (
	ExpectationRunning  ExpectationStatus = "running"
	ExpectationProposed ExpectationStatus = "proposed"
	ExpectationFailed   ExpectationStatus = "failed"
	ExpectationAccepted ExpectationStatus = "accepted"
	ExpectationRejected ExpectationStatus = "rejected"
)

const (
	expectationTimeout  = 15 * time.Minute
	maxExpectationPatch = 200 << 10
)

type Expectation struct {
	ID             ID                `json:"id"`
	ProjectID      ProjectID         `json:"projectId"`
	WorkItemID     string            `json:"workItemId"`
	Title          string            `json:"title"`
	TicketPath     string            `json:"ticketPath"`
	CriteriaSHA256 string            `json:"criteriaSHA256"`
	Status         ExpectationStatus `json:"status"`
	RequestedBy    UserID            `json:"requestedBy"`
	BaseRevision   string            `json:"baseRevision"`
	Commit         string            `json:"commit,omitempty"`
	Files          []string          `json:"files,omitempty"`
	Patch          string            `json:"patch,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	// Criteria are the acceptance criteria the author wrote tests for, in order.
	Criteria []string `json:"criteria,omitempty"`
	// Coverage is the author's own account of which test checks which criterion. Like
	// the notes, it is a claim a person reads, not evidence.
	Coverage []CriterionCoverage `json:"coverage,omitempty"`
	// Concerns are contradictions or problems the author found in the criteria.
	Concerns   []string    `json:"concerns,omitempty"`
	Run        *RefinerRun `json:"run,omitempty"`
	Log        string      `json:"log,omitempty"`
	Error      string      `json:"error,omitempty"`
	DecidedBy  *UserID     `json:"decidedBy,omitempty"`
	DecisionID string      `json:"decisionId,omitempty"`
	Delegated  string      `json:"delegated,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type StartExpectationsRequest struct {
	RequestID  string    `json:"requestId"`
	ProjectID  ProjectID `json:"projectId"`
	WorkItemID string    `json:"workItemId"`
	// BaseRevision defaults to the project repository's HEAD.
	BaseRevision string `json:"baseRevision,omitempty"`
}

type DecideExpectationsRequest struct {
	RequestID     string `json:"requestId"`
	ExpectationID ID     `json:"expectationId"`
}

// ExpectationInput is written to the expectation author's standard input. The author
// runs in a temporary worktree at the base revision.
type ExpectationInput struct {
	ExpectationID  ID              `json:"expectationId"`
	WorkItemID     string          `json:"workItemId"`
	Title          string          `json:"title"`
	TicketPath     string          `json:"ticketPath"`
	Ticket         string          `json:"ticket"`
	Criteria       []string        `json:"criteria"`
	Context        []ContextSource `json:"context"`
	ProtectedPaths []string        `json:"protectedPaths"`
	BaseRevision   string          `json:"baseRevision"`
}

// ExpectationOutput is the JSON the author writes to standard output after editing the
// worktree.
type ExpectationOutput struct {
	Notes    string              `json:"notes"`
	Coverage []CriterionCoverage `json:"coverage,omitempty"`
	Concerns []string            `json:"concerns,omitempty"`
	Run      RefinerRun          `json:"run"`
}

// CriterionCoverage names the tests that check one acceptance criterion, counted from 1
// in the input's criteria.
type CriterionCoverage struct {
	Criterion int      `json:"criterion"`
	Tests     []string `json:"tests"`
}

// Uncovered lists the criteria, counted from 1, that no test checks by the author's
// account. Without coverage every criterion counts as uncovered.
func (e Expectation) Uncovered() []int {
	covered := map[int]bool{}
	for _, entry := range e.Coverage {
		if len(entry.Tests) > 0 {
			covered[entry.Criterion] = true
		}
	}
	result := []int{}
	for index := range e.Criteria {
		if !covered[index+1] {
			result = append(result, index+1)
		}
	}
	return result
}

func cleanCoverage(coverage []CriterionCoverage, criteria int) []CriterionCoverage {
	result := []CriterionCoverage{}
	for _, entry := range coverage {
		if entry.Criterion < 1 || entry.Criterion > criteria {
			continue
		}
		tests := []string{}
		for _, test := range entry.Tests {
			if test = strings.TrimSpace(test); test != "" && len(tests) < 20 {
				tests = append(tests, truncateRunes(test, 200))
			}
		}
		result = append(result, CriterionCoverage{Criterion: entry.Criterion, Tests: tests})
	}
	return result
}

func cleanConcerns(concerns []string) []string {
	result := []string{}
	for _, concern := range concerns {
		if concern = strings.TrimSpace(concern); concern != "" && len(result) < 10 {
			result = append(result, truncateRunes(concern, maxRefinementRunes))
		}
	}
	return result
}

func (s *Service) StartExpectations(ctx context.Context, actor Principal, request StartExpectationsRequest) (Expectation, error) {
	if request.ProjectID == "" || strings.TrimSpace(request.WorkItemID) == "" {
		return Expectation{}, fault(CodeInvalid, "projectId and workItemId are required", nil)
	}
	if err := requireRequestID(request.RequestID); err != nil {
		return Expectation{}, err
	}
	if len(s.state.expectationAuthor) == 0 {
		return Expectation{}, fault(CodeNotReady, "no expectation author is configured; start the service with an expectation command", nil)
	}
	current, err := authorizeProject(ctx, s.state.db, actor, request.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Expectation{}, err
	}
	if replayed, ok, err := s.readExpectationReceipt(ctx, current.UserID, "start_expectations", request.RequestID, request); err != nil || ok {
		return replayed, err
	}
	project, err := readProject(ctx, s.state.db, request.ProjectID)
	if err != nil {
		return Expectation{}, err
	}
	if len(project.Policy.ProtectedPaths) == 0 {
		return Expectation{}, fault(CodeNotReady, "the project has no protected paths; holdout expectations must live where the implementer cannot change them", nil)
	}
	work, criteriaDigest, err := s.currentWorkItem(ctx, project, request.WorkItemID)
	if err != nil {
		return Expectation{}, err
	}
	if err := requireNoActiveJob(ctx, s.state.db, project.ID, request.WorkItemID); err != nil {
		return Expectation{}, err
	}
	revision := strings.TrimSpace(request.BaseRevision)
	if revision == "" {
		revision = "HEAD"
	}
	base, err := s.resolveCommit(ctx, project.RepositoryPath, revision)
	if err != nil {
		return Expectation{}, err
	}
	raw, err := os.ReadFile(work.Source)
	if err != nil {
		return Expectation{}, fault(CodeInternal, "read work item", err)
	}
	criteria, err := acceptanceCriteria(work.Source)
	if err != nil {
		return Expectation{}, fault(CodeInvalid, "read acceptance criteria", err)
	}
	title := ""
	if work.Title != nil {
		title = *work.Title
	}
	sources := []ContextSource{}
	if capture, err := recordread.NewCapture(project.BundlePath, project.AllowedSourceRoots, recordread.Limits{MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes}); err == nil {
		if assembled, err := taskcontext.AssembleWithCapture(ctx, work.Source, capture); err == nil {
			if converted, _, _, err := admissionContext(assembled); err == nil {
				sources = converted
			}
		}
		capture.Close()
	}
	id, err := newID("exp_")
	if err != nil {
		return Expectation{}, fault(CodeInternal, "create expectation identity", err)
	}
	now := s.state.now().UTC()
	expectation := Expectation{ID: ID(id), ProjectID: project.ID, WorkItemID: request.WorkItemID, Title: title, TicketPath: work.Source, CriteriaSHA256: criteriaDigest, Status: ExpectationRunning, RequestedBy: current.UserID, BaseRevision: base, Criteria: criteria, CreatedAt: now, UpdatedAt: now}
	input := ExpectationInput{ExpectationID: expectation.ID, WorkItemID: request.WorkItemID, Title: title, TicketPath: work.Source, Ticket: string(raw), Criteria: criteria, Context: sources, ProtectedPaths: project.Policy.ProtectedPaths, BaseRevision: base}

	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Expectation{}, fault(CodeInternal, "start expectation transaction", err)
	}
	defer tx.Rollback()
	if saved, ok, hash, err := receipt[Expectation](ctx, tx, current.UserID, "start_expectations", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		if err := saveExpectation(ctx, tx, expectation, true); err != nil {
			return Expectation{}, err
		}
		if err := saveReceipt(ctx, tx, current.UserID, "start_expectations", request.RequestID, hash, expectation, false, now); err != nil {
			return Expectation{}, err
		}
		if err := tx.Commit(); err != nil {
			return Expectation{}, fault(CodeInternal, "commit expectation start", err)
		}
	}
	s.state.refiners.Add(1)
	go s.runExpectationAuthor(expectation.ID, input, project)
	return expectation, nil
}

// runExpectationAuthor lets the author edit a temporary worktree, then keeps the result
// only if every changed file is protected. The commit is pinned until it is decided.
func (s *Service) runExpectationAuthor(id ID, input ExpectationInput, project Project) {
	defer s.state.refiners.Done()
	ctx, cancel := context.WithTimeout(s.state.background, expectationTimeout)
	defer cancel()
	result := struct {
		commit, patch, log string
		files              []string
		output             ExpectationOutput
		ran                bool
		err                error
	}{}
	directory := filepath.Join(s.state.workspaceRoot, "expectations", string(id))
	if err := os.MkdirAll(filepath.Dir(directory), 0o700); err != nil {
		result.err = err
	} else if _, err := s.git(ctx, project.RepositoryPath, "worktree", "add", "--detach", directory, input.BaseRevision); err != nil {
		result.err = fmt.Errorf("create expectation worktree: %w", err)
	} else {
		defer func() {
			_, _ = s.git(context.Background(), project.RepositoryPath, "worktree", "remove", "--force", directory)
		}()
		result.output, result.log, result.err = s.execExpectationAuthor(ctx, directory, input)
		result.ran = result.err == nil
		if result.err == nil {
			result.commit, result.files, result.err = s.commitExpectations(ctx, directory, input, result.output.Run)
		}
		if result.err == nil {
			if _, err := s.git(ctx, project.RepositoryPath, "update-ref", expectationPin(id), result.commit); err != nil {
				result.err = fmt.Errorf("pin expectations: %w", err)
			}
		}
		if result.err == nil {
			patch, _ := s.git(ctx, directory, "show", "--format=", "--patch", result.commit)
			result.patch = truncateRunes(patch, maxExpectationPatch)
		}
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	expectation, err := loadExpectation(context.Background(), s.state.db, id)
	if err != nil || expectation.Status != ExpectationRunning {
		return
	}
	expectation.Log, expectation.UpdatedAt = result.log, s.state.now().UTC()
	if result.ran {
		// Keep what the author said and spent, even when its work is refused.
		run := result.output.Run
		expectation.Notes, expectation.Run = truncateRunes(result.output.Notes, 4*maxRefinementRunes), &run
		expectation.Concerns = cleanConcerns(result.output.Concerns)
	}
	if result.err != nil {
		expectation.Status, expectation.Error = ExpectationFailed, truncateRunes(result.err.Error(), maxRefinementRunes)
	} else {
		expectation.Status, expectation.Commit, expectation.Files, expectation.Patch = ExpectationProposed, result.commit, result.files, result.patch
		expectation.Coverage = cleanCoverage(result.output.Coverage, len(expectation.Criteria))
	}
	tx, err := s.state.db.BeginTx(context.Background(), nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if saveExpectation(context.Background(), tx, expectation, false) == nil {
		_ = tx.Commit()
	}
}

func (s *Service) execExpectationAuthor(ctx context.Context, directory string, input ExpectationInput) (ExpectationOutput, string, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return ExpectationOutput{}, "", err
	}
	command := exec.CommandContext(ctx, s.state.expectationAuthor[0], s.state.expectationAuthor[1:]...)
	command.Dir = directory
	command.Stdin = bytes.NewReader(encoded)
	command.WaitDelay = 5 * time.Second
	stdout := &limitedBuffer{limit: maxRefinerOutput}
	stderr := &limitedBuffer{limit: maxRefinerLog}
	command.Stdout, command.Stderr = stdout, stderr
	runErr := command.Run()
	log := stderr.String()
	if ctx.Err() != nil {
		return ExpectationOutput{}, log, fmt.Errorf("expectation author stopped: %w", ctx.Err())
	}
	if runErr != nil {
		return ExpectationOutput{}, log, fmt.Errorf("expectation author failed: %w", runErr)
	}
	var output ExpectationOutput
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &output); err != nil {
		return ExpectationOutput{}, log, fmt.Errorf("expectation author output is not valid: %w", err)
	}
	return output, log, nil
}

// commitExpectations accepts the author's work only when it changed at least one file
// and every changed file matches a protected path.
func (s *Service) commitExpectations(ctx context.Context, directory string, input ExpectationInput, run RefinerRun) (string, []string, error) {
	if _, err := s.git(ctx, directory, "add", "-A"); err != nil {
		return "", nil, err
	}
	changed, err := s.git(ctx, directory, "diff", "--cached", "--name-only", "--no-renames", "-z", input.BaseRevision)
	if err != nil {
		return "", nil, err
	}
	var files, outside []string
	for _, file := range strings.Split(changed, "\x00") {
		if file == "" {
			continue
		}
		files = append(files, file)
		protected := false
		for _, pattern := range input.ProtectedPaths {
			if protectedMatch(pattern, file) {
				protected = true
				break
			}
		}
		if !protected {
			outside = append(outside, file)
		}
	}
	if len(files) == 0 {
		return "", nil, errors.New("the expectation author wrote no tests")
	}
	if len(outside) > 0 {
		return "", nil, fmt.Errorf("the expectation author changed files outside the protected paths: %s", strings.Join(outside, ", "))
	}
	message := fmt.Sprintf("Holdout expectations for %s\n\nExpectation: %s\nModel: %s\nSession: %s\n", input.WorkItemID, input.ExpectationID, run.Model, run.SessionID)
	if _, err := s.git(ctx, directory, "-c", "user.name=Waymark expectation author", "-c", "user.email=expectations@waymark.invalid", "commit", "-q", "-m", message); err != nil {
		return "", nil, err
	}
	commit, err := s.git(ctx, directory, "rev-parse", "HEAD")
	return commit, files, err
}

func (s *Service) AcceptExpectations(ctx context.Context, actor Principal, request DecideExpectationsRequest) (Expectation, error) {
	return s.decideExpectations(ctx, actor, request, ExpectationAccepted)
}

func (s *Service) RejectExpectations(ctx context.Context, actor Principal, request DecideExpectationsRequest) (Expectation, error) {
	return s.decideExpectations(ctx, actor, request, ExpectationRejected)
}

// decideExpectations records a person's decision on a proposed suite. Accepting needs
// a human, because it decides what the implementation will be held to.
func (s *Service) decideExpectations(ctx context.Context, actor Principal, request DecideExpectationsRequest, decision ExpectationStatus) (Expectation, error) {
	if err := requireRequestID(request.RequestID); err != nil {
		return Expectation{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	expectation, err := loadExpectation(ctx, s.state.db, request.ExpectationID)
	if err != nil {
		return Expectation{}, err
	}
	current, err := authorizeProject(ctx, s.state.db, actor, expectation.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Expectation{}, err
	}
	operation := "accept_expectations"
	if decision == ExpectationRejected {
		operation = "reject_expectations"
	}
	if replayed, ok, err := s.readExpectationReceipt(ctx, current.UserID, operation, request.RequestID, request); err != nil || ok {
		return replayed, err
	}
	if decision == ExpectationAccepted && current.Kind != UserKindHuman {
		return Expectation{}, fault(CodeForbidden, "accepting expectations requires a human principal", nil)
	}
	if expectation.Status != ExpectationProposed {
		return Expectation{}, fault(CodeConflict, "only proposed expectations can be decided", nil)
	}
	if decision == ExpectationAccepted {
		project, err := readProject(ctx, s.state.db, expectation.ProjectID)
		if err != nil {
			return Expectation{}, err
		}
		if _, digest, err := s.currentWorkItem(ctx, project, expectation.WorkItemID); err != nil {
			return Expectation{}, err
		} else if digest != expectation.CriteriaSHA256 {
			return Expectation{}, fault(CodeStale, "the work item's criteria changed after the expectations were written; write them again", nil)
		}
	}
	now := s.state.now().UTC()
	expectation.Status, expectation.DecidedBy, expectation.UpdatedAt = decision, &current.UserID, now
	stampDecision(ctx, &expectation.DecisionID, &expectation.Delegated)
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Expectation{}, fault(CodeInternal, "start expectation decision", err)
	}
	defer tx.Rollback()
	saved, ok, hash, err := receipt[Expectation](ctx, tx, current.UserID, operation, request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	if err := saveExpectation(ctx, tx, expectation, false); err != nil {
		return Expectation{}, err
	}
	if decision == ExpectationRejected {
		if err := dropConsent(ctx, tx, expectation.ProjectID, expectation.WorkItemID); err != nil {
			return Expectation{}, err
		}
	}
	if err := saveReceipt(ctx, tx, current.UserID, operation, request.RequestID, hash, expectation, false, now); err != nil {
		return Expectation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Expectation{}, fault(CodeInternal, "commit expectation decision", err)
	}
	return expectation, nil
}

func (s *Service) GetExpectations(ctx context.Context, actor Principal, id ID) (Expectation, error) {
	expectation, err := loadExpectation(ctx, s.state.db, id)
	if err != nil {
		return Expectation{}, err
	}
	if _, err := authorizeProject(ctx, s.state.db, actor, expectation.ProjectID, CapabilityRead); err != nil {
		return Expectation{}, err
	}
	return expectation, nil
}

func (s *Service) ListExpectations(ctx context.Context, actor Principal, projectID ProjectID, workItemID string) ([]Expectation, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return nil, err
	}
	rows, err := s.state.db.QueryContext(ctx, `SELECT data_json FROM expectations WHERE project_id=? AND (?='' OR work_item_id=?) ORDER BY created_at DESC LIMIT 50`, projectID, workItemID, workItemID)
	if err != nil {
		return nil, fault(CodeInternal, "list expectations", err)
	}
	defer rows.Close()
	result := []Expectation{}
	for rows.Next() {
		var encoded string
		var expectation Expectation
		if err := rows.Scan(&encoded); err != nil {
			return nil, fault(CodeInternal, "read expectations", err)
		}
		if err := json.Unmarshal([]byte(encoded), &expectation); err != nil {
			return nil, fault(CodeInternal, "decode expectations", err)
		}
		result = append(result, expectation)
	}
	return result, rows.Err()
}

// admissionExpectations resolves the accepted expectations a job will start from. The
// suite must belong to the same work item and still match its current criteria.
func (s *Service) admissionExpectations(ctx context.Context, project Project, workItemID string, id ID, criteriaDigest string) (Expectation, error) {
	expectation, err := loadExpectation(ctx, s.state.db, id)
	if err != nil {
		return Expectation{}, err
	}
	if expectation.ProjectID != project.ID || expectation.WorkItemID != workItemID {
		return Expectation{}, fault(CodeInvalid, "expectations belong to a different work item", nil)
	}
	if expectation.Status != ExpectationAccepted {
		return Expectation{}, fault(CodeNotReady, "expectations must be accepted before a job can use them", nil)
	}
	if expectation.CriteriaSHA256 != criteriaDigest {
		return Expectation{}, fault(CodeStale, "the work item's criteria changed after its expectations were accepted", nil)
	}
	if _, err := s.resolveCommit(ctx, project.RepositoryPath, expectation.Commit); err != nil {
		return Expectation{}, err
	}
	return expectation, nil
}

// currentWorkItem reads the work item and its current criteria fingerprint.
func (s *Service) currentWorkItem(ctx context.Context, project Project, workItemID string) (orientation.WorkItem, string, error) {
	result, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return orientation.WorkItem{}, "", fault(CodeInternal, "locate work item", err)
	}
	work, err := selectWork(result, workItemID)
	if err != nil {
		return orientation.WorkItem{}, "", err
	}
	digest := ""
	if work.CriteriaSHA256 != nil {
		digest = *work.CriteriaSHA256
	}
	return work, digest, nil
}

func (s *Service) readExpectationReceipt(ctx context.Context, actor UserID, operation, requestID string, request any) (Expectation, bool, error) {
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Expectation{}, false, fault(CodeInternal, "read expectation receipt", err)
	}
	defer tx.Rollback()
	saved, ok, _, err := receipt[Expectation](ctx, tx, actor, operation, requestID, request)
	return saved, ok, err
}

func saveExpectation(ctx context.Context, tx *sql.Tx, expectation Expectation, insert bool) error {
	encoded, err := marshal(expectation)
	if err != nil {
		return fault(CodeInternal, "encode expectations", err)
	}
	if insert {
		_, err = tx.ExecContext(ctx, `INSERT INTO expectations(id, project_id, work_item_id, status, data_json, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`, expectation.ID, expectation.ProjectID, expectation.WorkItemID, expectation.Status, encoded, formatTime(expectation.CreatedAt), formatTime(expectation.UpdatedAt))
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fault(CodeConflict, "expectations are already being written for this work item", err)
		}
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE expectations SET status=?, data_json=?, updated_at=? WHERE id=?`, expectation.Status, encoded, formatTime(expectation.UpdatedAt), expectation.ID)
	}
	if err != nil {
		return fault(CodeInternal, "save expectations", err)
	}
	return nil
}

func loadExpectation(ctx context.Context, q queryer, id ID) (Expectation, error) {
	var encoded string
	err := q.QueryRowContext(ctx, `SELECT data_json FROM expectations WHERE id=?`, id).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return Expectation{}, fault(CodeNotFound, "expectations not found", nil)
	}
	if err != nil {
		return Expectation{}, fault(CodeInternal, "read expectations", err)
	}
	var expectation Expectation
	if err := json.Unmarshal([]byte(encoded), &expectation); err != nil {
		return Expectation{}, fault(CodeInternal, "decode expectations", err)
	}
	return expectation, nil
}

// failInterruptedExpectations marks suites left running by a stopped service.
func failInterruptedExpectations(ctx context.Context, db *sql.DB, now time.Time) error {
	rows, err := db.QueryContext(ctx, `SELECT data_json FROM expectations WHERE status=?`, ExpectationRunning)
	if err != nil {
		return err
	}
	var interrupted []Expectation
	for rows.Next() {
		var encoded string
		var expectation Expectation
		if err := rows.Scan(&encoded); err != nil {
			rows.Close()
			return err
		}
		if json.Unmarshal([]byte(encoded), &expectation) == nil {
			interrupted = append(interrupted, expectation)
		}
	}
	rows.Close()
	for _, expectation := range interrupted {
		expectation.Status, expectation.Error, expectation.UpdatedAt = ExpectationFailed, "the service stopped before the expectation author finished", now
		encoded, err := marshal(expectation)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `UPDATE expectations SET status=?, data_json=?, updated_at=? WHERE id=?`, expectation.Status, encoded, formatTime(now), expectation.ID); err != nil {
			return err
		}
	}
	return nil
}

func expectationPin(id ID) string { return "refs/waymark/expectations/" + string(id) }
