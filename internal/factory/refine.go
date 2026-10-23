package factory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/taskcontext"
)

// A refinement proposes rewritten acceptance criteria and a few blocking questions
// for one work item before admission freezes its requirements. A refiner command
// drafts the proposal without editing anything; a person answers, edits, and accepts
// it, and only then is the ticket rewritten, through an expected-revision write.

type RefinementStatus string

const (
	RefinementRunning    RefinementStatus = "running"
	RefinementProposed   RefinementStatus = "proposed"
	RefinementFailed     RefinementStatus = "failed"
	RefinementAccepted   RefinementStatus = "accepted"
	RefinementRejected   RefinementStatus = "rejected"
	RefinementSuperseded RefinementStatus = "superseded"
	RefinementStale      RefinementStatus = "stale"
)

const (
	MaxRefinementQuestions = 3
	maxRefinedCriteria     = 20
	maxRefinementRunes     = 1000
	refinerTimeout         = 10 * time.Minute
	maxRefinerOutput       = 2 << 20
	maxRefinerLog          = 64 << 10
)

type RefinementQuestion struct {
	Question string `json:"question"`
	Unblocks string `json:"unblocks"`
}

type RefinementAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type RefinementProposal struct {
	Criteria  []string             `json:"criteria"`
	Questions []RefinementQuestion `json:"questions"`
	Notes     string               `json:"notes,omitempty"`
}

// RefinerRun is what the refiner reported about its own execution. It is an
// observation, not evidence that the proposal is right.
type RefinerRun struct {
	Model     string  `json:"model,omitempty"`
	SessionID string  `json:"sessionId,omitempty"`
	Turns     int     `json:"turns,omitempty"`
	CostUSD   float64 `json:"costUsd,omitempty"`
}

type Refinement struct {
	ID                  ID                  `json:"id"`
	ProjectID           ProjectID           `json:"projectId"`
	WorkItemID          string              `json:"workItemId"`
	Title               string              `json:"title"`
	TicketPath          string              `json:"ticketPath"`
	TicketSHA256        string              `json:"ticketSHA256"`
	Status              RefinementStatus    `json:"status"`
	RequestedBy         UserID              `json:"requestedBy"`
	PreviousID          ID                  `json:"previousId,omitempty"`
	PreviousAnswers     []RefinementAnswer  `json:"previousAnswers,omitempty"`
	Criteria            []string            `json:"criteria"`
	Before              *CriterionSignals   `json:"before,omitempty"`
	Proposal            *RefinementProposal `json:"proposal,omitempty"`
	After               *CriterionSignals   `json:"after,omitempty"`
	Run                 *RefinerRun         `json:"run,omitempty"`
	Log                 string              `json:"log,omitempty"`
	Error               string              `json:"error,omitempty"`
	AcceptedCriteria    []string            `json:"acceptedCriteria,omitempty"`
	Answers             []RefinementAnswer  `json:"answers,omitempty"`
	DecidedBy           *UserID             `json:"decidedBy,omitempty"`
	UpdatedTicketSHA256 string              `json:"updatedTicketSHA256,omitempty"`
	CreatedAt           time.Time           `json:"createdAt"`
	UpdatedAt           time.Time           `json:"updatedAt"`
}

type StartRefinementRequest struct {
	RequestID  string             `json:"requestId"`
	ProjectID  ProjectID          `json:"projectId"`
	WorkItemID string             `json:"workItemId"`
	PreviousID ID                 `json:"previousId,omitempty"`
	Answers    []RefinementAnswer `json:"answers,omitempty"`
}

type AcceptRefinementRequest struct {
	RequestID    string             `json:"requestId"`
	RefinementID ID                 `json:"refinementId"`
	Criteria     []string           `json:"criteria"`
	Answers      []RefinementAnswer `json:"answers,omitempty"`
}

type RejectRefinementRequest struct {
	RequestID    string `json:"requestId"`
	RefinementID ID     `json:"refinementId"`
}

// RefinerInput is written to the refiner command's standard input.
type RefinerInput struct {
	RefinementID     ID                  `json:"refinementId"`
	WorkItemID       string              `json:"workItemId"`
	Title            string              `json:"title"`
	TicketPath       string              `json:"ticketPath"`
	Ticket           string              `json:"ticket"`
	Criteria         []string            `json:"criteria"`
	Signals          *CriterionSignals   `json:"signals,omitempty"`
	Context          []ContextSource     `json:"context"`
	ProtectedPaths   []string            `json:"protectedPaths,omitempty"`
	PreviousProposal *RefinementProposal `json:"previousProposal,omitempty"`
	PreviousAnswers  []RefinementAnswer  `json:"previousAnswers,omitempty"`
	MaxQuestions     int                 `json:"maxQuestions"`
}

