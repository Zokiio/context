package factory

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
)

type WorkItemReference struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
type SetWorkItemDispositionRequest struct {
	RequestID      string    `json:"requestId"`
	ProjectID      ProjectID `json:"projectId"`
	WorkItemID     string    `json:"workItemId"`
	IfTicketSHA256 string    `json:"ifTicketSHA256"`
	Action         string    `json:"action"`
}

// ListWorkItems includes drafts and archived records under their stable identity.
// The queue is still the project's authored Current commitments section.
func (s *Service) ListWorkItems(ctx context.Context, actor Principal, projectID ProjectID) ([]WorkItemRecord, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return nil, err
	}
	project, err := readProject(ctx, s.state.db, projectID)
	if err != nil {
		return nil, err
	}
	result, err := s.orientProject(ctx, project)
	if err != nil {
		return nil, err
	}
	_, coordinateErr := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityCoordinate)
	if !result.InventoryComplete {
		return nil, detailFault(CodeNotReady, "task inventory is incomplete", map[string]any{"diagnostics": result.Diagnostics}, nil)
	}
	records := []WorkItemRecord{}
	for _, work := range result.WorkItems {
		if work.ID == nil || work.IdentityAmbiguous {
			continue
		}
		record, err := s.workItemRecord(ctx, s.state.db, project, result, work)
		if err != nil {
			return nil, err
		}
		if coordinateErr != nil {
			record.Actions = []string{}
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Draft.Title == records[j].Draft.Title {
			return records[i].WorkItemID < records[j].WorkItemID
		}
		return records[i].Draft.Title < records[j].Draft.Title
	})
	return records, nil
}

func workArchived(work orientation.WorkItem) bool {
	value, _ := work.Metadata["factoryArchived"].(bool)
	return value
}
func workQueued(work orientation.WorkItem) bool { return work.Committed != nil && *work.Committed }
func (s *Service) workItemRecord(ctx context.Context, q queryer, project Project, result orientation.Result, work orientation.WorkItem) (WorkItemRecord, error) {
	raw, err := os.ReadFile(work.Source)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "read work item", err)
	}
	criteria, err := acceptanceCriteria(work.Source)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInvalid, "read acceptance criteria", err)
	}
	draft := WorkItemDraft{Description: ticketDescription(raw), Criteria: criteria, BlockedBy: []string{}, Spec: sectionPaths(raw, work.Source, "Spec"), Context: sectionPaths(raw, work.Source, "Context")}
	if work.Title != nil {
		draft.Title = *work.Title
	}
	for _, edge := range work.Dependencies {
		if edge.To.ID != nil {
			draft.BlockedBy = append(draft.BlockedBy, *edge.To.ID)
		}
	}
	record := WorkItemRecord{ProjectID: project.ID, WorkItemID: *work.ID, Path: relativeRecord(project.BundlePath, work.Source), TicketSHA256: digestBytes(raw), Draft: draft, Queued: workQueued(work), Archived: workArchived(work), Dependents: []WorkItemReference{}}
	record.Readiness, record.Eligible = work.Readiness, work.Eligible
	if work.Execution != nil {
		record.Execution = *work.Execution
	}
	for _, other := range result.WorkItems {
		if other.ID == nil || *other.ID == *work.ID {
			continue
		}
		for _, edge := range other.Dependencies {
			if edge.To.ID != nil && *edge.To.ID == *work.ID {
				title := *other.ID
				if other.Title != nil {
					title = *other.Title
				}
				record.Dependents = append(record.Dependents, WorkItemReference{ID: *other.ID, Title: title})
				break
			}
		}
	}
	record.Stopped, err = workItemHeld(ctx, q, project.ID, *work.ID)
	if err != nil {
		return record, err
	}
	if err = requireNoTaskActivity(ctx, q, project.ID, *work.ID); err != nil {
		if ErrorCodeOf(err) == CodeInternal {
			return record, err
		}
		record.Frozen = true
		record.FrozenReason = err.Error()
	}
	if err = requireNoTaskPublication(ctx, q, project.ID); err != nil {
		record.PublicationPending = true
		record.Frozen = true
		record.FrozenReason = err.Error()
	}
	record.DeleteReason = s.draftDeleteReason(ctx, q, project, result, work, record)
	record.CanDelete = record.DeleteReason == ""
	record.Actions = workItemActions(record)
	return record, nil
}
func workItemActions(record WorkItemRecord) []string {
	actions := []string{}
	if record.PublicationPending {
		return []string{"reconcile"}
	}
	if record.Frozen || record.Deleted {
		return actions
	}
	if record.Archived {
		return []string{"restore"}
	}
	actions = append(actions, "edit", "archive")
	if record.Queued && !record.Stopped {
		actions = append(actions, "unqueue")
		if record.Eligible {
			actions = append(actions, "start")
		}
	} else {
		actions = append(actions, "queue")
	}
	if record.CanDelete {
		actions = append(actions, "delete")
	}
	return actions
}

