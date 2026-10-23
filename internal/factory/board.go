package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Zokiio/context/internal/orientation"
)

// The board shows every open work item in one of six columns. A person acts only on
// cards that need them; everything else runs by itself or waits.
type BoardColumn string

const (
	ColumnNext    BoardColumn = "next"
	ColumnPlan    BoardColumn = "plan"
	ColumnBuild   BoardColumn = "build"
	ColumnApprove BoardColumn = "approve"
	ColumnMerge   BoardColumn = "merge"
	ColumnDone    BoardColumn = "done"
)

type CardState string

const (
	CardNeedsYou CardState = "needs_you"
	CardRunning  CardState = "running"
	CardWaiting  CardState = "waiting"
	CardDone     CardState = "done"
)

// Card decisions name what answers a card: the plan before the build, the approval
// after it, a merge to retry, or a build that needs fixing.
const (
	CardDecisionPlan    = "plan"
	CardDecisionApprove = "approve"
	CardDecisionMerge   = "merge"
	CardDecisionFix     = "fix"
)

type BoardCard struct {
	WorkItemID string      `json:"workItemId"`
	Title      string      `json:"title"`
	Column     BoardColumn `json:"column"`
	State      CardState   `json:"state"`
	Decision   string      `json:"decision,omitempty"`
	// Status is a short label, and Line one plain sentence about what happens next.
	Status string `json:"status"`
	Line   string `json:"line"`
	// BlockedBy lists the work items this one depends on, done or not, for the work graph.
	BlockedBy []string `json:"blockedBy,omitempty"`
	// Facts are a few short labels for the card, such as "3 questions" or "Checks passed".
	Facts []CardFact `json:"facts,omitempty"`
	// Step is where the item is among its six steps: 0 plan, 1 agree, 2 build, 3 checks,
	// 4 approve, 5 merge, and 6 once it is done.
	Step int `json:"step"`
	// Since is when the current activity started, for an elapsed timer.
	Since    *time.Time `json:"since,omitempty"`
	Progress string     `json:"progress,omitempty"`
	CostUSD  float64    `json:"costUsd"`

	JobID         JobID `json:"jobId,omitempty"`
	JobVersion    int64 `json:"jobVersion,omitempty"`
	RefinementID  ID    `json:"refinementId,omitempty"`
	ExpectationID ID    `json:"expectationId,omitempty"`
	Questions     int   `json:"questions,omitempty"`
	Flagged       int   `json:"flagged,omitempty"`
	// Unchecked counts the requirements no locked test checks, so a person must.
	Unchecked int       `json:"unchecked,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CardFact is one short label on a card. Tone is "warn", "ok", "bad", or empty.
type CardFact struct {
	Text string `json:"text"`
	Tone string `json:"tone,omitempty"`
}

type ProjectBoard struct {
	Project       Project     `json:"project"`
	Cards         []BoardCard `json:"cards"`
	NeedsYou      int         `json:"needsYou"`
	Running       int         `json:"running"`
	MergedToday   int         `json:"mergedToday"`
	TrustedMerges int         `json:"trustedMerges"`
	// Pilot reports whether the autopilot runs on this installation, and CanRefine and
	// CanTest whether it can sharpen requirements and write locked tests for this project.
	Pilot       bool      `json:"pilot"`
	CanRefine   bool      `json:"canRefine"`
	CanTest     bool      `json:"canTest"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// boardInputs is everything one card is derived from.
type boardInputs struct {
	work         orientation.WorkItem
	job          *Job
	refinement   *Refinement
	expectation  *Expectation
	jobTests     *Expectation
	signals      *CriterionSignals
	consent      *planConsent
	policy       AskPolicy
	pilot        bool
	blockers     []string
	cost         float64
	canRefine    bool
	canTest      bool
	requireTests bool
	// ticketFile is the digest of the ticket file as it is now, the identity
	// refinements record.
	ticketFile string
}

func (s *Service) Board(ctx context.Context, actor Principal, projectID ProjectID) (ProjectBoard, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return ProjectBoard{}, err
	}
	project, err := readProject(ctx, s.state.db, projectID)
	if err != nil {
		return ProjectBoard{}, err
	}
	snapshot, err := s.boardSnapshot(ctx, project)
	if err != nil {
		return ProjectBoard{}, err
	}
	now := s.state.now().UTC()
	board := ProjectBoard{Project: project, Cards: []BoardCard{}, Pilot: s.state.pilot, CanRefine: len(s.state.refiner) > 0, CanTest: s.canWriteExpectations(project), GeneratedAt: now}
	if board.TrustedMerges, err = trustedMerges(ctx, s.state.db, projectID); err != nil {
		return ProjectBoard{}, err
	}
	for _, inputs := range snapshot {
		card, ok := deriveCard(inputs)
		if !ok {
			continue
		}
		if note, ok := s.pilotNoteFor(string(projectID) + "/" + card.WorkItemID); ok && note.card == cardKey(card) {
			card.State, card.Decision = CardNeedsYou, stuckDecision(card)
			card.Line = "The autopilot could not continue: " + firstSentence(note.message, "open it to see why.")
		}
		board.Cards = append(board.Cards, card)
		switch card.State {
		case CardNeedsYou:
			board.NeedsYou++
		case CardRunning:
			board.Running++
		case CardDone:
			if card.Since != nil && card.Since.Local().Format(time.DateOnly) == now.Local().Format(time.DateOnly) {
				board.MergedToday++
			}
		}
	}
	return board, nil
}