// RefinerOutput is the single JSON object a refiner command writes to standard output.
type RefinerOutput struct {
	Proposal RefinementProposal `json:"proposal"`
	Run      RefinerRun         `json:"run"`
}

func (s *Service) StartRefinement(ctx context.Context, actor Principal, request StartRefinementRequest) (Refinement, error) {
	if request.ProjectID == "" || strings.TrimSpace(request.WorkItemID) == "" {
		return Refinement{}, fault(CodeInvalid, "projectId and workItemId are required", nil)
	}
	if err := requireRequestID(request.RequestID); err != nil {
		return Refinement{}, err
	}
	if len(s.state.refiner) == 0 {
		return Refinement{}, fault(CodeNotReady, "no refiner is configured; start the service with a refiner command", nil)
	}
	answers, err := normalizeAnswers(request.Answers)
	if err != nil {
		return Refinement{}, err
	}
	current, err := authorizeProject(ctx, s.state.db, actor, request.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Refinement{}, err
	}
	if replayed, ok, err := s.readRefinementReceipt(ctx, current.UserID, "start_refinement", request.RequestID, request); err != nil || ok {
		return replayed, err
	}
	project, err := readProject(ctx, s.state.db, request.ProjectID)
	if err != nil {
		return Refinement{}, err
	}
	result, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return Refinement{}, fault(CodeInternal, "locate work item", err)
	}
	work, err := selectWork(result, request.WorkItemID)
	if err != nil {
		return Refinement{}, err
	}
	if err := requireNoActiveJob(ctx, s.state.db, project.ID, request.WorkItemID); err != nil {
		return Refinement{}, err
	}
	raw, err := os.ReadFile(work.Source)
	if err != nil {
		return Refinement{}, fault(CodeInternal, "read work item", err)
	}
	criteria, err := acceptanceCriteria(work.Source)
	if err != nil {
		return Refinement{}, fault(CodeInvalid, "read acceptance criteria", err)
	}
	title := ""
	if work.Title != nil {
		title = *work.Title
	}
	var previous *Refinement
	if request.PreviousID != "" {
		found, err := loadRefinement(ctx, s.state.db, request.PreviousID)
		if err != nil {
			return Refinement{}, err
		}
		if found.ProjectID != project.ID || found.WorkItemID != request.WorkItemID || found.Status != RefinementProposed {
			return Refinement{}, fault(CodeConflict, "previous refinement must be a current proposal for the same work item", nil)
		}
		previous = &found
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
	id, err := newID("ref_")
	if err != nil {
		return Refinement{}, fault(CodeInternal, "create refinement identity", err)
	}
	now := s.state.now().UTC()
	refinement := Refinement{ID: ID(id), ProjectID: project.ID, WorkItemID: request.WorkItemID, Title: title, TicketPath: work.Source, TicketSHA256: digestBytes(raw), Status: RefinementRunning, RequestedBy: current.UserID, PreviousID: request.PreviousID, PreviousAnswers: answers, Criteria: criteria, Before: s.scoreCriteria(ctx, title, criteria, true), CreatedAt: now, UpdatedAt: now}
	input := RefinerInput{RefinementID: refinement.ID, WorkItemID: request.WorkItemID, Title: title, TicketPath: work.Source, Ticket: string(raw), Criteria: criteria, Signals: refinement.Before, Context: sources, ProtectedPaths: project.Policy.ProtectedPaths, PreviousAnswers: answers, MaxQuestions: MaxRefinementQuestions}
	if previous != nil {
		input.PreviousProposal = previous.Proposal
	}

	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Refinement{}, fault(CodeInternal, "start refinement transaction", err)
	}
	defer tx.Rollback()
	if saved, ok, hash, err := receipt[Refinement](ctx, tx, current.UserID, "start_refinement", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		if err := requireNoActiveJob(ctx, tx, project.ID, request.WorkItemID); err != nil {
			return Refinement{}, err
		}
		if err := saveRefinement(ctx, tx, refinement, true); err != nil {
			return Refinement{}, err
		}
		if previous != nil {
			previous.Status, previous.UpdatedAt = RefinementSuperseded, now
			if err := saveRefinement(ctx, tx, *previous, false); err != nil {
				return Refinement{}, err
			}
		}
		if err := saveReceipt(ctx, tx, current.UserID, "start_refinement", request.RequestID, hash, refinement, false, now); err != nil {
			return Refinement{}, err
		}
		if err := tx.Commit(); err != nil {
			return Refinement{}, fault(CodeInternal, "commit refinement", err)
		}
	}
	s.state.refiners.Add(1)
	go s.runRefiner(refinement.ID, input, project.RepositoryPath)
	return refinement, nil
}

