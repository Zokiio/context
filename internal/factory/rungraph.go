package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Run step statuses, as the run graph colours them.
const (
	RunDone     = "done"
	RunRunning  = "running"
	RunNeedsYou = "needs_you"
	RunNext     = "next"
	RunIdle     = "idle"
	RunSkipped  = "skipped"
	RunFailed   = "failed"
)

// RunStep is one step of a work item's run: who or what does it, how it went, and
// what it cost.
type RunStep struct {
	Key string `json:"key"`
	// Kind is agent, you, check or automatic.
	Kind    string  `json:"kind"`
	Title   string  `json:"title"`
	Status  string  `json:"status"`
	Summary string  `json:"summary,omitempty"`
	Model   string  `json:"model,omitempty"`
	Turns   int     `json:"turns,omitempty"`
	Seconds float64 `json:"seconds,omitempty"`
	CostUSD float64 `json:"costUsd,omitempty"`
	// By names the person who decided, or the autopilot acting for them.
	By string `json:"by,omitempty"`
	// Notes are longer lines for the step's detail panel.
	Notes []string `json:"notes,omitempty"`
}

// ItemRun is a work item's run as a fixed set of steps, derived from its records.
type ItemRun struct {
	ProjectID   ProjectID `json:"projectId"`
	WorkItemID  string    `json:"workItemId"`
	Title       string    `json:"title"`
	JobID       JobID     `json:"jobId,omitempty"`
	CostUSD     float64   `json:"costUsd"`
	Repairs     int       `json:"repairs"`
	RepairLimit int       `json:"repairLimit"`
	Steps       []RunStep `json:"steps"`
}

// WorkItemRun reports each step of a work item's run: sharpen, tests, agree, build,
// checks, approve, merge, and repair.
func (s *Service) WorkItemRun(ctx context.Context, actor Principal, projectID ProjectID, workItemID string) (ItemRun, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return ItemRun{}, err
	}
	project, err := readProject(ctx, s.state.db, projectID)
	if err != nil {
		return ItemRun{}, err
	}
	snapshot, err := s.boardSnapshot(ctx, project)
	if err != nil {
		return ItemRun{}, err
	}
	var in *boardInputs
	for index := range snapshot {
		if snapshot[index].work.ID != nil && *snapshot[index].work.ID == workItemID {
			in = &snapshot[index]
		}
	}
	if in == nil {
		return ItemRun{}, fault(CodeNotFound, "work item not found", nil)
	}
	// The board omits cancelled jobs so an explicitly requeued task can be
	// dispatched again. A run graph describes retained execution instead: stopping
	// must not make its completed agreement, attempts, or checks disappear.
	var latestID JobID
	err = s.state.db.QueryRowContext(ctx, `SELECT id FROM jobs WHERE project_id=? AND work_item_id=? ORDER BY created_at DESC, id DESC LIMIT 1`, projectID, workItemID).Scan(&latestID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ItemRun{}, fault(CodeInternal, "read latest work item run", err)
	}
	if err == nil && (in.job == nil || in.job.ID != latestID) {
		job, err := loadJob(ctx, s.state.db, latestID)
		if err != nil {
			return ItemRun{}, err
		}
		if in.job != nil {
			in.cost -= implementationCost(*in.job)
		}
		in.job, in.jobTests = &job, nil
		in.dispute, err = latestJobDispute(ctx, s.state.db, job.ID)
		if err != nil {
			return ItemRun{}, err
		}
		if in.dispute == nil && job.Admission.DisputeID != "" {
			d, err := loadDispute(ctx, s.state.db, job.Admission.DisputeID)
			if err != nil {
				return ItemRun{}, err
			}
			in.dispute = &d
		}
		in.cost += implementationCost(job)
		if job.Admission.ExpectationsID != "" {
			tests, err := loadExpectation(ctx, s.state.db, job.Admission.ExpectationsID)
			if err != nil {
				return ItemRun{}, err
			}
			in.jobTests = &tests
		}
	}
	card, _ := deriveCard(*in)
	refinements, err := s.Refinements(ctx, actor, projectID, workItemID)
	if err != nil {
		return ItemRun{}, err
	}
	names := map[UserID]string{}
	name := func(id *UserID, delegated string) string {
		if id == nil {
			return ""
		}
		if _, ok := names[*id]; !ok {
			if user, err := readUser(ctx, s.state.db, *id); err == nil {
				names[*id] = user.Username
			}
		}
		if delegated != "" {
			return "the autopilot, for " + names[*id]
		}
		return names[*id]
	}
	run := ItemRun{ProjectID: projectID, WorkItemID: workItemID, Title: card.Title, CostUSD: card.CostUSD}
	job := in.job
	if job != nil {
		run.JobID, run.Repairs, run.RepairLimit = job.ID, job.RepairCount, job.Admission.RepairLimit
	}
	run.Steps = []RunStep{
		sharpenStep(*in, refinements),
		testsStep(*in),
		agreeStep(*in, card, name),
		buildStep(job, s.changeSummary(ctx, actor, job)),
		checksStep(job),
		approveStep(job, card, name),
		mergeStep(job),
		repairStep(job, card),
	}
	if in.dispute != nil {
		run.Steps = append(run.Steps, judgeStep(in.dispute))
	}
	return run, nil
}

