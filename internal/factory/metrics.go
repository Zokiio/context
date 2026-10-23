package factory

import (
	"context"
	"encoding/json"
	"sort"
	"time"
)

// ProjectMetrics are measured from the factory's own records: jobs, attempts, events,
// artifacts, refinements, and expectations. They answer how much of the work ran
// without rework, how long it took, how often a person decided something, and what the
// agents cost. Nothing here is estimated.
type ProjectMetrics struct {
	ProjectID   ProjectID `json:"projectId"`
	GeneratedAt time.Time `json:"generatedAt"`

	Jobs                int     `json:"jobs"`
	ActiveJobs          int     `json:"activeJobs"`
	AcceptedJobs        int     `json:"acceptedJobs"`
	MergedJobs          int     `json:"mergedJobs"`
	FailedJobs          int     `json:"failedJobs"`
	CancelledJobs       int     `json:"cancelledJobs"`
	FirstPassAccepted   int     `json:"firstPassAccepted"`
	FirstPassRate       float64 `json:"firstPassRate"`
	Repairs             int     `json:"repairs"`
	ProtectedRejections int     `json:"protectedRejections"`

	MergeChecksPassed int `json:"mergeChecksPassed"`
	MergeChecksFailed int `json:"mergeChecksFailed"`
	MergeFailures     int `json:"mergeFailures"`

	MedianAdmitToAcceptSeconds float64 `json:"medianAdmitToAcceptSeconds"`
	MedianAcceptToMergeSeconds float64 `json:"medianAcceptToMergeSeconds"`

	HumanDecisions         int     `json:"humanDecisions"`
	HumanDecisionsPerMerge float64 `json:"humanDecisionsPerMerge"`

	Refinements  DecisionCounts `json:"refinements"`
	Expectations DecisionCounts `json:"expectations"`

	ImplementationCostUSD float64 `json:"implementationCostUsd"`
	RefinementCostUSD     float64 `json:"refinementCostUsd"`
	ExpectationCostUSD    float64 `json:"expectationCostUsd"`
	TotalCostUSD          float64 `json:"totalCostUsd"`
	CostPerMergedJobUSD   float64 `json:"costPerMergedJobUsd"`
}

type DecisionCounts struct {
	Total    int `json:"total"`
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
	Failed   int `json:"failed"`
}

// humanEvents are the job events that record a person's decision.
var humanEvents = map[string]bool{"job_admitted": true, "review_recorded": true, "acceptance_published": true, "merge_verifying": true}