// runRefiner executes the configured refiner outside any transaction. The service
// waits for it on close, so an interrupted run is recorded as failed.
func (s *Service) runRefiner(id ID, input RefinerInput, directory string) {
	defer s.state.refiners.Done()
	ctx, cancel := context.WithTimeout(s.state.background, refinerTimeout)
	defer cancel()
	output, log, err := s.execRefiner(ctx, directory, input)
	if err == nil {
		err = normalizeProposal(&output.Proposal)
	}
	var after *CriterionSignals
	if err == nil {
		after = s.scoreCriteria(ctx, input.Title, output.Proposal.Criteria, true)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	refinement, loadErr := loadRefinement(context.Background(), s.state.db, id)
	if loadErr != nil || refinement.Status != RefinementRunning {
		return
	}
	refinement.Log, refinement.UpdatedAt = log, s.state.now().UTC()
	if err != nil {
		refinement.Status, refinement.Error = RefinementFailed, truncateRunes(err.Error(), maxRefinementRunes)
	} else {
		refinement.Status, refinement.Proposal, refinement.After, refinement.Run = RefinementProposed, &output.Proposal, after, &output.Run
	}
	tx, err := s.state.db.BeginTx(context.Background(), nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if saveRefinement(context.Background(), tx, refinement, false) == nil {
		_ = tx.Commit()
	}
}

func (s *Service) execRefiner(ctx context.Context, directory string, input RefinerInput) (RefinerOutput, string, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return RefinerOutput{}, "", err
	}
	command := exec.CommandContext(ctx, s.state.refiner[0], s.state.refiner[1:]...)
	command.Dir = directory
	command.Stdin = bytes.NewReader(encoded)
	command.WaitDelay = 5 * time.Second
	stdout := &limitedBuffer{limit: maxRefinerOutput}
	stderr := &limitedBuffer{limit: maxRefinerLog}
	command.Stdout, command.Stderr = stdout, stderr
	runErr := command.Run()
	log := stderr.String()
	if ctx.Err() != nil {
		return RefinerOutput{}, log, fmt.Errorf("refiner stopped: %w", ctx.Err())
	}
	if runErr != nil {
		return RefinerOutput{}, log, fmt.Errorf("refiner failed: %w", runErr)
	}
	var output RefinerOutput
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &output); err != nil {
		return RefinerOutput{}, log, fmt.Errorf("refiner output is not a proposal: %w", err)
	}
	return output, log, nil
}