func judgeStep(d *Dispute) RunStep {
	step := RunStep{Key: "judge", Kind: "agent", Title: "Judge the dispute", Status: RunDone, Summary: "Decision recorded"}
	if len(d.TestReviews) > 0 || d.Status == "reviewing_tests" {
		step.Title = "Review the test proposal"
	}
	addRun := func(run *RefinerRun) {
		if run != nil {
			step.Model = run.Model
			step.Turns += run.Turns
			step.CostUSD += run.CostUSD
		}
	}
	if d.Decision != nil {
		step.Notes = append(step.Notes, "Original dispute verdict: "+d.Decision.Rationale, d.Decision.Directive)
	}
	addRun(d.Run)
	for _, review := range d.TestReviews {
		step.Notes = append(step.Notes, fmt.Sprintf("Test proposal review %d · revision %s", review.Round, review.Commit))
		if review.Decision != nil {
			step.Notes = append(step.Notes, review.Decision.Rationale, review.Decision.Directive)
		}
		if review.Error != "" {
			step.Notes = append(step.Notes, review.Error)
		}
		addRun(review.Run)
	}
	switch d.Status {
	case "judging":
		step.Status, step.Summary = RunRunning, "Independent judge reviewing"
	case "reviewing_tests":
		step.Status, step.Summary = RunRunning, "Independent judge reviewing the test proposal"
	case "authoring_tests":
		step.Status, step.Summary = RunRunning, "Decision sent to the test author"
		if len(d.TestReviews) > 0 {
			step.Summary = "Tests returned to the author for improvement"
		}
	case "tests_proposed":
		step.Status, step.Summary = RunNeedsYou, "Test proposal ready for judge review"
	case "needs_human":
		step.Status, step.Summary = RunNeedsYou, "Waiting for your decision"
	case "routing":
		step.Status, step.Summary = RunRunning, "Controller arranging the handoff"
	case "resolved":
		if len(d.TestReviews) > 0 && d.NextOwner == "implementer" {
			latest := d.TestReviews[len(d.TestReviews)-1]
			if latest.Decision != nil && latest.Decision.Kind == DisputeAcceptTests {
				step.Summary = "Tests accepted; handed to the builder"
			}
		}
	case "stopped":
		step.Status, step.Summary = RunSkipped, "Dispute stopped"
	}
	step.Notes = append(step.Notes, "Next owner: "+d.NextOwner)
	if d.Error != "" {
		step.Notes = append(step.Notes, d.Error)
	}
	return step
}

