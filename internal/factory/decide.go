package factory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// A person makes two decisions per change: agreeing what done means before the build,
// and approving the change after it. Each decision can span several records, such as a
// review, an acceptance, and a merge. A decision note travels in the context so every
// record one decision produces carries the same decision identity, and records the
// autopilot produces on someone's behalf say so.
type decisionNote struct {
	ID        string
	Delegated string
}

type decisionKey struct{}

func withDecision(ctx context.Context, note decisionNote) context.Context {
	return context.WithValue(ctx, decisionKey{}, note)
}

func decisionFrom(ctx context.Context) (decisionNote, bool) {
	note, ok := ctx.Value(decisionKey{}).(decisionNote)
	return note, ok
}

// decisionContext gives a request its stable decision identity, keeping any delegation
// the caller already recorded.
func decisionContext(ctx context.Context, actor UserID, operation, requestID string) context.Context {
	note, _ := decisionFrom(ctx)
	if note.ID == "" {
		digest := sha256.Sum256([]byte(string(actor) + "\x00" + operation + "\x00" + requestID))
		note.ID = "dec_" + hex.EncodeToString(digest[:12])
	}
	return withDecision(ctx, note)
}

// annotate adds the context's decision note to an event's data.
func annotate(ctx context.Context, data any) any {
	note, ok := decisionFrom(ctx)
	if !ok || (note.ID == "" && note.Delegated == "") {
		return data
	}
	fields := map[string]any{}
	if encoded, err := json.Marshal(data); err == nil {
		_ = json.Unmarshal(encoded, &fields)
	}
	if fields == nil {
		fields = map[string]any{}
	}
	if note.ID != "" {
		fields["decision"] = note.ID
	}
	if note.Delegated != "" {
		fields["delegated"] = note.Delegated
	}
	return fields
}

// stampDecision records the context's decision note on a refinement or expectation
// decision.
func stampDecision(ctx context.Context, id, delegated *string) {
	if note, ok := decisionFrom(ctx); ok {
		*id, *delegated = note.ID, note.Delegated
	}
}

// Findings are what a reviewer caught that the checks missed.
const (
	FindingWrongRequirement = "wrong_requirement"
	FindingMissingTest      = "missing_test"
	FindingBug              = "bug"
	FindingHardToRead       = "hard_to_read"
	FindingOutOfScope       = "out_of_scope"
)

var findingKinds = []string{FindingWrongRequirement, FindingMissingTest, FindingBug, FindingHardToRead, FindingOutOfScope}

func normalizeFindings(values []string) ([]string, error) {
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !slices.Contains(findingKinds, value) {
			return nil, fault(CodeInvalid, "finding must be one of "+strings.Join(findingKinds, ", "), nil)
		}
		if !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result, nil
}

// ApproveRequest records the decision after the build in one step: the reviewer
// approves the exact candidate, accepts it, and merges it unless Merge is false.
type ApproveRequest struct {
	RequestID              string   `json:"requestId"`
	JobID                  JobID    `json:"jobId"`
	IfVersion              int64    `json:"ifVersion"`
	CandidateRevision      string   `json:"candidateRevision"`
	RequirementFingerprint string   `json:"requirementFingerprint"`
	Summary                string   `json:"summary,omitempty"`
	Findings               []string `json:"findings,omitempty"`
	Merge                  bool     `json:"merge"`
	TargetBranch           string   `json:"targetBranch,omitempty"`
}

const defaultApprovalSummary = "Approved: every required check passed, and the reviewer checked what the tests cannot."