func requireNoTaskActivity(ctx context.Context, q queryer, projectID ProjectID, workItemID string) error {
	if err := requireNoActiveJob(ctx, q, projectID, workItemID); err != nil {
		return err
	}
	var count int
	err := q.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM refinements WHERE project_id=? AND work_item_id=? AND status='running') + (SELECT count(*) FROM expectations WHERE project_id=? AND work_item_id=? AND status='running') + (SELECT count(*) FROM attempts a JOIN jobs j ON j.id=a.job_id WHERE j.project_id=? AND j.work_item_id=? AND a.status IN ('pending','assigned','start_registered','running','cancel_acknowledged','unknown'))`, projectID, workItemID, projectID, workItemID, projectID, workItemID).Scan(&count)
	if err != nil {
		return fault(CodeInternal, "inspect active task work", err)
	}
	if count > 0 {
		return fault(CodeConflict, "active or uncertain refinement, tests, or execution must stop before this work item can change", nil)
	}
	return nil
}

// requireTaskQueued is rechecked under the mutation lock at each dispatch. It
// deliberately uses authored commitments rather than a second queue database.
func (s *Service) requireTaskQueued(ctx context.Context, q queryer, project Project, workItemID string) error {
	if err := requireNoTaskPublication(ctx, q, project.ID); err != nil {
		return err
	}
	result, err := s.orientProject(ctx, project)
	if err != nil {
		return err
	}
	work, err := selectWork(result, workItemID)
	if err != nil {
		return err
	}
	if workArchived(work) {
		return fault(CodeNotReady, "restore and queue the archived work item before starting", nil)
	}
	if !workQueued(work) {
		return fault(CodeNotReady, "queue the draft work item before starting", nil)
	}
	held, err := workItemHeld(ctx, q, project.ID, workItemID)
	if err != nil {
		return err
	}
	if held {
		return fault(CodeConflict, "the work item was explicitly stopped; queue it again before starting", nil)
	}
	return nil
}

func (s *Service) SetWorkItemDisposition(ctx context.Context, actor Principal, request SetWorkItemDispositionRequest) (WorkItemRecord, error) {
	switch request.Action {
	case "queue", "unqueue", "archive", "restore", "delete":
	default:
		return WorkItemRecord{}, fault(CodeInvalid, "action must be queue, unqueue, archive, restore, or delete", nil)
	}
	return s.writeWorkItem(ctx, actor, request.ProjectID, "work_item_disposition", request.RequestID, request, func(tx *sql.Tx, current Principal, project Project) (workItemPublication, error) {
		result, err := s.orientProject(ctx, project)
		if err != nil {
			return workItemPublication{}, err
		}
		if !result.InventoryComplete || result.Project == nil || !result.Project.CommitmentsKnown {
			return workItemPublication{}, fault(CodeNotReady, "the complete project and commitment inventory is required to change task disposition", nil)
		}
		work, err := selectWork(result, request.WorkItemID)
		if err != nil {
			return workItemPublication{}, err
		}
		record, err := s.workItemRecord(ctx, tx, project, result, work)
		if err != nil {
			return workItemPublication{}, err
		}
		if record.TicketSHA256 != request.IfTicketSHA256 {
			return workItemPublication{}, fault(CodeStale, "the work item changed since you opened it; reload before changing its disposition", nil)
		}
		if err = requireNoTaskActivity(ctx, tx, project.ID, request.WorkItemID); err != nil {
			return workItemPublication{}, err
		}
		raw, err := os.ReadFile(work.Source)
		if err != nil {
			return workItemPublication{}, err
		}
		if digestBytes(raw) != request.IfTicketSHA256 {
			return workItemPublication{}, fault(CodeStale, "the work item changed before disposition publication", nil)
		}
		updated := raw
		queued := request.Action == "queue"
		switch request.Action {
		case "queue":
			if record.Archived {
				return workItemPublication{}, fault(CodeConflict, "restore the work item before queueing it", nil)
			}
		case "archive":
			updated, err = setRecordBool(raw, "factoryArchived", true)
		case "restore":
			updated, err = setRecordBool(raw, "factoryArchived", false)
		case "delete":
			if !record.CanDelete {
				return workItemPublication{}, fault(CodeConflict, record.DeleteReason, nil)
			}
			updated = nil
		}
		if err != nil {
			return workItemPublication{}, err
		}
		manifest, err := commitmentChange(result.Project.Source, record.Draft.Title, work.Source, queued, project)
		if err != nil {
			return workItemPublication{}, err
		}
		// Withdraw the commitment before removing or archiving a record. Pending
		// publication also blocks admission, so neither intermediate state can run.
		publication := workItemPublication{ProjectID: project.ID, WorkItemID: request.WorkItemID, Files: []workFileChange{manifest, {Path: work.Source, Before: raw, After: updated}}, Result: record, ClearConsent: request.Action != "queue", ClearHold: request.Action == "queue"}
		publication.Result.Deleted = request.Action == "delete"
		if publication.Result.Deleted {
			publication.Result.Actions = []string{}
		}
		return publication, nil
	})
}

func (s *Service) draftDeleteReason(ctx context.Context, q queryer, project Project, result orientation.Result, work orientation.WorkItem, record WorkItemRecord) string {
	if record.Queued {
		return "remove the work item from the queue before deleting it"
	}
	if record.Archived {
		return "restore the work item before deleting its unused draft"
	}
	if record.Frozen {
		return "active or uncertain work prevents draft deletion"
	}
	if work.Acceptance != nil && work.Acceptance.Record != nil {
		return "this work item has recorded acceptance; archive it instead"
	}
	if work.Execution == nil || *work.Execution != "unstarted" {
		return "only never-run, unstarted drafts can be deleted"
	}
	if len(record.Dependents) > 0 {
		return "other work items depend on this draft; archive it to retain those references"
	}
	var count int
	if err := q.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM jobs WHERE project_id=? AND work_item_id=?) + (SELECT count(*) FROM refinements WHERE project_id=? AND work_item_id=?) + (SELECT count(*) FROM expectations WHERE project_id=? AND work_item_id=?)`, project.ID, record.WorkItemID, project.ID, record.WorkItemID, project.ID, record.WorkItemID).Scan(&count); err != nil {
		return "task history could not be checked"
	}
	if count > 0 {
		return "this work item has execution or proposal history; archive it instead"
	}
	if !result.InventoryComplete {
		return "the complete project inventory is required before deleting a draft"
	}
	if reason := workReferences(project, work.Source, *work.ID); reason != "" {
		return reason
	}
	return ""
}

