package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// BuildSettings are a build's options. Empty fields take the project's defaults: its
// first allowed executor, up to two fixes, and up to an hour.
type BuildSettings struct {
	ExecutorKey     string `json:"executorKey,omitempty"`
	RepairLimit     *int   `json:"repairLimit,omitempty"`
	DeadlineSeconds int64  `json:"deadlineSeconds,omitempty"`
}

// ApprovePlanRequest records the decision before the build: agreeing what done means.
// With a refinement and answers it sends the answers back to the refiner. With a
// refinement alone it accepts the proposal, or Criteria when given. With expectations
// it accepts the locked tests. It then starts the next step: the locked tests when the
// project can have them, otherwise the build.
type ApprovePlanRequest struct {
	RequestID     string             `json:"requestId"`
	ProjectID     ProjectID          `json:"projectId"`
	WorkItemID    string             `json:"workItemId"`
	RefinementID  ID                 `json:"refinementId,omitempty"`
	Answers       []RefinementAnswer `json:"answers,omitempty"`
	Criteria      []string           `json:"criteria,omitempty"`
	ExpectationID ID                 `json:"expectationId,omitempty"`
	// Continue lets the factory go on to the build without asking again, unless
	// something new comes up: a new question, a flagged criterion, or a concern from the
	// test author.
	Continue bool          `json:"continue"`
	Build    BuildSettings `json:"build"`
}

type PlanResult struct {
	Refinement  *Refinement  `json:"refinement,omitempty"`
	Expectation *Expectation `json:"expectation,omitempty"`
	Job         *Job         `json:"job,omitempty"`
	// Waiting says why the build has not started yet.
	Waiting string `json:"waiting,omitempty"`
}

// planConsent is a person's go-ahead to continue to the build without asking again.
type planConsent struct {
	GivenBy   UserID        `json:"givenBy"`
	Build     BuildSettings `json:"build"`
	Decision  string        `json:"decision"`
	CreatedAt time.Time     `json:"createdAt"`
}