func (s *Service) AcceptRefinement(ctx context.Context, actor Principal, request AcceptRefinementRequest) (Refinement, error) {
	if err := requireRequestID(request.RequestID); err != nil {
		return Refinement{}, err
	}
	criteria, err := normalizeCriteria(request.Criteria)
	if err != nil {
		return Refinement{}, err
	}
	answers, err := normalizeAnswers(request.Answers)
	if err != nil {
		return Refinement{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Refinement{}, fault(CodeInternal, "start refinement acceptance", err)
	}
	defer tx.Rollback()
	refinement, err := loadRefinement(ctx, tx, request.RefinementID)
	if err != nil {
		return Refinement{}, err
	}
	current, err := authorizeProject(ctx, tx, actor, refinement.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Refinement{}, err
	}
	saved, ok, hash, err := receipt[Refinement](ctx, tx, current.UserID, "accept_refinement", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	if refinement.Status != RefinementProposed {
		return Refinement{}, fault(CodeConflict, "only a current proposal can be accepted", nil)
	}
	if err := requireNoActiveJob(ctx, tx, refinement.ProjectID, refinement.WorkItemID); err != nil {
		return Refinement{}, err
	}
	now := s.state.now().UTC()
	raw, err := os.ReadFile(refinement.TicketPath)
	if err != nil {
		return Refinement{}, fault(CodeInternal, "read work item", err)
	}
	if digestBytes(raw) != refinement.TicketSHA256 {
		refinement.Status, refinement.UpdatedAt = RefinementStale, now
		if err := saveRefinement(ctx, tx, refinement, false); err != nil {
			return Refinement{}, err
		}
		if err := tx.Commit(); err != nil {
			return Refinement{}, fault(CodeInternal, "record stale refinement", err)
		}
		return Refinement{}, fault(CodeStale, "the work item changed after refinement started; refine it again", nil)
	}
	model := ""
	if refinement.Run != nil {
		model = refinement.Run.Model
	}
	updated, err := rewriteCriteria(raw, refinement.TicketPath, criteria, refinementComment(refinement.ID, current.Username, model, now, answers))
	if err != nil {
		return Refinement{}, err
	}
	if err := writeFileCAS(refinement.TicketPath, raw, updated); err != nil {
		return Refinement{}, err
	}
	refinement.Status, refinement.AcceptedCriteria, refinement.Answers = RefinementAccepted, criteria, answers
	refinement.DecidedBy, refinement.UpdatedTicketSHA256, refinement.UpdatedAt = &current.UserID, digestBytes(updated), now
	if err := saveRefinement(ctx, tx, refinement, false); err != nil {
		return Refinement{}, err
	}
	if err := saveReceipt(ctx, tx, current.UserID, "accept_refinement", request.RequestID, hash, refinement, false, now); err != nil {
		return Refinement{}, err
	}
	if err := tx.Commit(); err != nil {
		return Refinement{}, fault(CodeInternal, "commit refinement acceptance", err)
	}
	return refinement, nil
}

func (s *Service) RejectRefinement(ctx context.Context, actor Principal, request RejectRefinementRequest) (Refinement, error) {
	if err := requireRequestID(request.RequestID); err != nil {
		return Refinement{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Refinement{}, fault(CodeInternal, "start refinement rejection", err)
	}
	defer tx.Rollback()
	refinement, err := loadRefinement(ctx, tx, request.RefinementID)
	if err != nil {
		return Refinement{}, err
	}
	current, err := authorizeProject(ctx, tx, actor, refinement.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Refinement{}, err
	}
	saved, ok, hash, err := receipt[Refinement](ctx, tx, current.UserID, "reject_refinement", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	if refinement.Status != RefinementProposed {
		return Refinement{}, fault(CodeConflict, "only a current proposal can be rejected", nil)
	}
	now := s.state.now().UTC()
	refinement.Status, refinement.DecidedBy, refinement.UpdatedAt = RefinementRejected, &current.UserID, now
	if err := saveRefinement(ctx, tx, refinement, false); err != nil {
		return Refinement{}, err
	}
	if err := saveReceipt(ctx, tx, current.UserID, "reject_refinement", request.RequestID, hash, refinement, false, now); err != nil {
		return Refinement{}, err
	}
	if err := tx.Commit(); err != nil {
		return Refinement{}, fault(CodeInternal, "commit refinement rejection", err)
	}
	return refinement, nil
}

func (s *Service) GetRefinement(ctx context.Context, actor Principal, id ID) (Refinement, error) {
	refinement, err := loadRefinement(ctx, s.state.db, id)
	if err != nil {
		return Refinement{}, err
	}
	if _, err := authorizeProject(ctx, s.state.db, actor, refinement.ProjectID, CapabilityRead); err != nil {
		return Refinement{}, err
	}
	return refinement, nil
}

// Refinements lists a project's refinements, newest first, optionally for one work item.
func (s *Service) Refinements(ctx context.Context, actor Principal, projectID ProjectID, workItemID string) ([]Refinement, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return nil, err
	}
	rows, err := s.state.db.QueryContext(ctx, `SELECT data_json FROM refinements WHERE project_id=? AND (?='' OR work_item_id=?) ORDER BY created_at DESC LIMIT 50`, projectID, workItemID, workItemID)
	if err != nil {
		return nil, fault(CodeInternal, "list refinements", err)
	}
	defer rows.Close()
	refinements := []Refinement{}
	for rows.Next() {
		var encoded string
		var refinement Refinement
		if err := rows.Scan(&encoded); err != nil {
			return nil, fault(CodeInternal, "read refinement", err)
		}
		if err := json.Unmarshal([]byte(encoded), &refinement); err != nil {
			return nil, fault(CodeInternal, "decode refinement", err)
		}
		refinements = append(refinements, refinement)
	}
	return refinements, rows.Err()
}

func (s *Service) readRefinementReceipt(ctx context.Context, actor UserID, operation, requestID string, request any) (Refinement, bool, error) {
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Refinement{}, false, fault(CodeInternal, "read refinement receipt", err)
	}
	defer tx.Rollback()
	saved, ok, _, err := receipt[Refinement](ctx, tx, actor, operation, requestID, request)
	return saved, ok, err
}

func requireNoActiveJob(ctx context.Context, q queryer, projectID ProjectID, workItemID string) error {
	var existing JobID
	err := q.QueryRowContext(ctx, `SELECT id FROM jobs WHERE project_id=? AND work_item_id=? AND active_claim=1`, projectID, workItemID).Scan(&existing)
	if err == nil {
		return detailFault(CodeConflict, "an active job froze this work item's criteria", map[string]any{"jobId": existing}, nil)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fault(CodeInternal, "inspect active job", err)
	}
	return nil
}

func saveRefinement(ctx context.Context, tx *sql.Tx, refinement Refinement, insert bool) error {
	encoded, err := marshal(refinement)
	if err != nil {
		return fault(CodeInternal, "encode refinement", err)
	}
	if insert {
		_, err = tx.ExecContext(ctx, `INSERT INTO refinements(id, project_id, work_item_id, status, data_json, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`, refinement.ID, refinement.ProjectID, refinement.WorkItemID, refinement.Status, encoded, formatTime(refinement.CreatedAt), formatTime(refinement.UpdatedAt))
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fault(CodeConflict, "a refinement is already running for this work item", err)
		}
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE refinements SET status=?, data_json=?, updated_at=? WHERE id=?`, refinement.Status, encoded, formatTime(refinement.UpdatedAt), refinement.ID)
	}
	if err != nil {
		return fault(CodeInternal, "save refinement", err)
	}
	return nil
}

func loadRefinement(ctx context.Context, q queryer, id ID) (Refinement, error) {
	var encoded string
	err := q.QueryRowContext(ctx, `SELECT data_json FROM refinements WHERE id=?`, id).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return Refinement{}, fault(CodeNotFound, "refinement not found", nil)
	}
	if err != nil {
		return Refinement{}, fault(CodeInternal, "read refinement", err)
	}
	var refinement Refinement
	if err := json.Unmarshal([]byte(encoded), &refinement); err != nil {
		return Refinement{}, fault(CodeInternal, "decode refinement", err)
	}
	return refinement, nil
}

// failInterruptedRefinements marks refinements left running by a stopped service.
func failInterruptedRefinements(ctx context.Context, db *sql.DB, now time.Time) error {
	rows, err := db.QueryContext(ctx, `SELECT data_json FROM refinements WHERE status=?`, RefinementRunning)
	if err != nil {
		return err
	}
	var interrupted []Refinement
	for rows.Next() {
		var encoded string
		var refinement Refinement
		if err := rows.Scan(&encoded); err != nil {
			rows.Close()
			return err
		}
		if json.Unmarshal([]byte(encoded), &refinement) == nil {
			interrupted = append(interrupted, refinement)
		}
	}
	rows.Close()
	for _, refinement := range interrupted {
		refinement.Status, refinement.Error, refinement.UpdatedAt = RefinementFailed, "the service stopped before the refiner finished", now
		encoded, err := marshal(refinement)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `UPDATE refinements SET status=?, data_json=?, updated_at=? WHERE id=?`, refinement.Status, encoded, formatTime(now), refinement.ID); err != nil {
			return err
		}
	}
	return nil
}

func normalizeProposal(proposal *RefinementProposal) error {
	criteria, err := normalizeCriteria(proposal.Criteria)
	if err != nil {
		return err
	}
	proposal.Criteria = criteria
	questions := []RefinementQuestion{}
	for _, question := range proposal.Questions {
		question.Question = truncateRunes(oneLine(question.Question), maxRefinementRunes)
		question.Unblocks = truncateRunes(oneLine(question.Unblocks), maxRefinementRunes)
		if question.Question != "" {
			questions = append(questions, question)
		}
	}
	if extra := len(questions) - MaxRefinementQuestions; extra > 0 {
		questions = questions[:MaxRefinementQuestions]
		proposal.Notes = strings.TrimSpace(proposal.Notes + fmt.Sprintf(" (%d further questions were dropped; at most %d are kept.)", extra, MaxRefinementQuestions))
	}
	proposal.Questions = questions
	proposal.Notes = truncateRunes(strings.TrimSpace(proposal.Notes), 4*maxRefinementRunes)
	return nil
}

func normalizeCriteria(input []string) ([]string, error) {
	criteria := []string{}
	for _, criterion := range input {
		if criterion = oneLine(criterion); criterion != "" {
			if len([]rune(criterion)) > maxRefinementRunes {
				return nil, fault(CodeInvalid, "an acceptance criterion is too long", nil)
			}
			criteria = append(criteria, criterion)
		}
	}
	if len(criteria) == 0 || len(criteria) > maxRefinedCriteria {
		return nil, fault(CodeInvalid, fmt.Sprintf("a refinement needs 1 to %d acceptance criteria", maxRefinedCriteria), nil)
	}
	return criteria, nil
}

func normalizeAnswers(input []RefinementAnswer) ([]RefinementAnswer, error) {
	answers := []RefinementAnswer{}
	for _, answer := range input {
		answer.Question, answer.Answer = oneLine(answer.Question), oneLine(answer.Answer)
		if answer.Answer == "" {
			continue
		}
		if len([]rune(answer.Question)) > maxRefinementRunes || len([]rune(answer.Answer)) > maxRefinementRunes {
			return nil, fault(CodeInvalid, "a refinement answer is too long", nil)
		}
		answers = append(answers, answer)
	}
	if len(answers) > MaxRefinementQuestions {
		return nil, fault(CodeInvalid, fmt.Sprintf("at most %d answers are accepted", MaxRefinementQuestions), nil)
	}
	return answers, nil
}

// rewriteCriteria replaces the single Acceptance criteria section and appends a
// comment that records how the criteria changed.
func rewriteCriteria(raw []byte, path string, criteria []string, comment string) ([]byte, error) {
	document, err := recordread.ParseDocument(raw)
	if err != nil || !bytes.HasSuffix(raw, document.Body) {
		return nil, fault(CodeInvalid, "work item cannot be parsed for rewriting", err)
	}
	offset := len(raw) - len(document.Body)
	sections := recordread.Sections(document.Body, path, map[string]string{"Acceptance criteria": "criteria"}).Sections
	if len(sections) != 1 {
		return nil, fault(CodeInvalid, "work item must have exactly one Acceptance criteria section to rewrite; edit it by hand", nil)
	}
	section := sections[0]
	bullet := "- "
	if strings.Contains(section.Content, "- [ ]") || strings.Contains(strings.ToLower(section.Content), "- [x]") {
		bullet = "- [ ] "
	}
	var block strings.Builder
	block.WriteString("\n")
	for _, criterion := range criteria {
		block.WriteString(bullet + criterion + "\n")
	}
	block.WriteString("\n")
	start, end := offset+section.ContentStart, offset+section.End
	updated := append(append(append([]byte{}, raw[:start]...), block.String()...), raw[end:]...)

	document, err = recordread.ParseDocument(updated)
	if err != nil || !bytes.HasSuffix(updated, document.Body) {
		return nil, fault(CodeInternal, "rewritten work item cannot be parsed", err)
	}
	offset = len(updated) - len(document.Body)
	comments := recordread.Sections(document.Body, path, map[string]string{"Comments": "comments"}).Sections
	if len(comments) == 0 {
		trimmed := bytes.TrimRight(updated, "\n")
		return append(append(trimmed, "\n\n## Comments\n\n"...), comment...), nil
	}
	last := comments[len(comments)-1]
	insert := offset + last.End
	before := bytes.TrimRight(updated[:insert], "\n")
	after := updated[insert:]
	result := append(append(append([]byte{}, before...), "\n\n"...), comment...)
	if len(after) > 0 {
		result = append(append(result, '\n'), after...)
	}
	return result, nil
}

func refinementComment(id ID, username, model string, now time.Time, answers []RefinementAnswer) string {
	var b strings.Builder
	proposer := "a refiner"
	if model != "" {
		proposer = model
	}
	fmt.Fprintf(&b, "- %s: %s accepted refined acceptance criteria proposed by %s (Waymark refinement %s).\n", now.Format("2006-01-02"), username, proposer, id)
	for _, answer := range answers {
		fmt.Fprintf(&b, "  - Q: %s A: %s\n", answer.Question, answer.Answer)
	}
	return b.String()
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func truncateRunes(value string, limit int) string {
	if runes := []rune(value); len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

type limitedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.Len(); room < len(p) {
		if room > 0 {
			b.Buffer.Write(p[:room])
		}
		b.truncated = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}
