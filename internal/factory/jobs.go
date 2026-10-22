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
	"path/filepath"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/taskcontext"
)

func (s *Service) Submit(ctx context.Context, actor Principal, request SubmitRequest) (Job, error) {
	if request.ProjectID == "" || strings.TrimSpace(request.WorkItemID) == "" || strings.TrimSpace(request.BaseRevision) == "" || strings.TrimSpace(request.ExecutorKey) == "" {
		return Job{}, fault(CodeInvalid, "projectId, workItemId, baseRevision, and executorKey are required", nil)
	}
	if err := requireRequestID(request.RequestID); err != nil {
		return Job{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()

	current, err := authorizeProject(ctx, s.state.db, actor, request.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Job{}, err
	}
	if replayed, ok, err := s.readJobReceipt(ctx, current.UserID, "submit", request.RequestID, request); err != nil || ok {
		return replayed, err
	}
	project, err := readProject(ctx, s.state.db, request.ProjectID)
	if err != nil {
		return Job{}, err
	}
	if !contains(project.Policy.AllowedExecutors, request.ExecutorKey) {
		return Job{}, fault(CodeInvalid, "executor is not allowed by project policy", nil)
	}
	if request.RepairLimit < 0 || request.RepairLimit > project.Policy.MaxRepairAttempts {
		return Job{}, fault(CodeInvalid, "repairLimit exceeds project policy", nil)
	}
	now := s.state.now().UTC()
	if !now.Before(request.Deadline) {
		return Job{}, fault(CodeDeadline, "deadline must be in the future", nil)
	}
	if request.Deadline.After(now.Add(time.Duration(project.Policy.MaxJobDurationSeconds) * time.Second)) {
		return Job{}, fault(CodeInvalid, "deadline exceeds project policy", nil)
	}
	var executorFixture bool
	if err := s.state.db.QueryRowContext(ctx, `SELECT fixture FROM executor_profiles WHERE key=?`, request.ExecutorKey).Scan(&executorFixture); errors.Is(err, sql.ErrNoRows) {
		return Job{}, fault(CodeInvalid, "executor profile is unavailable", nil)
	} else if err != nil {
		return Job{}, fault(CodeInternal, "read executor profile", err)
	}
	if executorFixture && !project.Policy.Fixture {
		return Job{}, fault(CodeInvalid, "fixture executor cannot run an ordinary project", nil)
	}
	base, err := s.resolveCommit(ctx, project.RepositoryPath, request.BaseRevision)
	if err != nil {
		return Job{}, err
	}

	preliminary, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return Job{}, fault(CodeInternal, "locate work item", err)
	}
	work, err := selectWork(preliminary, request.WorkItemID)
	if err != nil {
		return Job{}, err
	}
	if err := prepareAcceptanceSection(work.Source); err != nil {
		return Job{}, err
	}

	capture, err := recordread.NewCapture(project.BundlePath, project.AllowedSourceRoots, recordread.Limits{MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return Job{}, fault(CodeInternal, "create admission capture", err)
	}
	defer capture.Close()
	contextResult, err := taskcontext.AssembleWithCapture(ctx, work.Source, capture)
	if err != nil {
		return Job{}, fault(CodeInternal, "assemble task context", err)
	}
	orientationResult, err := orientation.OrientWithCapture(ctx, capture)
	if err != nil {
		return Job{}, fault(CodeInternal, "evaluate work item", err)
	}
	selected, err := selectWork(orientationResult, request.WorkItemID)
	if err != nil {
		return Job{}, err
	}
	if !contextResult.Complete || !contextResult.TraversalComplete {
		return Job{}, detailFault(CodeNotReady, "task context is incomplete", map[string]any{"diagnostics": contextResult.Diagnostics}, nil)
	}
	if !orientationResult.InventoryComplete {
		return Job{}, detailFault(CodeNotReady, "project identity inventory is incomplete", map[string]any{"diagnostics": orientationResult.Diagnostics}, nil)
	}
	if selected.Readiness != "ready" || !selected.Eligible {
		return Job{}, detailFault(CodeNotReady, "work item is not ready and eligible", map[string]any{"readiness": selected.Readiness, "reasons": selected.ExclusionReasons}, nil)
	}
	if selected.FingerprintVersion == nil || selected.TicketSHA256 == nil || selected.CriteriaSHA256 == nil {
		return Job{}, fault(CodeNotReady, "work item requirement fingerprints are unavailable", nil)
	}
	if selected.Source != work.Source {
		return Job{}, fault(CodeStale, "work item identity changed during admission", nil)
	}
	if orientationResult.Project == nil || ProjectID(orientationResult.Project.ID) != project.ID {
		return Job{}, fault(CodeStale, "project identity changed during admission", nil)
	}

	sources, diagnostics, contextDigest, err := admissionContext(contextResult)
	if err != nil {
		return Job{}, fault(CodeInternal, "retain admission context", err)
	}
	sources, contextDigest, err = appendSelectedRequirements(capture, orientationResult, selected.Source, sources)
	if err != nil {
		return Job{}, fault(CodeInternal, "retain selected requirements", err)
	}
	if err := checkSourcesUnchanged(sources); err != nil {
		return Job{}, err
	}
	admission := Admission{
		TicketPath: selected.Source, BaseRevision: base, FingerprintVersion: *selected.FingerprintVersion,
		TicketSHA256: *selected.TicketSHA256, CriteriaSHA256: *selected.CriteriaSHA256,
		ContextSHA256: contextDigest, ContextComplete: true, Sources: sources, Diagnostics: diagnostics,
		ExecutorKey: request.ExecutorKey, RequiredChecks: append([]string(nil), project.Policy.RequiredChecks...),
		RepairLimit: request.RepairLimit, Deadline: request.Deadline.UTC(), AdmittedAt: now,
	}
	jobID, err := newID("job_")
	if err != nil {
		return Job{}, fault(CodeInternal, "create job identity", err)
	}
	attemptID, err := newID("att_")
	if err != nil {
		return Job{}, fault(CodeInternal, "create attempt identity", err)
	}
	dispatchID, err := newID("dsp_")
	if err != nil {
		return Job{}, fault(CodeInternal, "create dispatch identity", err)
	}
	workspacePath := s.workspacePath(JobID(jobID), AttemptID(attemptID))

	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, fault(CodeInternal, "start admission transaction", err)
	}
	defer tx.Rollback()
	current, err = authorizeProject(ctx, tx, actor, request.ProjectID, CapabilityCoordinate)
	if err != nil {
		return Job{}, err
	}
	if saved, ok, hash, err := receipt[Job](ctx, tx, current.UserID, "submit", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		if err := checkSourcesUnchanged(sources); err != nil {
			return Job{}, err
		}
		var existing JobID
		if err := tx.QueryRowContext(ctx, `SELECT id FROM jobs WHERE project_id=? AND work_item_id=? AND active_claim=1`, project.ID, request.WorkItemID).Scan(&existing); err == nil {
			return Job{}, detailFault(CodeConflict, "work item already has an active implementation claim", map[string]any{"jobId": existing}, nil)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return Job{}, fault(CodeInternal, "inspect implementation claim", err)
		}
		admissionJSON, err := marshal(admission)
		if err != nil {
			return Job{}, fault(CodeInternal, "encode admission", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO jobs(id, project_id, work_item_id, coordinator_id, status, version, admission_json, created_at, updated_at) VALUES(?,?,?,?,?,1,?,?,?)`, jobID, project.ID, request.WorkItemID, current.UserID, JobQueued, admissionJSON, formatTime(now), formatTime(now)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "one_active_implementation_claim") || strings.Contains(strings.ToLower(err.Error()), "unique") {
				return Job{}, fault(CodeConflict, "work item already has an active implementation claim", err)
			}
			return Job{}, fault(CodeInternal, "create job", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO attempts(id, job_id, stage, ordinal, dispatch_id, workspace_path, input_revision, status) VALUES(?,?,?,?,?,?,?,?)`, attemptID, jobID, StageImplementation, 1, dispatchID, workspacePath, base, AttemptPending); err != nil {
			return Job{}, fault(CodeInternal, "create implementation attempt", err)
		}
		if _, err := appendEvent(ctx, tx, JobID(jobID), &current.UserID, "job_admitted", map[string]any{"attemptId": attemptID, "baseRevision": base, "contextSHA256": contextDigest}, now); err != nil {
			return Job{}, err
		}
		job, err := loadJob(ctx, tx, JobID(jobID))
		if err != nil {
			return Job{}, err
		}
		if err := saveReceipt(ctx, tx, current.UserID, "submit", request.RequestID, hash, job, false, now); err != nil {
			return Job{}, err
		}
		if err := tx.Commit(); err != nil {
			return Job{}, fault(CodeInternal, "commit job admission", err)
		}
		return job, nil
	}
}

func (s *Service) Jobs(ctx context.Context, actor Principal, filter JobFilter) ([]JobSummary, error) {
	current, err := requirePrincipal(ctx, s.state.db, actor)
	if err != nil {
		return nil, err
	}
	if filter.ProjectID != "" {
		if _, err := authorizeProject(ctx, s.state.db, current, filter.ProjectID, CapabilityRead); err != nil {
			return nil, err
		}
	}
	query := `SELECT j.id, j.version, j.project_id, j.work_item_id, j.status, j.candidate_revision, j.admission_json, j.updated_at FROM jobs j`
	where := []string{}
	args := []any{}
	if !current.InstallationAdmin {
		where = append(where, `EXISTS(SELECT 1 FROM project_grants g WHERE g.project_id=j.project_id AND g.user_id=? AND g.capability=?)`)
		args = append(args, current.UserID, CapabilityRead)
	}
	if filter.ProjectID != "" {
		where = append(where, `j.project_id=?`)
		args = append(args, filter.ProjectID)
	}
	if filter.Status != "" {
		where = append(where, `j.status=?`)
		args = append(args, filter.Status)
	}
	if filter.After != "" {
		where = append(where, `j.id>?`)
		args = append(args, filter.After)
	}
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY j.id`
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, limit)
	rows, err := s.state.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fault(CodeInternal, "list jobs", err)
	}
	defer rows.Close()
	jobs := []JobSummary{}
	for rows.Next() {
		var summary JobSummary
		var admissionJSON, updated string
		var admission Admission
		if err := rows.Scan(&summary.ID, &summary.Version, &summary.ProjectID, &summary.WorkItemID, &summary.Status, &summary.CandidateRevision, &admissionJSON, &updated); err != nil {
			return nil, fault(CodeInternal, "read job summary", err)
		}
		if err := json.Unmarshal([]byte(admissionJSON), &admission); err != nil {
			return nil, fault(CodeInternal, "decode job admission", err)
		}
		summary.Deadline = admission.Deadline
		summary.UpdatedAt, _ = parseTime(updated)
		jobs = append(jobs, summary)
	}
	return jobs, rows.Err()
}

func (s *Service) GetJob(ctx context.Context, actor Principal, jobID JobID) (Job, error) {
	projectID, err := jobProject(ctx, s.state.db, jobID)
	if err != nil {
		return Job{}, err
	}
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return Job{}, err
	}
	job, err := loadJob(ctx, s.state.db, jobID)
	if err != nil {
		return Job{}, err
	}
	job.AvailableActions = availableActions(job)
	return job, nil
}

func (s *Service) readJobReceipt(ctx context.Context, actor UserID, operation, requestID string, request any) (Job, bool, error) {
	tx, err := s.state.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Job{}, false, fault(CodeInternal, "read command receipt", err)
	}
	defer tx.Rollback()
	job, ok, _, err := receipt[Job](ctx, tx, actor, operation, requestID, request)
	return job, ok, err
}

func admissionContext(result taskcontext.Result) ([]ContextSource, []ContextDiagnostic, string, error) {
	sources := make([]ContextSource, 0, len(result.Sources))
	for _, source := range result.Sources {
		reasons, err := json.Marshal(source.Reasons)
		if err != nil {
			return nil, nil, "", err
		}
		sources = append(sources, ContextSource{Path: source.Path, Text: source.Text, SHA256: source.SHA256, Reasons: reasons})
	}
	diagnostics := make([]ContextDiagnostic, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		diagnostics = append(diagnostics, ContextDiagnostic{Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message, Path: diagnostic.Path})
	}
	encoded, err := json.Marshal(sources)
	if err != nil {
		return nil, nil, "", err
	}
	digest := sha256.Sum256(encoded)
	return sources, diagnostics, hex.EncodeToString(digest[:]), nil
}

func appendSelectedRequirements(capture *recordread.Capture, result orientation.Result, ticket string, sources []ContextSource) ([]ContextSource, string, error) {
	seen := map[string]bool{}
	for _, source := range sources {
		seen[source.Path] = true
	}
	for _, source := range result.Sources {
		selected := false
		for _, reason := range source.Reasons {
			if reason.From == ticket && (reason.Kind == "spec" || reason.Kind == "context" || reason.Kind == "blocked_by_decision") {
				selected = true
				break
			}
		}
		if !selected || seen[source.Path] {
			continue
		}
		if !capture.Admitted(source.Path, recordread.DocumentSource) {
			return nil, "", fmt.Errorf("selected requirement was not admitted: %s", source.Path)
		}
		captured, diagnostic := capture.Read(source.Path, recordread.DocumentSource)
		if diagnostic != nil {
			return nil, "", errors.New(diagnostic.Message)
		}
		reasons, err := json.Marshal(source.Reasons)
		if err != nil {
			return nil, "", err
		}
		sources = append(sources, ContextSource{Path: captured.Path, Text: captured.Text, SHA256: captured.SHA256, Reasons: reasons})
		seen[captured.Path] = true
	}
	encoded, err := json.Marshal(sources)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(encoded)
	return sources, hex.EncodeToString(digest[:]), nil
}

func selectWork(result orientation.Result, workItemID string) (orientation.WorkItem, error) {
	var found *orientation.WorkItem
	for index := range result.WorkItems {
		work := result.WorkItems[index]
		if work.ID == nil || *work.ID != workItemID {
			continue
		}
		if found != nil || work.IdentityAmbiguous {
			return orientation.WorkItem{}, fault(CodeConflict, "work item identity is ambiguous", nil)
		}
		copy := work
		found = &copy
	}
	if found == nil {
		return orientation.WorkItem{}, fault(CodeNotFound, "work item not found", nil)
	}
	return *found, nil
}

func prepareAcceptanceSection(path string) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return fault(CodeStale, "read work item before admission", err)
	}
	document, err := recordread.ParseDocument(original)
	if err != nil {
		return fault(CodeNotReady, "parse work item before admission", err)
	}
	sections := recordread.Sections(document.Body, path, map[string]string{"Acceptance": "acceptance"})
	if len(sections.Sections) > 0 || sections.InvalidSections["Acceptance"] {
		return nil
	}
	updated := append([]byte(nil), original...)
	if len(updated) > 0 && updated[len(updated)-1] != '\n' {
		updated = append(updated, '\n')
	}
	updated = append(updated, []byte("\n## Acceptance\n\nNone\n")...)
	return writeFileCAS(path, original, updated)
}

func writeFileCAS(path string, expected, updated []byte) error {
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fault(CodeInternal, "open record lock", err)
	}
	defer lock.Close()
	if err := lockFile(lock, false); err != nil {
		return fault(CodeInternal, "lock record", err)
	}
	defer unlockFile(lock)
	current, err := os.ReadFile(path)
	if err != nil {
		return fault(CodeStale, "re-read record", err)
	}
	if !bytes.Equal(current, expected) {
		return fault(CodeStale, "record changed before write", nil)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fault(CodeInternal, "inspect record permissions", err)
	}
	temporary := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.factory-%d", filepath.Base(path), time.Now().UnixNano()))
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fault(CodeInternal, "create record temporary file", err)
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(updated); err != nil {
		return fault(CodeInternal, "write record temporary file", err)
	}
	if err := file.Sync(); err != nil {
		return fault(CodeInternal, "sync record temporary file", err)
	}
	if err := file.Close(); err != nil {
		return fault(CodeInternal, "close record temporary file", err)
	}
	current, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(current, expected) {
		return fault(CodeStale, "record changed before commit", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fault(CodeInternal, "commit record write", err)
	}
	remove = false
	directory, err := os.Open(filepath.Dir(path))
	if err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}

func checkSourcesUnchanged(sources []ContextSource) error {
	for _, source := range sources {
		content, err := os.ReadFile(source.Path)
		if err != nil {
			return detailFault(CodeStale, "captured source became unavailable", map[string]any{"path": source.Path}, err)
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != source.SHA256 || string(content) != source.Text {
			return detailFault(CodeStale, "captured source changed during admission", map[string]any{"path": source.Path}, nil)
		}
	}
	return nil
}

func (s *Service) resolveCommit(ctx context.Context, repository, revision string) (string, error) {
	command := exec.CommandContext(ctx, s.state.gitBinary, "-C", repository, "rev-parse", "--verify", revision+"^{commit}")
	output, err := command.Output()
	if err != nil {
		return "", fault(CodeInvalid, "baseRevision does not resolve to a commit", err)
	}
	commit := strings.TrimSpace(string(output))
	if commit == "" || strings.ContainsAny(commit, " \t\r\n") {
		return "", fault(CodeInvalid, "baseRevision resolved to an invalid commit", nil)
	}
	return commit, nil
}

func (s *Service) workspacePath(jobID JobID, attemptID AttemptID) string {
	return filepath.Join(s.state.workspaceRoot, string(jobID), string(attemptID), "checkout")
}

func loadJob(ctx context.Context, q queryer, jobID JobID) (Job, error) {
	var job Job
	var admissionJSON, created, updated string
	var implementer sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id, version, project_id, work_item_id, coordinator_id, implementer_id, status, admission_json, candidate_revision, repair_count, acceptance_record_path, acceptance_status, created_at, updated_at FROM jobs WHERE id=?`, jobID).Scan(
		&job.ID, &job.Version, &job.ProjectID, &job.WorkItemID, &job.CoordinatorID, &implementer, &job.Status, &admissionJSON, &job.CandidateRevision, &job.RepairCount, &job.AcceptanceRecordPath, &job.AcceptanceStatus, &created, &updated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, fault(CodeNotFound, "job not found", nil)
	}
	if err != nil {
		return Job{}, fault(CodeInternal, "read job", err)
	}
	if implementer.Valid {
		value := UserID(implementer.String)
		job.ImplementerID = &value
	}
	if err := json.Unmarshal([]byte(admissionJSON), &job.Admission); err != nil {
		return Job{}, fault(CodeInternal, "decode admission", err)
	}
	job.CreatedAt, _ = parseTime(created)
	job.UpdatedAt, _ = parseTime(updated)
	job.Attempts, err = loadAttempts(ctx, q, jobID)
	if err != nil {
		return Job{}, err
	}
	job.Checks, err = loadChecks(ctx, q, jobID, job.Admission.RequiredChecks, job.CandidateRevision, job.Admission.TicketSHA256)
	if err != nil {
		return Job{}, err
	}
	job.Reviews, err = loadReviews(ctx, q, jobID)
	if err != nil {
		return Job{}, err
	}
	job.Events, err = loadEvents(ctx, q, jobID)
	return job, err
}

func loadAttempts(ctx context.Context, q queryer, jobID JobID) ([]Attempt, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, job_id, stage, ordinal, check_key, dispatch_id, worker_id, native_run_id, workspace_path, input_revision, candidate_revision, status, outcome, check_status, lease_expires_at, last_heartbeat_at, started_at, finished_at, cancellation_requested FROM attempts WHERE job_id=? ORDER BY ordinal, stage, check_key`, jobID)
	if err != nil {
		return nil, fault(CodeInternal, "read attempts", err)
	}
	attempts := []Attempt{}
	for rows.Next() {
		var attempt Attempt
		var worker, lease, heartbeat, started, finished sql.NullString
		if err := rows.Scan(&attempt.ID, &attempt.JobID, &attempt.Stage, &attempt.Ordinal, &attempt.CheckKey, &attempt.DispatchID, &worker, &attempt.NativeRunID, &attempt.WorkspacePath, &attempt.InputRevision, &attempt.CandidateRevision, &attempt.Status, &attempt.Outcome, &attempt.CheckStatus, &lease, &heartbeat, &started, &finished, &attempt.CancellationRequested); err != nil {
			rows.Close()
			return nil, fault(CodeInternal, "decode attempt", err)
		}
		if worker.Valid {
			value := UserID(worker.String)
			attempt.WorkerID = &value
		}
		attempt.LeaseExpiresAt, attempt.LastHeartbeatAt = nullableTime(lease), nullableTime(heartbeat)
		attempt.StartedAt, attempt.FinishedAt = nullableTime(started), nullableTime(finished)
		attempts = append(attempts, attempt)
	}
	err = rows.Err()
	rows.Close()
	return attempts, err
}

func loadChecks(ctx context.Context, q queryer, jobID JobID, required []string, candidate, requirement string) ([]CheckResult, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, check_key, status, candidate_revision, requirement_fingerprint, observer_id, attempt_id, command_json, environment, observed_at FROM check_results WHERE job_id=? ORDER BY created_at, id`, jobID)
	if err != nil {
		return nil, fault(CodeInternal, "read check results", err)
	}
	results := []CheckResult{}
	seenCurrent := map[string]bool{}
	for rows.Next() {
		var id string
		var result CheckResult
		var observer, attempt, observed sql.NullString
		var command string
		if err := rows.Scan(&id, &result.CheckKey, &result.Status, &result.CandidateRevision, &result.RequirementFingerprint, &observer, &attempt, &command, &result.Environment, &observed); err != nil {
			rows.Close()
			return nil, fault(CodeInternal, "decode check result", err)
		}
		_ = json.Unmarshal([]byte(command), &result.Command)
		if observer.Valid {
			value := UserID(observer.String)
			result.ObserverID = &value
		}
		if attempt.Valid {
			value := AttemptID(attempt.String)
			result.AttemptID = &value
		}
		result.ObservedAt = nullableTime(observed)
		if result.CandidateRevision == candidate && result.RequirementFingerprint == requirement {
			seenCurrent[result.CheckKey] = true
		}
		results = append(results, result)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for index := range results {
		results[index].Artifacts, err = loadArtifacts(ctx, q, jobID, results[index].AttemptID)
		if err != nil {
			return nil, err
		}
	}
	for _, check := range required {
		if !seenCurrent[check] {
			results = append(results, CheckResult{CheckKey: check, Status: CheckPending, CandidateRevision: candidate, RequirementFingerprint: requirement, Artifacts: []Artifact{}})
		}
	}
	return results, nil
}

func loadArtifacts(ctx context.Context, q queryer, jobID JobID, attemptID *AttemptID) ([]Artifact, error) {
	query := `SELECT id, name, media_type, sha256, byte_count, content, created_at FROM artifacts WHERE job_id=?`
	args := []any{jobID}
	if attemptID != nil {
		query += ` AND attempt_id=?`
		args = append(args, *attemptID)
	} else {
		query += ` AND attempt_id IS NULL`
	}
	query += ` ORDER BY created_at, id`
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fault(CodeInternal, "read artifacts", err)
	}
	defer rows.Close()
	artifacts := []Artifact{}
	for rows.Next() {
		var artifact Artifact
		var created string
		if err := rows.Scan(&artifact.ID, &artifact.Name, &artifact.MediaType, &artifact.SHA256, &artifact.Bytes, &artifact.Content, &created); err != nil {
			return nil, fault(CodeInternal, "decode artifact", err)
		}
		artifact.CreatedAt, _ = parseTime(created)
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}

func loadReviews(ctx context.Context, q queryer, jobID JobID) ([]Review, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, reviewer_id, candidate_revision, requirement_fingerprint, decision, summary, created_at FROM reviews WHERE job_id=? ORDER BY created_at, id`, jobID)
	if err != nil {
		return nil, fault(CodeInternal, "read reviews", err)
	}
	defer rows.Close()
	reviews := []Review{}
	for rows.Next() {
		var review Review
		var created string
		if err := rows.Scan(&review.ID, &review.ReviewerID, &review.CandidateRevision, &review.RequirementFingerprint, &review.Decision, &review.Summary, &created); err != nil {
			return nil, fault(CodeInternal, "decode review", err)
		}
		review.CreatedAt, _ = parseTime(created)
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func loadEvents(ctx context.Context, q queryer, jobID JobID) ([]Event, error) {
	rows, err := q.QueryContext(ctx, `SELECT sequence, job_id, actor_id, kind, data_json, created_at FROM events WHERE job_id=? ORDER BY sequence`, jobID)
	if err != nil {
		return nil, fault(CodeInternal, "read events", err)
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var event Event
		var actor sql.NullString
		var data, created string
		if err := rows.Scan(&event.Sequence, &event.JobID, &actor, &event.Kind, &data, &created); err != nil {
			return nil, fault(CodeInternal, "decode event", err)
		}
		if actor.Valid {
			value := UserID(actor.String)
			event.ActorID = &value
		}
		event.Data = json.RawMessage(data)
		event.CreatedAt, _ = parseTime(created)
		events = append(events, event)
	}
	return events, rows.Err()
}

func appendEvent(ctx context.Context, tx *sql.Tx, jobID JobID, actorID *UserID, kind string, data any, now time.Time) (Event, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return Event{}, fault(CodeInternal, "encode event", err)
	}
	var actor any
	if actorID != nil {
		actor = *actorID
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO events(job_id, actor_id, kind, data_json, created_at) VALUES(?,?,?,?,?)`, jobID, actor, kind, string(encoded), formatTime(now))
	if err != nil {
		return Event{}, fault(CodeInternal, "append event", err)
	}
	sequence, _ := result.LastInsertId()
	return Event{Sequence: sequence, JobID: jobID, ActorID: actorID, Kind: kind, Data: encoded, CreatedAt: now}, nil
}

func jobProject(ctx context.Context, q queryer, jobID JobID) (ProjectID, error) {
	var projectID ProjectID
	if err := q.QueryRowContext(ctx, `SELECT project_id FROM jobs WHERE id=?`, jobID).Scan(&projectID); errors.Is(err, sql.ErrNoRows) {
		return "", fault(CodeNotFound, "job not found", nil)
	} else if err != nil {
		return "", fault(CodeInternal, "read job project", err)
	}
	return projectID, nil
}

func availableActions(job Job) []ActionKind {
	switch job.Status {
	case JobQueued, JobReadyForReview, JobReadyForAcceptance, JobFailed:
		return []ActionKind{ActionCancel}
	case JobImplementing, JobVerifying:
		return []ActionKind{ActionCancel}
	case JobVerificationFailed:
		return []ActionKind{ActionRepair, ActionCancel}
	case JobImplementationUnknown, JobCancellationUnknown:
		return []ActionKind{ActionReconcile, ActionResolve}
	case JobCancelRequested:
		return []ActionKind{ActionReconcile}
	default:
		return []ActionKind{}
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