func sharpenStep(in boardInputs, refinements []Refinement) RunStep {
	step := RunStep{Key: "sharpen", Kind: "agent", Title: "Sharpen requirements", Status: RunSkipped, Summary: "Not needed"}
	if len(refinements) == 0 {
		if in.signals != nil && in.signals.FlaggedCount > 0 {
			step.Status, step.Summary = RunNext, plural(in.signals.FlaggedCount, "requirement looks vague", "requirements look vague")
		}
		return step
	}
	for _, refinement := range refinements {
		if refinement.Run != nil {
			step.Turns += refinement.Run.Turns
			step.CostUSD += refinement.Run.CostUSD
		}
	}
	latest := refinements[0]
	if latest.Run != nil {
		step.Model = latest.Run.Model
	}
	step.Notes = append(step.Notes, plural(len(refinements), "run", "runs")+".")
	for _, answer := range latest.PreviousAnswers {
		step.Notes = append(step.Notes, "You answered “"+answer.Question+"” with “"+answer.Answer+"”.")
	}
	switch latest.Status {
	case RefinementRunning:
		step.Status, step.Summary = RunRunning, "Rewriting the requirements"
	case RefinementProposed:
		questions := 0
		if latest.Proposal != nil {
			questions = len(latest.Proposal.Questions)
		}
		step.Status, step.Summary = RunNeedsYou, "Read the sharpened requirements"
		if questions > 0 {
			step.Summary = plural(questions, "question to answer", "questions to answer")
		}
	case RefinementAccepted:
		step.Status, step.Summary = RunDone, plural(len(latest.AcceptedCriteria), "requirement", "requirements")
		// The accepted answers repeat the earlier run's; count each question once.
		questions := map[string]bool{}
		for _, answer := range append(append([]RefinementAnswer(nil), latest.Answers...), latest.PreviousAnswers...) {
			questions[answer.Question] = true
		}
		if answered := len(questions); answered > 0 {
			step.Summary += " · " + plural(answered, "question answered", "questions answered")
		}
	case RefinementFailed:
		step.Status, step.Summary = RunFailed, sentence(firstSentence(latest.Error, "Sharpening failed."))
	default:
		step.Status, step.Summary = RunSkipped, "Proposal set aside"
	}
	return step
}

func testsStep(in boardInputs) RunStep {
	step := RunStep{Key: "tests", Kind: "agent", Title: "Write locked tests", Status: RunNext}
	expectation := in.expectation
	if in.job != nil && in.jobTests != nil {
		expectation = in.jobTests
	}
	if expectation == nil {
		if !in.canTest {
			step.Status, step.Summary = RunSkipped, "No test author here"
		}
		if in.job != nil {
			step.Status, step.Summary = RunSkipped, "Built without locked tests"
		}
		return step
	}
	if expectation.Run != nil {
		step.Model, step.Turns, step.CostUSD = expectation.Run.Model, expectation.Run.Turns, expectation.Run.CostUSD
	}
	if expectation.Notes != "" {
		step.Notes = append(step.Notes, expectation.Notes)
	}
	step.Notes = append(step.Notes, expectation.Concerns...)
	tested := len(expectation.Criteria) - len(expectation.Uncovered())
	summary := fmt.Sprintf("%d of %d tested", tested, len(expectation.Criteria))
	if len(expectation.Concerns) > 0 {
		summary += " · " + plural(len(expectation.Concerns), "concern", "concerns")
	}
	switch expectation.Status {
	case ExpectationRunning:
		step.Status, step.Summary = RunRunning, "Writing tests"
	case ExpectationProposed, ExpectationAccepted:
		step.Status, step.Summary = RunDone, summary
	case ExpectationFailed:
		step.Status, step.Summary = RunFailed, sentence(firstSentence(expectation.Error, "Writing tests failed."))
	default:
		step.Status, step.Summary = RunFailed, "Tests rejected"
	}
	return step
}

func agreeStep(in boardInputs, card BoardCard, name func(*UserID, string) string) RunStep {
	step := RunStep{Key: "agree", Kind: "you", Title: "Agree what done means", Status: RunNext}
	switch {
	case in.job != nil:
		step.Status, step.Summary = RunDone, "Agreed"
		step.By = name(&in.job.CoordinatorID, "")
		if tests := in.jobTests; tests != nil && tests.DecidedBy != nil {
			step.By = name(tests.DecidedBy, tests.Delegated)
		}
		if step.By != "" {
			step.Summary = "Agreed by " + step.By
		}
	case card.Decision == CardDecisionPlan && card.State == CardNeedsYou && card.Column == ColumnPlan:
		step.Status, step.Summary = RunNeedsYou, card.Status
	case in.consent != nil:
		step.Summary = "Go-ahead given"
	}
	return step
}