// Refuse deletion if any authorized Markdown source refers to the draft, or if
// the bounded scan cannot establish that it is unreferenced. References inside
// code examples are not relationships. No files are changed by this inspection.
func workReferences(project Project, target, workItemID string) string {
	reader, err := recordread.NewReader(project.BundlePath, project.AllowedSourceRoots)
	if err != nil {
		return "source roots could not be checked for references"
	}
	defer reader.Close()
	targetCanonical, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "the draft source could not be resolved"
	}
	seen := map[string]bool{}
	roots := append([]string{project.BundlePath}, project.AllowedSourceRoots...)
	files := 0
	var total int64
	maxFiles := project.ReaderLimits.MaxFiles
	if maxFiles <= 0 {
		maxFiles = recordread.DefaultMaxFiles
	}
	maxBytes := project.ReaderLimits.MaxBytes
	if maxBytes <= 0 {
		maxBytes = recordread.DefaultMaxBytes
	}
	reason := ""
	stop := errors.New("stop reference scan")
	for _, root := range roots {
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				reason = "some source paths could not be checked for references"
				return stop
			}
			if entry.IsDir() {
				if path != root && (entry.Name() == ".git" || entry.Name() == ".context-cache" || entry.Name() == "node_modules") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			canonical, e := filepath.EvalSymlinks(path)
			if e != nil {
				reason = "a Markdown source could not be resolved"
				return stop
			}
			if seen[canonical] || canonical == targetCanonical {
				return nil
			}
			seen[canonical] = true
			files++
			info, e := entry.Info()
			if e != nil {
				reason = "a Markdown source could not be inspected"
				return stop
			}
			total += info.Size()
			if files > maxFiles || total > maxBytes {
				reason = "reference scan exceeds the project's reader limits; archive the draft or increase the explicit limits"
				return stop
			}
			source, diagnostic := reader.Read(path, recordread.DocumentSource)
			if diagnostic != nil {
				reason = "a Markdown source could not be checked for references"
				return stop
			}
			if document, parseErr := recordread.ParseDocument([]byte(source.Text)); parseErr == nil {
				if referenced, ok := document.Metadata["workItemId"].(string); ok && referenced == workItemID {
					reason = "the draft is referenced by " + relativeRecord(project.BundlePath, source.Path) + "; archive it instead"
					return stop
				}
			}
			for _, link := range markdownTargets([]byte(source.Text)) {
				linked, diagnostic := reader.LinkPath(source.Path, recordread.Relationship{Destination: link, Link: link})
				if diagnostic != nil {
					continue
				}
				resolved, e := filepath.EvalSymlinks(linked)
				if e == nil && resolved == targetCanonical {
					reason = "the draft is referenced by " + relativeRecord(project.BundlePath, source.Path) + "; archive it instead"
					return stop
				}
			}
			return nil
		})
		if err != nil {
			return reason
		}
	}
	return ""
}
