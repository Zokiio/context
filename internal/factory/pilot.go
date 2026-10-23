package factory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultPilotInterval = 5 * time.Second

// The autopilot takes the steps that need no decision and the decisions a person
// delegated: a go-ahead on a plan, or the project's ask policy. It acts as that person,
// and every record it writes says the decision was delegated and why. It does one step
// per work item per pass, builds one change per project at a time, and never retries a
// step that failed until the work item changes.
type pilotState struct {
	mu    sync.Mutex
	notes map[string]pilotNote
}

// pilotNote is the last step the autopilot could not take for a work item, keyed by
// the card it saw, so the board can show it and the autopilot does not repeat it.
type pilotNote struct {
	card    string
	message string
}

func (s *Service) runPilot() {
	defer s.state.refiners.Done()
	ticker := time.NewTicker(s.state.pilotInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.state.background.Done():
			return
		case <-ticker.C:
			s.pilotOnce(s.state.background)
		}
	}
}

// PilotOnce runs one autopilot pass over every project. The service runs passes on a
// timer when it is opened with Options.Pilot; tests call this directly.
func (s *Service) PilotOnce(ctx context.Context) {
	s.pilotOnce(ctx)
}

func (s *Service) pilotOnce(ctx context.Context) {
	rows, err := s.state.db.QueryContext(ctx, `SELECT id FROM projects ORDER BY created_at`)
	if err != nil {
		return
	}
	ids := []ProjectID{}
	for rows.Next() {
		var id ProjectID
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		project, err := readProject(ctx, s.state.db, id)
		if err != nil {
			continue
		}
		s.pilotProject(ctx, project)
	}
}

type pilotBusy struct {
	refining, writingTests, building bool
}

func (s *Service) pilotProject(ctx context.Context, project Project) {
	snapshot, err := s.boardSnapshot(ctx, project)
	if err != nil {
		return
	}
	busy := pilotBusy{}
	for _, in := range snapshot {
		if in.refinement != nil && in.refinement.Status == RefinementRunning {
			busy.refining = true
		}
		if in.expectation != nil && in.expectation.Status == ExpectationRunning {
			busy.writingTests = true
		}
		if in.job != nil && activeJob(in.job.Status) {
			busy.building = true
		}
	}
	for _, in := range snapshot {
		card, ok := deriveCard(in)
		if !ok || ctx.Err() != nil {
			continue
		}
		key := string(project.ID) + "/" + card.WorkItemID
		seen := cardKey(card)
		if note, ok := s.pilotNoteFor(key); ok && note.card == seen {
			continue
		}
		if err := s.pilotStep(ctx, project, in, card, &busy); err != nil {
			s.setPilotNote(key, pilotNote{card: seen, message: err.Error()})
		} else {
			s.clearPilotNote(key)
		}
	}
}

func activeJob(status JobStatus) bool {
	switch status {
	case JobAccepted, JobFailed, JobCancelled, JobStale:
		return false
	}
	return true
}

func cardKey(card BoardCard) string {
	return fmt.Sprintf("%s|%s|%s|%d|%s|%s", card.Column, card.State, card.Status, card.JobVersion, card.RefinementID, card.ExpectationID)
}