// boardSnapshot gathers each work item's latest job, refinement, expectations, cached
// criterion signals, and consent.
func (s *Service) boardSnapshot(ctx context.Context, project Project) ([]boardInputs, error) {
	result, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return nil, fault(CodeInternal, "read project orientation", err)
	}
	jobs, err := latestJobs(ctx, s.state.db, project.ID)
	if err != nil {
		return nil, err
	}
	refinements, refinementCost, err := latestRecords[Refinement](ctx, s.state.db, `SELECT data_json FROM refinements WHERE project_id=? ORDER BY created_at, id`, project.ID, func(r Refinement) (string, float64) { return r.WorkItemID, runCost(r.Run) })
	if err != nil {
		return nil, err
	}
	expectations, expectationCost, err := latestRecords[Expectation](ctx, s.state.db, `SELECT data_json FROM expectations WHERE project_id=? ORDER BY created_at, id`, project.ID, func(e Expectation) (string, float64) { return e.WorkItemID, runCost(e.Run) })
	if err != nil {
		return nil, err
	}
	consents, err := loadConsents(ctx, s.state.db, project.ID)
	if err != nil {
		return nil, err
	}
	// Scores are cached by criterion text, so only new or changed criteria reach the
	// decision model; the timeout keeps an unreachable model from stalling the board.
	scoring, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	titles := map[string]string{}
	for _, work := range result.WorkItems {
		if work.ID != nil && work.Title != nil {
			titles[*work.ID] = *work.Title
		}
	}
	snapshot := []boardInputs{}
	for _, work := range result.WorkItems {
		if work.ID == nil {
			continue
		}
		id := *work.ID
		inputs := boardInputs{work: work, policy: project.Policy.Ask, pilot: s.state.pilot, cost: refinementCost[id] + expectationCost[id], canRefine: len(s.state.refiner) > 0, canTest: s.canWriteExpectations(project), requireTests: project.Policy.RequireExpectations}
		if job := jobs[id]; job != nil {
			inputs.job = job
			inputs.cost += implementationCost(*job)
			if job.Admission.ExpectationsID != "" {
				if tests, err := loadExpectation(ctx, s.state.db, job.Admission.ExpectationsID); err == nil {
					inputs.jobTests = &tests
				}
			}
		}
		if refinement, ok := refinements[id]; ok {
			inputs.refinement = &refinement
		}
		if expectation, ok := expectations[id]; ok {
			inputs.expectation = &expectation
		}
		if consent, ok := consents[id]; ok {
			inputs.consent = &consent
		}
		if s.state.decision != nil && (work.Execution == nil || *work.Execution == "unstarted") {
			inputs.signals = s.criterionSignals(scoring, work.Title, work.Source, true)
		}
		inputs.blockers = blockers(work, titles, jobs)
		if raw, err := os.ReadFile(work.Source); err == nil {
			inputs.ticketFile = digestBytes(raw)
		}
		snapshot = append(snapshot, inputs)
	}
	return snapshot, nil
}