// buildAttempts lists a job's implementation attempts in order; the first is the build
// and the rest are fixes.
func buildAttempts(job *Job) []Attempt {
	attempts := []Attempt{}
	for _, attempt := range job.Attempts {
		if attempt.Stage == StageImplementation && attempt.Purpose != AttemptPurposeMerge {
			attempts = append(attempts, attempt)
		}
	}
	sort.SliceStable(attempts, func(i, j int) bool { return attempts[i].Ordinal < attempts[j].Ordinal })
	return attempts
}

// agentResult reads what an agent reported about its own run, when it reported it.
func agentResult(attempt Attempt) (model string, turns int, cost float64, message string) {
	for _, artifact := range attempt.Artifacts {
		if artifact.Name != "agent-result.json" {
			continue
		}
		var result struct {
			Model   string  `json:"model"`
			Turns   int     `json:"turns"`
			CostUSD float64 `json:"costUsd"`
			Result  string  `json:"result"`
		}
		if json.Unmarshal([]byte(artifact.Content), &result) == nil {
			return result.Model, result.Turns, result.CostUSD, result.Result
		}
	}
	return "", 0, 0, ""
}

func attemptSeconds(attempt Attempt) float64 {
	if attempt.StartedAt == nil || attempt.FinishedAt == nil {
		return 0
	}
	return attempt.FinishedAt.Sub(*attempt.StartedAt).Seconds()
}

func buildStep(job *Job, change string) RunStep {
	step := RunStep{Key: "build", Kind: "agent", Title: "Build the change", Status: RunNext}
	if job == nil {
		return step
	}
	attempts := buildAttempts(job)
	if len(attempts) == 0 {
		step.Status, step.Summary = RunRunning, "Waiting for a builder"
		return step
	}
	first := attempts[0]
	model, turns, cost, message := agentResult(first)
	step.Model, step.Turns, step.CostUSD, step.Seconds = model, turns, cost, attemptSeconds(first)
	if message != "" {
		step.Notes = append(step.Notes, "The builder said: “"+truncateRunes(strings.TrimSpace(message), 600)+"”")
	}
	switch {
	case first.Status == AttemptUnknown:
		step.Status, step.Summary = RunNeedsYou, "Execution outcome unknown"
		if first.CancellationRequested {
			step.Summary = "Stop outcome unknown"
		}
	case first.Status == AttemptCancelled:
		step.Status, step.Summary = RunFailed, "Build stopped before completion"
		if first.StartedAt == nil {
			step.Status, step.Summary = RunSkipped, "Stopped before the build started"
		}
	case first.Outcome == OutcomeDisputed:
		step.Status, step.Summary = RunFailed, "Stopped to dispute the requirements or tests"
	case first.FinishedAt == nil:
		step.Status, step.Summary = RunRunning, "Building"
		if first.CancellationRequested {
			step.Summary = "Stop requested; waiting for the builder"
		}
	case first.CandidateRevision != "":
		step.Status, step.Summary = RunDone, change
		if step.Summary == "" {
			step.Summary = "Candidate ready"
		}
	default:
		step.Status, step.Summary = RunFailed, sentence(firstSentence(first.Summary, "The build failed."))
	}
	return step
}

// changeSummary counts the files a job's candidate changes, for the build step.
func (s *Service) changeSummary(ctx context.Context, actor Principal, job *Job) string {
	if job == nil || job.CandidateRevision == "" {
		return ""
	}
	change, err := s.Change(ctx, actor, job.ID)
	if err != nil {
		return ""
	}
	added, removed := 0, 0
	for _, file := range change.Files {
		added, removed = added+file.Added, removed+file.Removed
	}
	return fmt.Sprintf("%s · +%d −%d", plural(len(change.Files), "file", "files"), added, removed)
}