// Approve reviews, accepts, and optionally merges a job as one decision. Each step keeps
// its own rules: an independent human reviewer, a valid acceptance, and checks on the
// merge commit. A retry with the same request continues from the step that is left.
func (s *Service) Approve(ctx context.Context, actor Principal, request ApproveRequest) (Job, error) {
	if request.JobID == "" || request.IfVersion <= 0 || request.CandidateRevision == "" || request.RequirementFingerprint == "" {
		return Job{}, fault(CodeInvalid, "jobId, ifVersion, candidateRevision, and requirementFingerprint are required", nil)
	}
	if err := requireRequestID(request.RequestID); err != nil {
		return Job{}, err
	}
	if _, err := normalizeFindings(request.Findings); err != nil {
		return Job{}, err
	}
	actor, err := requirePrincipal(ctx, s.state.db, actor)
	if err != nil {
		return Job{}, err
	}
	job, err := s.GetJob(ctx, actor, request.JobID)
	if err != nil {
		return Job{}, err
	}
	capabilities, err := s.Capabilities(ctx, actor, job.ProjectID)
	if err != nil {
		return Job{}, err
	}
	needed := []Capability{CapabilityAccept}
	if job.Status == JobReadyForReview {
		needed = append(needed, CapabilityReview)
	}
	if request.Merge {
		needed = append(needed, CapabilityMerge)
	}
	for _, capability := range needed {
		if !actor.InstallationAdmin && !hasCapability(capabilities, capability) {
			return Job{}, fault(CodeForbidden, "approving needs the "+string(capability)+" capability", nil)
		}
	}
	ctx = decisionContext(ctx, actor.UserID, "approve", request.RequestID)
	note, _ := decisionFrom(ctx)
	if job.CandidateRevision != request.CandidateRevision || job.Admission.TicketSHA256 != request.RequirementFingerprint {
		return Job{}, fault(CodeStale, "approval does not match the current candidate and requirements", nil)
	}
	// A retry finds the job further along. Only a fresh approval must match the version.
	if job.Version != request.IfVersion && !decisionRecorded(job, note.ID) {
		return Job{}, detailFault(CodeConflict, "job version changed", map[string]any{"currentVersion": job.Version}, nil)
	}
	summary := strings.TrimSpace(request.Summary)
	if summary == "" {
		summary = defaultApprovalSummary
	}
	if job.Status == JobReadyForReview {
		if job, err = s.Review(ctx, actor, ReviewRequest{RequestID: request.RequestID + "/review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Decision: ReviewApproved, Summary: summary, Findings: request.Findings}); err != nil {
			return Job{}, err
		}
	}
	if job.Status == JobAccepting {
		return Job{}, fault(CodeUncertain, "the acceptance is still being published; open the job to reconcile it", nil)
	}
	if job.Status == JobReadyForAcceptance {
		if job, err = s.Accept(ctx, actor, AcceptRequest{RequestID: request.RequestID + "/accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Summary: summary}); err != nil {
			return Job{}, err
		}
	}
	if job.Status != JobAccepted {
		return Job{}, detailFault(CodeNotReady, "the job is not ready to approve", map[string]any{"status": job.Status}, nil)
	}
	if request.Merge && (job.Merge == nil || job.Merge.Status == MergeStatusFailed) {
		attempt := fmt.Sprintf("%s/merge/%d", request.RequestID, mergeFailures(job))
		if job, err = s.Merge(ctx, actor, MergeRequest{RequestID: attempt, JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, TargetBranch: request.TargetBranch}); err != nil {
			return Job{}, err
		}
	}
	return job, nil
}

// decisionRecorded reports whether the job already has an event from this decision.
func decisionRecorded(job Job, decision string) bool {
	for _, event := range job.Events {
		var data struct {
			Decision string `json:"decision"`
		}
		if json.Unmarshal(event.Data, &data) == nil && data.Decision == decision {
			return true
		}
	}
	return false
}

func mergeFailures(job Job) int {
	count := 0
	for _, event := range job.Events {
		if event.Kind == "merge_failed" {
			count++
		}
	}
	return count
}

// Ask policies say when the factory stops and waits for a person.
type AskBeforeBuild string

const (
	AskBeforeAlways      AskBeforeBuild = "always"
	AskBeforeWhenUnclear AskBeforeBuild = "when_unclear"
	AskBeforeNever       AskBeforeBuild = "never"
)

type AskAfterBuild string

const (
	AskAfterAlways       AskAfterBuild = "always"
	AskAfterWhenUntested AskAfterBuild = "when_untested"
	AskAfterNever        AskAfterBuild = "never"
)

// TrustedMergesForNever is how many merges without a reported problem a project needs
// before the factory may merge without asking.
const TrustedMergesForNever = 10

type AskPolicy struct {
	BeforeBuild AskBeforeBuild `json:"beforeBuild,omitempty"`
	AfterBuild  AskAfterBuild  `json:"afterBuild,omitempty"`
	// Autopilot starts ready work and does the steps that need no decision. It acts as
	// SetBy, the person who saved the policy.
	Autopilot bool       `json:"autopilot,omitempty"`
	SetBy     *UserID    `json:"setBy,omitempty"`
	SetAt     *time.Time `json:"setAt,omitempty"`
}

func (p AskPolicy) Before() AskBeforeBuild {
	if p.BeforeBuild == "" {
		return AskBeforeAlways
	}
	return p.BeforeBuild
}

func (p AskPolicy) After() AskAfterBuild {
	if p.AfterBuild == "" {
		return AskAfterAlways
	}
	return p.AfterBuild
}

type SetAskPolicyRequest struct {
	RequestID   string         `json:"requestId"`
	ProjectID   ProjectID      `json:"projectId"`
	BeforeBuild AskBeforeBuild `json:"beforeBuild"`
	AfterBuild  AskAfterBuild  `json:"afterBuild"`
	Autopilot   bool           `json:"autopilot"`
}

// SetAskPolicy saves when the factory asks. The person saving it delegates the
// decisions it skips, so they must hold the capabilities those decisions need.
func (s *Service) SetAskPolicy(ctx context.Context, actor Principal, request SetAskPolicyRequest) (Project, error) {
	if !slices.Contains([]AskBeforeBuild{AskBeforeAlways, AskBeforeWhenUnclear, AskBeforeNever}, request.BeforeBuild) {
		return Project{}, fault(CodeInvalid, "beforeBuild must be always, when_unclear, or never", nil)
	}
	if !slices.Contains([]AskAfterBuild{AskAfterAlways, AskAfterWhenUntested, AskAfterNever}, request.AfterBuild) {
		return Project{}, fault(CodeInvalid, "afterBuild must be always, when_untested, or never", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, fault(CodeInternal, "start ask policy", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, request.ProjectID, CapabilityProjectAdmin)
	if err != nil {
		return Project{}, err
	}
	if current.Kind != UserKindHuman {
		return Project{}, fault(CodeForbidden, "only a person can decide when the factory asks", nil)
	}
	saved, ok, hash, err := receipt[Project](ctx, tx, current.UserID, "set_ask_policy", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	needed := []Capability{}
	if request.Autopilot || request.BeforeBuild != AskBeforeAlways {
		needed = append(needed, CapabilityCoordinate)
	}
	if request.AfterBuild != AskAfterAlways {
		needed = append(needed, CapabilityReview, CapabilityAccept, CapabilityMerge)
	}
	if !current.InstallationAdmin {
		capabilities, err := readCapabilities(ctx, tx, current.UserID, request.ProjectID)
		if err != nil {
			return Project{}, err
		}
		for _, capability := range needed {
			if !hasCapability(capabilities, capability) {
				return Project{}, fault(CodeForbidden, "you can only delegate decisions you are allowed to make: "+string(capability)+" is missing", nil)
			}
		}
	}
	if request.AfterBuild == AskAfterNever {
		trusted, err := trustedMerges(ctx, tx, request.ProjectID)
		if err != nil {
			return Project{}, err
		}
		if trusted < TrustedMergesForNever {
			return Project{}, detailFault(CodeNotReady, fmt.Sprintf("merging without asking unlocks after %d merges with no reported problem", TrustedMergesForNever), map[string]any{"trustedMerges": trusted}, nil)
		}
	}
	project, err := readProject(ctx, tx, request.ProjectID)
	if err != nil {
		return Project{}, err
	}
	now := s.state.now().UTC()
	project.Policy.Ask = AskPolicy{BeforeBuild: request.BeforeBuild, AfterBuild: request.AfterBuild, Autopilot: request.Autopilot, SetBy: &current.UserID, SetAt: &now}
	if err := saveProjectPolicy(ctx, tx, project); err != nil {
		return Project{}, err
	}
	if err := saveReceipt(ctx, tx, current.UserID, "set_ask_policy", request.RequestID, hash, project, false, now); err != nil {
		return Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return Project{}, fault(CodeInternal, "commit ask policy", err)
	}
	return project, nil
}

func saveProjectPolicy(ctx context.Context, tx *sql.Tx, project Project) error {
	encoded, err := marshal(project.Policy)
	if err != nil {
		return fault(CodeInternal, "encode project policy", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE projects SET policy_json=? WHERE id=?`, encoded, project.ID); err != nil {
		return fault(CodeInternal, "save project policy", err)
	}
	return nil
}

// trustedMerges counts the merges since the project's last reported problem.
func trustedMerges(ctx context.Context, q queryer, projectID ProjectID) (int, error) {
	var since string
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(MAX(e.created_at),'') FROM events e JOIN jobs j ON j.id=e.job_id WHERE j.project_id=? AND e.kind='problem_reported'`, projectID).Scan(&since); err != nil {
		return 0, fault(CodeInternal, "read problem reports", err)
	}
	var count int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(DISTINCT e.job_id) FROM events e JOIN jobs j ON j.id=e.job_id WHERE j.project_id=? AND e.kind='job_merged' AND e.created_at > ? AND NOT EXISTS (SELECT 1 FROM events p WHERE p.job_id=e.job_id AND p.kind='problem_reported')`, projectID, since).Scan(&count); err != nil {
		return 0, fault(CodeInternal, "count trusted merges", err)
	}
	return count, nil
}

type ReportProblemRequest struct {
	RequestID string `json:"requestId"`
	JobID     JobID  `json:"jobId"`
	Summary   string `json:"summary"`
}

// ReportProblem records that merged work turned out wrong. It resets the project's
// trust count and makes the factory ask one level more often.
func (s *Service) ReportProblem(ctx context.Context, actor Principal, request ReportProblemRequest) (Job, error) {
	summary := strings.TrimSpace(request.Summary)
	if request.JobID == "" || summary == "" {
		return Job{}, fault(CodeInvalid, "jobId and summary are required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start problem report", err)
	}
	defer tx.Rollback()
	projectID, err := jobProject(ctx, tx, request.JobID)
	if err != nil {
		return Job{}, err
	}
	current, err := authorizeProject(ctx, tx, actor, projectID, CapabilityCoordinate)
	if err != nil {
		return Job{}, err
	}
	if current.Kind != UserKindHuman {
		return Job{}, fault(CodeForbidden, "only a person can report a problem", nil)
	}
	saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "report_problem", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	job, err := loadJob(ctx, tx, request.JobID)
	if err != nil {
		return Job{}, err
	}
	if job.Status != JobAccepted || !job.Merge.Merged() {
		return Job{}, fault(CodeNotReady, "problems are reported on merged work", nil)
	}
	now := s.state.now().UTC()
	if _, err := appendEvent(ctx, tx, job.ID, &current.UserID, "problem_reported", map[string]any{"summary": truncateRunes(summary, maxRefinementRunes)}, now); err != nil {
		return Job{}, err
	}
	project, err := readProject(ctx, tx, projectID)
	if err != nil {
		return Job{}, err
	}
	lowered := project.Policy.Ask
	switch lowered.After() {
	case AskAfterNever:
		lowered.AfterBuild = AskAfterWhenUntested
	case AskAfterWhenUntested:
		lowered.AfterBuild = AskAfterAlways
	}
	if lowered.AfterBuild != project.Policy.Ask.AfterBuild {
		project.Policy.Ask = lowered
		if err := saveProjectPolicy(ctx, tx, project); err != nil {
			return Job{}, err
		}
		if _, err := appendEvent(ctx, tx, job.ID, &current.UserID, "ask_policy_lowered", map[string]any{"afterBuild": lowered.AfterBuild}, now); err != nil {
			return Job{}, err
		}
	}
	if job, err = loadJob(ctx, tx, job.ID); err != nil {
		return Job{}, err
	}
	job.AvailableActions = availableActions(job)
	if err := saveReceipt(ctx, tx, current.UserID, "report_problem", request.RequestID, hash, job, false, now); err != nil {
		return Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return Job{}, fault(CodeInternal, "commit problem report", err)
	}
	return job, nil
}