// latestJobs loads each work item's most recent job that was not cancelled.
func latestJobs(ctx context.Context, q queryer, projectID ProjectID) (map[string]*Job, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, work_item_id FROM jobs WHERE project_id=? AND status<>? ORDER BY created_at, id`, projectID, JobCancelled)
	if err != nil {
		return nil, fault(CodeInternal, "list jobs for the board", err)
	}
	latest := map[string]JobID{}
	for rows.Next() {
		var id JobID
		var workItemID string
		if err := rows.Scan(&id, &workItemID); err != nil {
			rows.Close()
			return nil, fault(CodeInternal, "read jobs for the board", err)
		}
		latest[workItemID] = id
	}
	if err := rows.Close(); err != nil {
		return nil, fault(CodeInternal, "read jobs for the board", err)
	}
	result := map[string]*Job{}
	for workItemID, id := range latest {
		job, err := loadJob(ctx, q, id)
		if err != nil {
			return nil, err
		}
		result[workItemID] = &job
	}
	return result, nil
}

// latestRecords decodes refinement or expectation records in creation order and keeps
// each work item's last one, summing every run's cost.
func latestRecords[T any](ctx context.Context, q queryer, query string, projectID ProjectID, key func(T) (string, float64)) (map[string]T, map[string]float64, error) {
	rows, err := q.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, nil, fault(CodeInternal, "read records for the board", err)
	}
	defer rows.Close()
	latest, cost := map[string]T{}, map[string]float64{}
	for rows.Next() {
		var encoded string
		var record T
		if err := rows.Scan(&encoded); err != nil {
			return nil, nil, fault(CodeInternal, "read records for the board", err)
		}
		if json.Unmarshal([]byte(encoded), &record) != nil {
			continue
		}
		workItemID, spent := key(record)
		latest[workItemID] = record
		cost[workItemID] += spent
	}
	return latest, cost, rows.Err()
}

func runCost(run *RefinerRun) float64 {
	if run == nil {
		return 0
	}
	return run.CostUSD
}

func implementationCost(job Job) float64 {
	total := 0.0
	for _, attempt := range job.Attempts {
		for _, artifact := range attempt.Artifacts {
			var result struct {
				CostUSD float64 `json:"costUsd"`
			}
			if artifact.Name == "agent-result.json" && json.Unmarshal([]byte(artifact.Content), &result) == nil {
				total += result.CostUSD
			}
		}
	}
	return total
}

// blockers names the prerequisites that are not done: not completed in the records, or
// accepted by a factory job that is not merged yet.
func blockers(work orientation.WorkItem, titles map[string]string, jobs map[string]*Job) []string {
	result := []string{}
	for _, edge := range work.Dependencies {
		if edge.To.ID == nil {
			continue
		}
		id := *edge.To.ID
		name := titles[id]
		if name == "" {
			name = id
		}
		job := jobs[id]
		switch {
		case edge.Status != "pass":
			result = append(result, name)
		case job != nil && job.Status == JobAccepted && !job.Merge.Merged():
			result = append(result, name)
		}
	}
	return result
}

// deriveCard places a work item on the board. It is a pure function of the item's
// records, so the board, the list view, and the autopilot agree.
func deriveCard(in boardInputs) (BoardCard, bool) {
	work := in.work
	if work.Execution != nil && *work.Execution == "cancelled" {
		return BoardCard{}, false
	}
	card := BoardCard{WorkItemID: *work.ID, Title: *work.ID, CostUSD: in.cost}
	for _, edge := range work.Dependencies {
		if edge.To.ID != nil {
			card.BlockedBy = append(card.BlockedBy, *edge.To.ID)
		}
	}
	if work.Title != nil {
		card.Title = *work.Title
	}
	if in.signals != nil {
		card.Flagged = in.signals.FlaggedCount
	}
	if in.job != nil {
		jobCard(&card, in)
		return card, true
	}
	if work.Execution != nil && *work.Execution == "completed" {
		card.Column, card.State, card.Status, card.Line, card.Step = ColumnDone, CardDone, "Done", "Recorded as done in the project.", 6
		return card, true
	}
	if in.expectation != nil && work.CriteriaSHA256 != nil && in.expectation.CriteriaSHA256 == *work.CriteriaSHA256 && expectationsCard(&card, in) {
		return card, true
	}
	if in.refinement != nil && refinementCard(&card, in) {
		return card, true
	}
	readyCard(&card, in)
	return card, true
}

func fact(card *BoardCard, tone, text string) {
	card.Facts = append(card.Facts, CardFact{Text: text, Tone: tone})
}

func needs(card *BoardCard, column BoardColumn, decision, status, line string) {
	card.Column, card.State, card.Decision, card.Status, card.Line = column, CardNeedsYou, decision, status, line
}

func running(card *BoardCard, column BoardColumn, status, line string, since *time.Time) {
	card.Column, card.State, card.Status, card.Line, card.Since = column, CardRunning, status, line, since
}

func waiting(card *BoardCard, column BoardColumn, status, line string) {
	card.Column, card.State, card.Status, card.Line = column, CardWaiting, status, line
}

func plural(count int, one, many string) string {
	if count == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", count, many)
}

// autopilotFor reports whether the autopilot will take the next step for this item: a
// person gave the go-ahead, or the ask policy lets it.
func autopilotFor(in boardInputs, policyAllows bool) bool {
	return in.pilot && (in.consent != nil || (in.policy.Autopilot && policyAllows))
}

func jobCard(card *BoardCard, in boardInputs) {
	job := in.job
	card.JobID, card.JobVersion, card.UpdatedAt, card.Step = job.ID, job.Version, job.UpdatedAt, 2
	switch job.Status {
	case JobAccepted:
		card.Step = 5
		switch {
		case job.Merge.Merged():
			card.Column, card.State, card.Status, card.Step = ColumnDone, CardDone, "Merged", 6
			card.Line, card.Since = "Passed the first time.", job.Merge.MergedAt
			if job.RepairCount > 0 {
				card.Line = "Passed after " + plural(job.RepairCount, "fix.", "fixes.")
			}
		case job.Merge != nil && job.Merge.Status == MergeStatusVerifying:
			at := job.Merge.RequestedAt
			running(card, ColumnMerge, "Merging", "main moves only when every check passes on it.", &at)
			fact(card, "", "Checks on the merge")
		case job.Merge != nil:
			needs(card, ColumnMerge, CardDecisionMerge, "Merge failed", sentence(firstSentence(job.Merge.Failure, "Main was left as it was.")))
			fact(card, "bad", "Main unchanged")
		default:
			needs(card, ColumnMerge, CardDecisionMerge, "Ready to merge", "Approved but not merged. Work that depends on it waits.")
			fact(card, "ok", "Approved")
		}
	case JobAccepting:
		card.Step = 4
		running(card, ColumnApprove, "Recording the approval", "The acceptance record is being written.", &job.UpdatedAt)
	case JobReadyForReview, JobReadyForAcceptance:
		card.Step = 4
		line := "Checks passed. No locked tests, so check every requirement yourself."
		tested := in.jobTests != nil && len(in.jobTests.Criteria) > 0
		if tested {
			card.Unchecked = len(in.jobTests.Uncovered())
			if card.Unchecked == 0 {
				line = "Checks passed, and a locked test checks every requirement."
			} else {
				line = "Checks passed. " + plural(card.Unchecked, "requirement only you can check.", "requirements only you can check.")
			}
		}
		fact(card, "ok", "Checks passed")
		switch {
		case !tested:
			fact(card, "warn", "No locked tests")
		case card.Unchecked > 0:
			fact(card, "warn", plural(card.Unchecked, "for you to check", "for you to check"))
		}
		if approvesByPolicy(in) {
			running(card, ColumnApprove, "Approving for you", line, &job.UpdatedAt)
			return
		}
		needs(card, ColumnApprove, CardDecisionApprove, "Approve the change", line)
	case JobQueued:
		running(card, ColumnBuild, "Waiting for a builder", "It starts when a builder is free.", &job.CreatedAt)
	case JobImplementing:
		attempt := latestAttempt(*job, StageImplementation)
		status := "Building"
		if job.RepairCount > 0 {
			status = "Fixing"
			fact(card, "", fmt.Sprintf("Fix %d of %d", job.RepairCount, job.Admission.RepairLimit))
		}
		running(card, ColumnBuild, status, "An agent is writing the change in its own copy.", attemptStart(attempt, job.UpdatedAt))
		card.Progress = latestProgress(*job)
	case JobVerifying:
		card.Step = 3
		running(card, ColumnBuild, "Running checks", "The checks run in a fresh copy, not the agent's.", attemptStart(latestAttempt(*job, StageVerification), job.UpdatedAt))
	case JobVerificationFailed:
		card.Step = 3
		left := job.RepairCount < job.Admission.RepairLimit
		if left && autopilotFor(in, true) {
			running(card, ColumnBuild, "Fixing", fmt.Sprintf("Fix %d of %d starts by itself.", job.RepairCount+1, job.Admission.RepairLimit), &job.UpdatedAt)
			fact(card, "", fmt.Sprintf("Fix %d of %d", job.RepairCount+1, job.Admission.RepairLimit))
			return
		}
		line := "No fixes are left."
		if left {
			line = fmt.Sprintf("%d of %d fixes used. Ask for a fix or reject it.", job.RepairCount, job.Admission.RepairLimit)
		}
		needs(card, ColumnBuild, CardDecisionFix, "Checks failed", line)
		fact(card, "bad", "Checks failed")
		fact(card, "", fmt.Sprintf("%d of %d fixes used", job.RepairCount, job.Admission.RepairLimit))
	case JobFailed:
		needs(card, ColumnBuild, CardDecisionFix, "Failed", "The build failed. Open it to see why.")
		fact(card, "bad", "Build failed")
	case JobStale:
		needs(card, ColumnBuild, CardDecisionFix, "Requirements changed", "The requirements changed after the build started.")
	case JobImplementationUnknown, JobCancellationUnknown:
		needs(card, ColumnBuild, CardDecisionFix, "Outcome unknown", "The builder stopped reporting. Open it to decide what happened.")
	case JobCancelRequested:
		running(card, ColumnBuild, "Stopping", "The build was asked to stop.", &job.UpdatedAt)
	default:
		running(card, ColumnBuild, strings.ReplaceAll(string(job.Status), "_", " "), "", &job.UpdatedAt)
	}
}

// approvesByPolicy reports whether the autopilot will approve the job for the person
// who saved the ask policy. A go-ahead on the plan never covers the approval.
func approvesByPolicy(in boardInputs) bool {
	return in.pilot && in.policy.SetBy != nil && autoApproves(in)
}

// autoApproves reports whether the ask policy lets the factory approve this job: when
// asked only for untested requirements and every requirement has a locked test, or
// never asked once trust is earned.
func autoApproves(in boardInputs) bool {
	switch in.policy.After() {
	case AskAfterNever:
		return true
	case AskAfterWhenUntested:
		return in.jobTests != nil && len(in.jobTests.Criteria) > 0 && len(in.jobTests.Uncovered()) == 0
	}
	return false
}

func (in boardInputs) policyRequiresTests() bool { return in.requireTests }

// itemClear reports whether the work item needs no sharpening by the decision model.
func itemClear(in boardInputs) bool {
	return in.signals != nil && in.signals.Status == "observed" && in.signals.FlaggedCount == 0
}

func expectationsCard(card *BoardCard, in boardInputs) bool {
	expectation := in.expectation
	card.ExpectationID, card.UpdatedAt, card.Unchecked = expectation.ID, expectation.UpdatedAt, len(expectation.Uncovered())
	tests := countTests(*expectation)
	switch expectation.Status {
	case ExpectationRunning:
		running(card, ColumnPlan, "Writing locked tests", "A separate agent writes tests the builder cannot change.", &expectation.CreatedAt)
	case ExpectationProposed:
		card.Step = 1
		if len(expectation.Concerns) > 0 {
			fact(card, "warn", plural(len(expectation.Concerns), "concern", "concerns"))
		}
		fact(card, "", plural(tests, "test", "tests"))
		if card.Unchecked > 0 {
			fact(card, "warn", plural(card.Unchecked, "untested", "untested"))
		}
		line := plural(tests, "locked test", "locked tests") + " for " + plural(len(expectation.Criteria), "requirement.", "requirements.")
		if len(expectation.Concerns) > 0 {
			line = "The test author found " + plural(len(expectation.Concerns), "problem", "problems") + " in the requirements."
		} else if autopilotFor(in, in.policy.Before() != AskBeforeAlways && (itemClear(in) || in.policy.Before() == AskBeforeNever)) {
			card.Step = 2
			running(card, ColumnPlan, "Starting the build", line, &expectation.UpdatedAt)
			return true
		}
		needs(card, ColumnPlan, CardDecisionPlan, "Read the locked tests", line)
	case ExpectationAccepted:
		card.Step = 1
		fact(card, "ok", "Tests agreed")
		if len(in.blockers) > 0 {
			waiting(card, ColumnPlan, "Waits for "+strings.Join(in.blockers, ", "), "The build starts once that is merged.")
		} else if autopilotFor(in, in.policy.Before() != AskBeforeAlways) {
			running(card, ColumnPlan, "Starting the build", "The locked tests are agreed.", &expectation.UpdatedAt)
		} else {
			needs(card, ColumnPlan, CardDecisionPlan, "Start the build", "The locked tests are agreed.")
		}
	case ExpectationFailed:
		needs(card, ColumnPlan, CardDecisionPlan, "Writing tests failed", sentence(firstSentence(expectation.Error, "Try again or change the requirements.")))
	case ExpectationRejected:
		needs(card, ColumnPlan, CardDecisionPlan, "Tests rejected", "Write them again, or change the requirements.")
	default:
		return false
	}
	return true
}

func countTests(expectation Expectation) int {
	seen := map[string]bool{}
	for _, entry := range expectation.Coverage {
		for _, test := range entry.Tests {
			seen[test] = true
		}
	}
	if len(seen) == 0 {
		return len(expectation.Files)
	}
	return len(seen)
}

func refinementCard(card *BoardCard, in boardInputs) bool {
	refinement := in.refinement
	ticket := in.ticketFile
	card.RefinementID, card.UpdatedAt = refinement.ID, refinement.UpdatedAt
	switch {
	case refinement.Status == RefinementRunning && refinement.TicketSHA256 == ticket:
		running(card, ColumnPlan, "Sharpening requirements", "An agent is rewriting the requirements so a test can check each one.", &refinement.CreatedAt)
	case refinement.Status == RefinementProposed && refinement.TicketSHA256 == ticket:
		questions := 0
		proposed := 0
		if refinement.Proposal != nil {
			questions, proposed = len(refinement.Proposal.Questions), len(refinement.Proposal.Criteria)
		}
		card.Questions = questions
		flagged := refinement.After != nil && refinement.After.FlaggedCount > 0
		if questions > 0 {
			fact(card, "warn", plural(questions, "question", "questions"))
		}
		if flagged {
			fact(card, "warn", plural(refinement.After.FlaggedCount, "still vague", "still vague"))
		}
		fact(card, "", plural(proposed, "requirement", "requirements"))
		line := "Your " + plural(len(refinement.Criteria), "requirement was", "requirements were") + " too vague to test. The agent rewrote them as " + fmt.Sprint(proposed) + "."
		if flagged && questions == 0 {
			line = "Rewritten as " + fmt.Sprint(proposed) + " requirements, but " + plural(refinement.After.FlaggedCount, "still looks", "still look") + " vague, so it stopped for you."
		}
		if questions == 0 && !flagged && autopilotFor(in, in.policy.Before() == AskBeforeNever) {
			running(card, ColumnPlan, "Agreeing for you", line, &refinement.UpdatedAt)
			return true
		}
		status := "Read the sharpened requirements"
		if questions > 0 {
			status = "Answer " + plural(questions, "question", "questions")
		}
		needs(card, ColumnPlan, CardDecisionPlan, status, line)
	case refinement.Status == RefinementFailed && refinement.TicketSHA256 == ticket:
		needs(card, ColumnPlan, CardDecisionPlan, "Sharpening failed", sentence(firstSentence(refinement.Error, "Try again or edit the requirements.")))
	case refinement.Status == RefinementAccepted && refinement.UpdatedTicketSHA256 == ticket:
		card.Step = 1
		fact(card, "ok", "Requirements agreed")
		next := in.canTest || (!in.policyRequiresTests() && (in.consent != nil || in.policy.Before() != AskBeforeAlways))
		if next && autopilotFor(in, in.policy.Autopilot) {
			status := "Starting the build"
			if in.canTest {
				status = "Writing locked tests next"
			}
			running(card, ColumnPlan, status, "The sharpened requirements are agreed.", &refinement.UpdatedAt)
		} else {
			needs(card, ColumnPlan, CardDecisionPlan, "Requirements agreed", "Start the locked tests or the build.")
		}
	default:
		return false
	}
	return true
}

func readyCard(card *BoardCard, in boardInputs) {
	work := in.work
	if len(in.blockers) > 0 {
		waiting(card, ColumnNext, "Waits for "+strings.Join(in.blockers, ", "), "It starts once that is done.")
		fact(card, "", "After "+strings.Join(in.blockers, ", "))
		return
	}
	if !work.Eligible {
		reason := "It is not marked ready for agents."
		for _, finding := range work.ExclusionReasons {
			if finding.Code != "commitment_unknown" && finding.Code != "not_committed" {
				reason = firstSentence(finding.Message, reason)
				break
			}
		}
		waiting(card, ColumnNext, "Not ready", reason)
		return
	}
	if in.signals != nil && len(in.signals.Criteria) > 0 {
		fact(card, "", plural(len(in.signals.Criteria), "requirement", "requirements"))
	}
	switch {
	case card.Flagged > 0:
		fact(card, "warn", plural(card.Flagged, "looks vague", "look vague"))
	case itemClear(in):
		fact(card, "ok", "All checkable")
	}
	vague := plural(card.Flagged, "requirement looks", "requirements look") + " vague"
	if startsByItself(in, card.Flagged) {
		line := "Ready."
		if card.Flagged > 0 {
			line = vague + ", so it will be sharpened first."
		}
		waiting(card, ColumnNext, "Starts by itself", line)
		return
	}
	line := "Agree what done means to start it."
	if card.Flagged > 0 {
		line = vague + ". Sharpen them before it starts."
	}
	waiting(card, ColumnNext, "Ready", line)
	card.Decision = CardDecisionPlan
}

// stuckDecision names the decision that takes over when the autopilot cannot.
func stuckDecision(card BoardCard) string {
	switch card.Column {
	case ColumnApprove:
		return CardDecisionApprove
	case ColumnMerge:
		return CardDecisionMerge
	case ColumnBuild:
		return CardDecisionFix
	}
	return CardDecisionPlan
}

// startsByItself reports whether the autopilot will start a ready work item: it can
// sharpen vague requirements, and then write locked tests or skip the plan decision.
func startsByItself(in boardInputs, flagged int) bool {
	if !in.pilot || !in.policy.Autopilot {
		return false
	}
	if flagged > 0 && !in.canRefine && in.policy.Before() != AskBeforeNever {
		return false
	}
	return in.canTest || (flagged > 0 && in.canRefine) || (in.policy.Before() != AskBeforeAlways && !in.policyRequiresTests())
}

func latestAttempt(job Job, stage Stage) *Attempt {
	var latest *Attempt
	for index := range job.Attempts {
		attempt := &job.Attempts[index]
		if attempt.Stage == stage && attempt.Purpose != AttemptPurposeMerge && (latest == nil || attempt.Ordinal >= latest.Ordinal) {
			latest = attempt
		}
	}
	return latest
}

func attemptStart(attempt *Attempt, fallback time.Time) *time.Time {
	if attempt != nil && attempt.StartedAt != nil {
		return attempt.StartedAt
	}
	return &fallback
}

func latestProgress(job Job) string {
	for index := len(job.Events) - 1; index >= 0; index-- {
		if job.Events[index].Kind != "worker_progress" {
			continue
		}
		var data struct {
			Progress string `json:"progress"`
		}
		if json.Unmarshal(job.Events[index].Data, &data) == nil {
			return data.Progress
		}
	}
	return ""
}

func firstSentence(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	if index := strings.IndexAny(text, ".\n"); index > 0 && index < 200 {
		return strings.TrimSpace(text[:index+1])
	}
	return truncateRunes(text, 200)
}

// sentence makes service text read as a sentence of its own: capitalized, with a stop.
func sentence(text string) string {
	first, size := utf8.DecodeRuneInString(text)
	if size == 0 {
		return text
	}
	text = string(unicode.ToUpper(first)) + text[size:]
	for _, stop := range []string{".", "!", "?", "…"} {
		if strings.HasSuffix(text, stop) {
			return text
		}
	}
	return text + "."
}