func checksStep(job *Job) RunStep {
	step := RunStep{Key: "checks", Kind: "check", Title: "Run the checks", Status: RunNext}
	if job == nil {
		return step
	}
	keys := append([]string(nil), job.Admission.RequiredChecks...)
	step.Model = strings.Join(keys, ", ")
	latest := map[string]Attempt{}
	for _, attempt := range job.Attempts {
		if attempt.Stage == StageVerification && attempt.Purpose != AttemptPurposeMerge && attempt.CandidateRevision == job.CandidateRevision {
			if existing, ok := latest[attempt.CheckKey]; !ok || attempt.Ordinal > existing.Ordinal {
				latest[attempt.CheckKey] = attempt
			}
		}
	}
	for _, key := range keys {
		if attempt, ok := latest[key]; ok {
			step.Seconds += attemptSeconds(attempt)
			status := string(attempt.CheckStatus)
			if status == "" {
				status = string(attempt.Status)
			}
			step.Notes = append(step.Notes, key+": "+status)
		}
	}
	if job.Status == JobCancelRequested || job.Status == JobCancellationUnknown || job.Status == JobCancelled {
		step.Status, step.Summary = stoppedChecks(job, keys, latest)
		return step
	}
	if job.Status == JobDisputed || job.Status == JobSuperseded {
		step.Status, step.Summary = RunIdle, "Did not run"
		passed := 0
		for _, attempt := range latest {
			if attempt.CheckStatus == CheckFailed || attempt.CheckStatus == CheckUnavailable || attempt.CheckStatus == CheckSkipped {
				step.Status, step.Summary = RunFailed, "A check failed"
				return step
			}
			if attempt.CheckStatus == CheckPassed {
				passed++
			}
		}
		if len(keys) > 0 && passed == len(keys) {
			step.Status, step.Summary = RunDone, "Passed in a clean copy"
		} else if len(latest) > 0 {
			step.Summary = "Checks incomplete before the dispute"
		}
		return step
	}
	switch job.Status {
	case JobQueued, JobImplementing:
		if job.RepairCount > 0 {
			step.Status, step.Summary = RunFailed, "Failed; a fix is running"
		}
	case JobVerifying:
		step.Status, step.Summary = RunRunning, "In a clean copy"
	case JobVerificationFailed:
		step.Status, step.Summary = RunFailed, "A check failed"
	case JobFailed, JobStale, JobImplementationUnknown, JobCancellationUnknown, JobCancelRequested, JobCancelled:
		step.Status, step.Summary = RunIdle, "Did not run"
	default:
		step.Status, step.Summary = RunDone, "Passed in a clean copy"
	}
	return step
}

func stoppedChecks(job *Job, keys []string, latest map[string]Attempt) (string, string) {
	passed := 0
	for _, key := range keys {
		attempt, ok := latest[key]
		if !ok {
			continue
		}
		if attempt.CheckStatus == CheckPassed {
			passed++
			continue
		}
		if attempt.Status == AttemptUnknown {
			return RunNeedsYou, "Check stop outcome unknown"
		}
		if attempt.Status != AttemptCancelled && attempt.FinishedAt == nil && job.Status != JobCancelled {
			return RunRunning, "Stop requested; waiting for checks"
		}
	}
	if passed == len(keys) && len(keys) > 0 {
		return RunDone, "Passed in a clean copy"
	}
	if len(latest) > 0 {
		return RunFailed, "Stopped before all checks passed"
	}
	return RunSkipped, "Stopped before the checks"
}

func approveStep(job *Job, card BoardCard, name func(*UserID, string) string) RunStep {
	step := RunStep{Key: "approve", Kind: "you", Title: "Approve the change", Status: RunNext}
	if job == nil {
		return step
	}
	switch job.Status {
	case JobCancelRequested, JobCancellationUnknown, JobCancelled:
		step.Status, step.Summary = RunSkipped, "Stopped before approval"
	case JobReadyForReview, JobReadyForAcceptance:
		step.Status, step.Summary = RunNeedsYou, "Checks passed"
		if card.State == CardRunning {
			step.Status, step.Summary = RunRunning, "Approving for you"
		}
		if card.Unchecked > 0 {
			step.Summary = plural(card.Unchecked, "to check by hand", "to check by hand")
		}
	case JobAccepting:
		step.Status, step.Summary = RunRunning, "Recording the approval"
	case JobAccepted:
		step.Status, step.Summary = RunDone, "Approved"
		for index := len(job.Reviews) - 1; index >= 0; index-- {
			if review := job.Reviews[index]; review.Decision == ReviewApproved {
				step.By = name(&review.ReviewerID, delegatedReview(job, review.ID))
				step.Summary = "Approved by " + step.By
				break
			}
		}
	}
	return step
}