func (s *Service) ApprovePlan(ctx context.Context, actor Principal, request ApprovePlanRequest) (PlanResult, error) {
	if request.ProjectID == "" || strings.TrimSpace(request.WorkItemID) == "" {
		return PlanResult{}, fault(CodeInvalid, "projectId and workItemId are required", nil)
	}
	if err := requireRequestID(request.RequestID); err != nil {
		return PlanResult{}, err
	}
	current, err := authorizeProject(ctx, s.state.db, actor, request.ProjectID, CapabilityCoordinate)
	if err != nil {
		return PlanResult{}, err
	}
	if current.Kind != UserKindHuman {
		return PlanResult{}, fault(CodeForbidden, "only a person can agree what done means", nil)
	}
	ctx = decisionContext(ctx, current.UserID, "plan", request.RequestID)
	note, _ := decisionFrom(ctx)
	project, err := readProject(ctx, s.state.db, request.ProjectID)
	if err != nil {
		return PlanResult{}, err
	}
	if err := s.requireTaskQueued(ctx, s.state.db, project, request.WorkItemID); err != nil {
		return PlanResult{}, err
	}
	consent := planConsent{GivenBy: current.UserID, Build: request.Build, Decision: note.ID, CreatedAt: s.state.now().UTC()}
	remember := func() error {
		if request.Continue {
			return s.saveConsent(ctx, project.ID, request.WorkItemID, consent)
		}
		return s.clearConsent(ctx, project.ID, request.WorkItemID)
	}
	result := PlanResult{}
	switch {
	case request.RefinementID != "" && len(request.Answers) > 0:
		refinement, err := s.StartRefinement(ctx, current, StartRefinementRequest{RequestID: request.RequestID + "/refine", ProjectID: project.ID, WorkItemID: request.WorkItemID, PreviousID: request.RefinementID, Answers: request.Answers})
		if err != nil {
			return PlanResult{}, err
		}
		result.Refinement, result.Waiting = &refinement, "The agent is sharpening the requirements with your answers."
		return result, remember()
	case request.RefinementID != "":
		refinement, err := s.GetRefinement(ctx, current, request.RefinementID)
		if err != nil {
			return PlanResult{}, err
		}
		criteria := request.Criteria
		if len(criteria) == 0 && refinement.Proposal != nil {
			criteria = refinement.Proposal.Criteria
		}
		accepted, err := s.AcceptRefinement(ctx, current, AcceptRefinementRequest{RequestID: request.RequestID + "/accept-refinement", RefinementID: refinement.ID, Criteria: criteria, Answers: refinement.PreviousAnswers})
		if err != nil {
			return PlanResult{}, err
		}
		result.Refinement = &accepted
	case request.ExpectationID != "":
		expectation, err := s.GetExpectations(ctx, current, request.ExpectationID)
		if err != nil {
			return PlanResult{}, err
		}
		if expectation.Status != ExpectationAccepted {
			if expectation, err = s.AcceptExpectations(ctx, current, DecideExpectationsRequest{RequestID: request.RequestID + "/accept-expectations", ExpectationID: request.ExpectationID}); err != nil {
				return PlanResult{}, err
			}
		}
		result.Expectation = &expectation
	}
	if result.Expectation == nil && s.canWriteExpectations(project) {
		started, err := s.StartExpectations(ctx, current, StartExpectationsRequest{RequestID: request.RequestID + "/expectations", ProjectID: project.ID, WorkItemID: request.WorkItemID})
		if err != nil {
			return result, err
		}
		result.Expectation, result.Waiting = &started, "A separate agent is writing the locked tests."
		return result, remember()
	}
	if project.Policy.RequireExpectations && result.Expectation == nil {
		return result, fault(CodeNotReady, "this project builds only from locked tests, and no test author is configured", nil)
	}
	job, err := s.startBuild(ctx, current, project, request.WorkItemID, result.Expectation, request.Build, request.RequestID+"/build")
	if ErrorCodeOf(err) == CodeNotReady {
		// A prerequisite is not merged yet. Remember the go-ahead so the build starts
		// when it is.
		consent.Build = request.Build
		if saveErr := s.saveConsent(ctx, project.ID, request.WorkItemID, consent); saveErr != nil {
			return result, saveErr
		}
		result.Waiting = err.Error()
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Job = &job
	return result, s.clearConsent(ctx, project.ID, request.WorkItemID)
}

func (s *Service) canWriteExpectations(project Project) bool {
	return len(project.Policy.ProtectedPaths) > 0 && len(s.state.expectationAuthor) > 0
}

// startBuild submits a job with the build settings, filling in project defaults. A job
// already active for the work item is returned instead, so a retried decision does not
// start a second build.
func (s *Service) startBuild(ctx context.Context, actor Principal, project Project, workItemID string, expectation *Expectation, settings BuildSettings, requestID string) (Job, error) {
	var active JobID
	err := s.state.db.QueryRowContext(ctx, `SELECT id FROM jobs WHERE project_id=? AND work_item_id=? AND active_claim=1`, project.ID, workItemID).Scan(&active)
	if err == nil {
		return s.GetJob(ctx, actor, active)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Job{}, fault(CodeInternal, "check active build", err)
	}
	executor := settings.ExecutorKey
	if executor == "" {
		if len(project.Policy.AllowedExecutors) == 0 {
			return Job{}, fault(CodeNotReady, "the project allows no executor yet", nil)
		}
		executor = project.Policy.AllowedExecutors[0]
	}
	repairs := min(2, project.Policy.MaxRepairAttempts)
	if settings.RepairLimit != nil {
		repairs = *settings.RepairLimit
	}
	seconds := int64(3600)
	if settings.DeadlineSeconds > 0 {
		seconds = settings.DeadlineSeconds
	}
	seconds = min(seconds, project.Policy.MaxJobDurationSeconds)
	request := SubmitRequest{RequestID: requestID, ProjectID: project.ID, WorkItemID: workItemID, BaseRevision: "HEAD", ExecutorKey: executor, RepairLimit: repairs, Deadline: s.state.now().UTC().Add(time.Duration(seconds)*time.Second - time.Second)}
	if expectation != nil {
		request.ExpectationsID, request.BaseRevision = expectation.ID, ""
	}
	return s.Submit(ctx, actor, request)
}

func (s *Service) saveConsent(ctx context.Context, projectID ProjectID, workItemID string, consent planConsent) error {
	encoded, err := marshal(consent)
	if err != nil {
		return fault(CodeInternal, "encode consent", err)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	project, err := readProject(ctx, s.state.db, projectID)
	if err != nil {
		return err
	}
	if err := s.requireTaskQueued(ctx, s.state.db, project, workItemID); err != nil {
		return err
	}
	if _, err := s.state.db.ExecContext(ctx, `INSERT INTO plan_consents(project_id, work_item_id, data_json, created_at) VALUES(?,?,?,?) ON CONFLICT(project_id, work_item_id) DO UPDATE SET data_json=excluded.data_json, created_at=excluded.created_at`, projectID, workItemID, encoded, formatTime(consent.CreatedAt)); err != nil {
		return fault(CodeInternal, "save consent", err)
	}
	return nil
}

func (s *Service) clearConsent(ctx context.Context, projectID ProjectID, workItemID string) error {
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	if _, err := s.state.db.ExecContext(ctx, `DELETE FROM plan_consents WHERE project_id=? AND work_item_id=?`, projectID, workItemID); err != nil {
		return fault(CodeInternal, "clear consent", err)
	}
	return nil
}

// dropConsent ends a go-ahead inside a transaction that already holds the mutation lock.
func dropConsent(ctx context.Context, tx *sql.Tx, projectID ProjectID, workItemID string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM plan_consents WHERE project_id=? AND work_item_id=?`, projectID, workItemID); err != nil {
		return fault(CodeInternal, "clear consent", err)
	}
	return nil
}

func loadConsents(ctx context.Context, q queryer, projectID ProjectID) (map[string]planConsent, error) {
	rows, err := q.QueryContext(ctx, `SELECT work_item_id, data_json FROM plan_consents WHERE project_id=?`, projectID)
	if err != nil {
		return nil, fault(CodeInternal, "read consents", err)
	}
	defer rows.Close()
	result := map[string]planConsent{}
	for rows.Next() {
		var workItemID, encoded string
		var consent planConsent
		if err := rows.Scan(&workItemID, &encoded); err != nil {
			return nil, fault(CodeInternal, "decode consent", err)
		}
		if json.Unmarshal([]byte(encoded), &consent) == nil {
			result[workItemID] = consent
		}
	}
	return result, rows.Err()
}