func (s *Service) Metrics(ctx context.Context, actor Principal, projectID ProjectID) (ProjectMetrics, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return ProjectMetrics{}, err
	}
	metrics := ProjectMetrics{ProjectID: projectID, GeneratedAt: s.state.now().UTC()}
	type jobRow struct {
		status          JobStatus
		repairs         int
		created         time.Time
		accepted, merge *time.Time
		merged          bool
	}
	jobs := map[JobID]*jobRow{}
	rows, err := s.state.db.QueryContext(ctx, `SELECT id, status, repair_count, merge_json, created_at FROM jobs WHERE project_id=?`, projectID)
	if err != nil {
		return ProjectMetrics{}, fault(CodeInternal, "read jobs for metrics", err)
	}
	for rows.Next() {
		var id JobID
		var row jobRow
		var mergeJSON, created string
		if err := rows.Scan(&id, &row.status, &row.repairs, &mergeJSON, &created); err != nil {
			rows.Close()
			return ProjectMetrics{}, fault(CodeInternal, "decode jobs for metrics", err)
		}
		row.created, _ = parseTime(created)
		if mergeJSON != "" {
			var record MergeRecord
			if json.Unmarshal([]byte(mergeJSON), &record) == nil && record.Merged() {
				row.merged = true
			}
		}
		jobs[id] = &row
	}
	rows.Close()
	events, err := s.state.db.QueryContext(ctx, `SELECT e.job_id, e.kind, e.created_at, e.actor_id IS NOT NULL FROM events e JOIN jobs j ON j.id=e.job_id WHERE j.project_id=? ORDER BY e.sequence`, projectID)
	if err != nil {
		return ProjectMetrics{}, fault(CodeInternal, "read events for metrics", err)
	}
	for events.Next() {
		var id JobID
		var kind, created string
		var byPerson bool
		if err := events.Scan(&id, &kind, &created, &byPerson); err != nil {
			events.Close()
			return ProjectMetrics{}, fault(CodeInternal, "decode events for metrics", err)
		}
		at, _ := parseTime(created)
		row := jobs[id]
		switch kind {
		case "acceptance_published":
			row.accepted = &at
		case "job_merged":
			row.merge = &at
		case "protected_paths_changed":
			metrics.ProtectedRejections++
		case "merge_failed":
			metrics.MergeFailures++
		}
		if humanEvents[kind] || (kind == "job_merged" && byPerson) {
			metrics.HumanDecisions++
		}
	}
	events.Close()
	var toAccept, toMerge []float64
	for _, row := range jobs {
		metrics.Jobs++
		metrics.Repairs += row.repairs
		switch row.status {
		case JobAccepted:
			metrics.AcceptedJobs++
			if row.repairs == 0 {
				metrics.FirstPassAccepted++
			}
		case JobFailed, JobVerificationFailed:
			metrics.FailedJobs++
		case JobCancelled:
			metrics.CancelledJobs++
		default:
			metrics.ActiveJobs++
		}
		if row.merged {
			metrics.MergedJobs++
		}
		if row.accepted != nil {
			toAccept = append(toAccept, row.accepted.Sub(row.created).Seconds())
			if row.merge != nil {
				toMerge = append(toMerge, row.merge.Sub(*row.accepted).Seconds())
			}
		}
	}
	if metrics.AcceptedJobs > 0 {
		metrics.FirstPassRate = float64(metrics.FirstPassAccepted) / float64(metrics.AcceptedJobs)
	}
	metrics.MedianAdmitToAcceptSeconds, metrics.MedianAcceptToMergeSeconds = median(toAccept), median(toMerge)
	if err := s.state.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN a.check_status=? THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN a.check_status IN (?,?,?) THEN 1 ELSE 0 END),0) FROM attempts a JOIN jobs j ON j.id=a.job_id WHERE j.project_id=? AND a.purpose=?`, CheckPassed, CheckFailed, CheckUnknown, CheckUnavailable, projectID, AttemptPurposeMerge).Scan(&metrics.MergeChecksPassed, &metrics.MergeChecksFailed); err != nil {
		return ProjectMetrics{}, fault(CodeInternal, "read merge checks for metrics", err)
	}
	results, err := s.state.db.QueryContext(ctx, `SELECT a.content FROM artifacts a JOIN jobs j ON j.id=a.job_id WHERE j.project_id=? AND a.name='agent-result.json'`, projectID)
	if err != nil {
		return ProjectMetrics{}, fault(CodeInternal, "read agent results for metrics", err)
	}
	for results.Next() {
		var content string
		var result struct {
			CostUSD float64 `json:"costUsd"`
		}
		if results.Scan(&content) == nil && json.Unmarshal([]byte(content), &result) == nil {
			metrics.ImplementationCostUSD += result.CostUSD
		}
	}
	results.Close()
	refinements, err := s.Refinements(ctx, actor, projectID, "")
	if err != nil {
		return ProjectMetrics{}, err
	}
	for _, refinement := range refinements {
		metrics.Refinements.count(string(refinement.Status))
		if refinement.Run != nil {
			metrics.RefinementCostUSD += refinement.Run.CostUSD
		}
		if refinement.Status == RefinementAccepted || refinement.Status == RefinementRejected {
			metrics.HumanDecisions++
		}
	}
	expectations, err := s.ListExpectations(ctx, actor, projectID, "")
	if err != nil {
		return ProjectMetrics{}, err
	}
	for _, expectation := range expectations {
		metrics.Expectations.count(string(expectation.Status))
		if expectation.Run != nil {
			metrics.ExpectationCostUSD += expectation.Run.CostUSD
		}
		if expectation.Status == ExpectationAccepted || expectation.Status == ExpectationRejected {
			metrics.HumanDecisions++
		}
	}
	metrics.TotalCostUSD = metrics.ImplementationCostUSD + metrics.RefinementCostUSD + metrics.ExpectationCostUSD
	if metrics.MergedJobs > 0 {
		metrics.CostPerMergedJobUSD = metrics.TotalCostUSD / float64(metrics.MergedJobs)
		metrics.HumanDecisionsPerMerge = float64(metrics.HumanDecisions) / float64(metrics.MergedJobs)
	}
	return metrics, nil
}

func (counts *DecisionCounts) count(status string) {
	counts.Total++
	switch status {
	case "accepted":
		counts.Accepted++
	case "rejected":
		counts.Rejected++
	case "failed", "stale":
		counts.Failed++
	}
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}