// pilotStep takes at most one step for one work item.
func (s *Service) pilotStep(ctx context.Context, project Project, in boardInputs, card BoardCard, busy *pilotBusy) error {
	policy := project.Policy.Ask
	workItemID := *in.work.ID
	if in.stopped || in.publishing || workArchived(in.work) || !workQueued(in.work) {
		return nil
	}
	if err := s.requireTaskQueued(ctx, s.state.db, project, workItemID); err != nil {
		return err
	}
	byConsent := in.consent != nil
	byPolicy := policy.SetBy != nil
	act := func(reason string, byPerson UserID, step func(context.Context, Principal) error) error {
		id, err := newID("dec_")
		if err != nil {
			return err
		}
		return step(withDecision(ctx, decisionNote{ID: id, Delegated: "autopilot: " + reason}), Principal{UserID: byPerson})
	}
	actor := func() (UserID, string, bool) {
		if byConsent {
			return in.consent.GivenBy, "your go-ahead on the plan", true
		}
		if byPolicy && policy.Autopilot {
			return *policy.SetBy, "the project's ask policy", true
		}
		return "", "", false
	}
	build := func(person UserID, reason string, tests *Expectation) error {
		if busy.building {
			return nil
		}
		settings := BuildSettings{}
		if byConsent {
			settings = in.consent.Build
		}
		err := act(reason, person, func(ctx context.Context, p Principal) error {
			_, err := s.startBuild(ctx, p, project, workItemID, tests, settings, fmt.Sprintf("pilot/build/%s/%s", workItemID, card.UpdatedAt.Format(time.RFC3339Nano)))
			return err
		})
		if err == nil {
			busy.building = true
			return s.clearConsent(ctx, project.ID, workItemID)
		}
		return err
	}

	if job := in.job; job != nil {
		switch {
		case job.Status == JobVerificationFailed && job.RepairCount < job.Admission.RepairLimit && card.State == CardRunning:
			person, reason, ok := actor()
			if !ok {
				return nil
			}
			return act(reason, person, func(ctx context.Context, p Principal) error {
				_, err := s.Action(ctx, p, JobActionRequest{RequestID: fmt.Sprintf("pilot/repair/%s/%d", job.ID, job.RepairCount), JobID: job.ID, IfVersion: job.Version, Action: ActionRepair})
				return err
			})
		case (job.Status == JobReadyForReview || job.Status == JobReadyForAcceptance) && approvesByPolicy(in):
			reason := "the project asks only when a requirement has no locked test, and every requirement has one"
			if policy.After() == AskAfterNever {
				reason = "the project merges without asking"
			}
			return act(reason, *policy.SetBy, func(ctx context.Context, p Principal) error {
				_, err := s.Approve(ctx, p, ApproveRequest{RequestID: fmt.Sprintf("pilot/approve/%s/%d", job.ID, job.Version), JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "Approved by the ask policy: " + reason + ".", Merge: true})
				return err
			})
		case job.Status == JobAccepted && job.Merge != nil && job.Merge.Status == MergeStatusFailed && retryableMerge(job.Merge.Failure) && mergeFailures(*job) < 2 && byPolicy && policy.Autopilot:
			return act("trying the merge again: it failed without a failing check", *policy.SetBy, func(ctx context.Context, p Principal) error {
				_, err := s.Merge(ctx, p, MergeRequest{RequestID: fmt.Sprintf("pilot/merge/%s/%d", job.ID, mergeFailures(*job)), JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, TargetBranch: job.Merge.Branch})
				return err
			})
		}
		return nil
	}
	if card.State != CardRunning && card.State != CardWaiting {
		return nil
	}
	if expectation := in.expectation; card.ExpectationID != "" && expectation != nil {
		person, reason, ok := actor()
		if !ok {
			return nil
		}
		switch {
		case expectation.Status == ExpectationProposed && card.State == CardRunning:
			return act(reason, person, func(ctx context.Context, p Principal) error {
				accepted, err := s.AcceptExpectations(ctx, p, DecideExpectationsRequest{RequestID: "pilot/accept-expectations/" + string(expectation.ID), ExpectationID: expectation.ID})
				if err != nil {
					return err
				}
				return build(person, reason, &accepted)
			})
		case expectation.Status == ExpectationAccepted && card.State == CardRunning:
			return build(person, reason, expectation)
		}
		return nil
	}
	if refinement := in.refinement; card.RefinementID != "" && refinement != nil {
		person, reason, ok := actor()
		if !ok || card.State != CardRunning {
			return nil
		}
		switch refinement.Status {
		case RefinementProposed:
			return act(reason, person, func(ctx context.Context, p Principal) error {
				_, err := s.AcceptRefinement(ctx, p, AcceptRefinementRequest{RequestID: "pilot/accept-refinement/" + string(refinement.ID), RefinementID: refinement.ID, Criteria: refinement.Proposal.Criteria, Answers: refinement.PreviousAnswers})
				return err
			})
		case RefinementAccepted:
			return s.pilotNextAfterPlan(ctx, project, workItemID, person, reason, byConsent, busy, act, build)
		}
		return nil
	}
	if card.Column == ColumnNext && card.Status == "Starts by itself" && byPolicy && policy.Autopilot {
		person := *policy.SetBy
		reason := "the project's autopilot starts ready work"
		if card.Flagged > 0 && len(s.state.refiner) > 0 {
			if busy.refining {
				return nil
			}
			busy.refining = true
			return act(reason, person, func(ctx context.Context, p Principal) error {
				_, err := s.StartRefinement(ctx, p, StartRefinementRequest{RequestID: "pilot/refine/" + workItemID + "/" + ticketOf(in), ProjectID: project.ID, WorkItemID: workItemID})
				return err
			})
		}
		if card.Flagged > 0 && policy.Before() != AskBeforeNever {
			// Vague requirements and no refiner: a person has to sharpen them.
			return nil
		}
		return s.pilotNextAfterPlan(ctx, project, workItemID, person, reason, false, busy, act, build)
	}
	return nil
}

// pilotNextAfterPlan starts the locked tests when the project can have them, or the
// build when the ask policy lets the factory skip the plan decision.
func (s *Service) pilotNextAfterPlan(ctx context.Context, project Project, workItemID string, person UserID, reason string, consented bool, busy *pilotBusy, act func(string, UserID, func(context.Context, Principal) error) error, build func(UserID, string, *Expectation) error) error {
	if s.canWriteExpectations(project) {
		if busy.writingTests {
			return nil
		}
		busy.writingTests = true
		return act(reason, person, func(ctx context.Context, p Principal) error {
			_, err := s.StartExpectations(ctx, p, StartExpectationsRequest{RequestID: fmt.Sprintf("pilot/expectations/%s/%d", workItemID, time.Now().UnixNano()), ProjectID: project.ID, WorkItemID: workItemID})
			return err
		})
	}
	if project.Policy.RequireExpectations || (!consented && project.Policy.Ask.Before() == AskBeforeAlways) {
		return nil
	}
	return build(person, reason, nil)
}

func ticketOf(in boardInputs) string {
	if in.work.TicketSHA256 == nil {
		return ""
	}
	return *in.work.TicketSHA256
}

// retryableMerge reports whether a failed merge might pass on a retry: the branch moved
// or a check stopped reporting, rather than a check failing on the merge commit.
func retryableMerge(failure string) bool {
	return !strings.HasPrefix(failure, "required check")
}

func (s *Service) pilotNoteFor(key string) (pilotNote, bool) {
	s.state.pilotNotes.mu.Lock()
	defer s.state.pilotNotes.mu.Unlock()
	note, ok := s.state.pilotNotes.notes[key]
	return note, ok
}

func (s *Service) setPilotNote(key string, note pilotNote) {
	s.state.pilotNotes.mu.Lock()
	defer s.state.pilotNotes.mu.Unlock()
	if s.state.pilotNotes.notes == nil {
		s.state.pilotNotes.notes = map[string]pilotNote{}
	}
	s.state.pilotNotes.notes[key] = note
}

func (s *Service) clearPilotNote(key string) {
	s.state.pilotNotes.mu.Lock()
	defer s.state.pilotNotes.mu.Unlock()
	delete(s.state.pilotNotes.notes, key)
}
