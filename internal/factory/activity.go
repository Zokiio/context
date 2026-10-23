package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const activityHistoryLimit = 100

// WorkItemActivity is a snapshot of retained execution records. Lists are newest
// first and include cancelled jobs. Truncated reports an older history beyond the
// limit; it never means that the retained records were deleted.
type WorkItemActivity struct {
	ProjectID    ProjectID       `json:"projectId"`
	WorkItemID   string          `json:"workItemId"`
	GeneratedAt  time.Time       `json:"generatedAt"`
	Held         bool            `json:"held"`
	Jobs         []Job           `json:"jobs"`
	Refinements  []Refinement    `json:"refinements"`
	Expectations []Expectation   `json:"expectations"`
	Actors       []ActivityActor `json:"actors"`
	Truncated    bool            `json:"truncated"`
}

// ActivityActor identifies a participant already named by a retained record.
// Credentials and installation privileges are not part of a project's history.
type ActivityActor struct {
	ID       UserID   `json:"id"`
	Username string   `json:"username"`
	Kind     UserKind `json:"kind"`
}

// WorkItemActivity reads the task's operational history independently of its board
// card. Artifact contents remain available through the authenticated artifact
// endpoint; this polling snapshot includes their identity and metadata only.
// Raw terminal receipts, which may embed logs, remain available in job details.
func (s *Service) WorkItemActivity(ctx context.Context, actor Principal, projectID ProjectID, workItemID string) (WorkItemActivity, error) {
	if projectID == "" || strings.TrimSpace(workItemID) == "" {
		return WorkItemActivity{}, fault(CodeInvalid, "projectId and workItemId are required", nil)
	}
	tx, err := s.state.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return WorkItemActivity{}, fault(CodeInternal, "start work item activity snapshot", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, projectID, CapabilityRead)
	if err != nil {
		return WorkItemActivity{}, err
	}
	capabilities, err := readCapabilities(ctx, tx, current.UserID, projectID)
	if err != nil {
		return WorkItemActivity{}, err
	}
	result := WorkItemActivity{ProjectID: projectID, WorkItemID: workItemID, GeneratedAt: s.state.now().UTC(), Jobs: []Job{}, Actors: []ActivityActor{}}
	result.Held, err = workItemHeld(ctx, tx, projectID, workItemID)
	if err != nil {
		return WorkItemActivity{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM jobs WHERE project_id=? AND work_item_id=? ORDER BY created_at DESC, id DESC LIMIT ?`, projectID, workItemID, activityHistoryLimit+1)
	if err != nil {
		return WorkItemActivity{}, fault(CodeInternal, "list work item jobs", err)
	}
	ids := []JobID{}
	for rows.Next() {
		var id JobID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return WorkItemActivity{}, fault(CodeInternal, "read work item job identity", err)
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return WorkItemActivity{}, fault(CodeInternal, "read work item jobs", err)
	}
	if len(ids) > activityHistoryLimit {
		result.Truncated = true
		ids = ids[:activityHistoryLimit]
	}
	actors := map[UserID]bool{}
	addActor := func(id *UserID) {
		if id != nil && *id != "" {
			actors[*id] = true
		}
	}
	for _, id := range ids {
		job, err := loadJobRecord(ctx, tx, id, false)
		if err != nil {
			return WorkItemActivity{}, err
		}
		job.AvailableActions = permittedJobActions(job, current, capabilities)
		addActor(&job.CoordinatorID)
		addActor(job.ImplementerID)
		for index := range job.Attempts {
			job.Attempts[index].TerminalReceipt = ""
			addActor(job.Attempts[index].WorkerID)
		}
		for index := range job.Checks {
			addActor(job.Checks[index].ObserverID)
		}
		if job.Merge != nil {
			addActor(&job.Merge.MergedBy)
		}
		for _, review := range job.Reviews {
			addActor(&review.ReviewerID)
		}
		for _, event := range job.Events {
			addActor(event.ActorID)
		}
		result.Jobs = append(result.Jobs, job)
	}
	var truncated bool
	result.Refinements, truncated, err = activityRecords[Refinement](ctx, tx, "refinements", projectID, workItemID)
	if err != nil {
		return WorkItemActivity{}, err
	}
	result.Truncated = result.Truncated || truncated
	for _, refinement := range result.Refinements {
		addActor(&refinement.RequestedBy)
		addActor(refinement.DecidedBy)
	}
	result.Expectations, truncated, err = activityRecords[Expectation](ctx, tx, "expectations", projectID, workItemID)
	if err != nil {
		return WorkItemActivity{}, err
	}
	result.Truncated = result.Truncated || truncated
	for _, expectation := range result.Expectations {
		addActor(&expectation.RequestedBy)
		addActor(expectation.DecidedBy)
	}
	for id := range actors {
		user, err := readUser(ctx, tx, id)
		if err != nil {
			return WorkItemActivity{}, err
		}
		result.Actors = append(result.Actors, ActivityActor{ID: id, Username: user.Username, Kind: user.Kind})
	}
	sort.Slice(result.Actors, func(i, j int) bool { return result.Actors[i].ID < result.Actors[j].ID })
	if err := tx.Commit(); err != nil {
		return WorkItemActivity{}, fault(CodeInternal, "finish work item activity snapshot", err)
	}
	// Never require current requirements to reproduce a retained job's history.
	// A task with no history must still exist in the authored project records.
	if len(result.Jobs)+len(result.Refinements)+len(result.Expectations) == 0 {
		if _, err := s.GetWorkItem(ctx, actor, projectID, workItemID); err != nil {
			return WorkItemActivity{}, err
		}
	}
	return result, nil
}

func permittedJobActions(job Job, actor Principal, capabilities []Capability) []ActionKind {
	actions := []ActionKind{}
	for _, action := range availableActions(job) {
		required := CapabilityCoordinate
		if action == ActionResolve {
			required = CapabilityProjectAdmin
		}
		if actor.InstallationAdmin || hasCapability(capabilities, required) {
			actions = append(actions, action)
		}
	}
	return actions
}

func activityRecords[T any](ctx context.Context, q queryer, table string, projectID ProjectID, workItemID string) ([]T, bool, error) {
	// Table names are constants at the two call sites, never request input.
	rows, err := q.QueryContext(ctx, `SELECT data_json FROM `+table+` WHERE project_id=? AND work_item_id=? ORDER BY created_at DESC, id DESC LIMIT ?`, projectID, workItemID, activityHistoryLimit+1)
	if err != nil {
		return nil, false, fault(CodeInternal, "list work item "+table, err)
	}
	defer rows.Close()
	result := []T{}
	for rows.Next() {
		var encoded string
		var record T
		if err := rows.Scan(&encoded); err != nil {
			return nil, false, fault(CodeInternal, "read work item "+table, err)
		}
		if err := json.Unmarshal([]byte(encoded), &record); err != nil {
			return nil, false, fault(CodeInternal, "decode work item "+table, err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fault(CodeInternal, "read work item "+table, err)
	}
	if len(result) > activityHistoryLimit {
		return result[:activityHistoryLimit], true, nil
	}
	return result, false, nil
}