// delegatedReview reports the autopilot's reason when it recorded a review.
func delegatedReview(job *Job, reviewID ID) string {
	for _, event := range job.Events {
		if event.Kind != "review_recorded" {
			continue
		}
		var data struct {
			ReviewID  ID     `json:"reviewId"`
			Delegated string `json:"delegated"`
		}
		if json.Unmarshal(event.Data, &data) == nil && data.ReviewID == reviewID {
			return data.Delegated
		}
	}
	return ""
}

func mergeStep(job *Job) RunStep {
	step := RunStep{Key: "merge", Kind: "automatic", Title: "Merge into main", Status: RunNext, Summary: "Checks rerun on the merge"}
	if job != nil && (job.Status == JobCancelRequested || job.Status == JobCancellationUnknown || job.Status == JobCancelled) {
		step.Status, step.Summary = RunSkipped, "Stopped before merging"
		return step
	}
	if job == nil || job.Status != JobAccepted {
		return step
	}
	if job.Merge == nil {
		step.Status, step.Summary = RunNeedsYou, "Ready to merge"
		return step
	}
	step.Title = "Merge into " + job.Merge.Branch
	switch {
	case job.Merge.Merged():
		step.Status, step.Summary = RunDone, job.Merge.Branch+" moved"
	case job.Merge.Status == MergeStatusVerifying:
		step.Status, step.Summary = RunRunning, "Checks on the merge"
	default:
		step.Status, step.Summary = RunFailed, sentence(firstSentence(job.Merge.Failure, "The merge failed."))
	}
	return step
}

func repairStep(job *Job, card BoardCard) RunStep {
	step := RunStep{Key: "repair", Kind: "agent", Title: "Fix the change", Status: RunIdle, Summary: "Not needed"}
	if job == nil {
		return step
	}
	for _, attempt := range buildAttempts(job)[min(1, len(buildAttempts(job))):] {
		model, turns, cost, _ := agentResult(attempt)
		step.Model, step.Turns, step.CostUSD, step.Seconds = model, step.Turns+turns, step.CostUSD+cost, step.Seconds+attemptSeconds(attempt)
	}
	switch {
	case job.RepairCount > 0 && (job.Status == JobCancelRequested || job.Status == JobCancellationUnknown || job.Status == JobCancelled):
		attempts := buildAttempts(job)
		last := attempts[len(attempts)-1]
		switch {
		case last.CandidateRevision != "":
			step.Status, step.Summary = RunDone, plural(job.RepairCount, "fix", "fixes")
		case last.Status == AttemptUnknown:
			step.Status, step.Summary = RunNeedsYou, "Fix stop outcome unknown"
		case job.Status != JobCancelled && last.FinishedAt == nil:
			step.Status, step.Summary = RunRunning, "Stop requested; waiting for the fix"
		default:
			step.Status, step.Summary = RunFailed, "Fix stopped before completion"
		}
	case job.Status == JobImplementing && job.RepairCount > 0:
		step.Status, step.Summary = RunRunning, fmt.Sprintf("Fix %d of %d", job.RepairCount, job.Admission.RepairLimit)
	case job.Status == JobVerificationFailed && card.State == CardNeedsYou:
		step.Status, step.Summary = RunNeedsYou, fmt.Sprintf("%d of %d fixes used", job.RepairCount, job.Admission.RepairLimit)
	case job.RepairCount > 0:
		step.Status, step.Summary = RunDone, plural(job.RepairCount, "fix", "fixes")
	}
	return step
}
