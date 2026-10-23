package factory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// A publication is retained before any authored file changes. Replaying its
// request accepts only the exact before or after bytes of each destination.
// Until it finishes, the project cannot dispatch or publish another task edit.
type workFileChange struct {
	Path   string `json:"path"`
	Before []byte `json:"before"`
	After  []byte `json:"after"`
}
type workItemPublication struct {
	ProjectID    ProjectID        `json:"projectId"`
	WorkItemID   string           `json:"workItemId"`
	Files        []workFileChange `json:"files"`
	Result       WorkItemRecord   `json:"result"`
	ClearConsent bool             `json:"clearConsent"`
	ClearHold    bool             `json:"clearHold"`
}

func (s *Service) writeWorkItem(ctx context.Context, actor Principal, projectID ProjectID, operation, requestID string, request any, prepare func(*sql.Tx, Principal, Project) (workItemPublication, error)) (WorkItemRecord, error) {
	if err := requireRequestID(requestID); err != nil {
		return WorkItemRecord{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "start task publication", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, projectID, CapabilityCoordinate)
	if err != nil {
		return WorkItemRecord{}, err
	}
	saved, ok, hash, err := receipt[WorkItemRecord](ctx, tx, current.UserID, operation, requestID, request)
	if err != nil || ok {
		return saved, err
	}
	var publication workItemPublication
	var storedHash, encoded string
	err = tx.QueryRowContext(ctx, `SELECT request_hash,plan_json FROM work_item_writes WHERE actor_id=? AND operation=? AND request_id=?`, current.UserID, operation, requestID).Scan(&storedHash, &encoded)
	if err == nil {
		if storedHash != hash {
			return WorkItemRecord{}, fault(CodeConflict, "requestId was already used for a different task publication", nil)
		}
		if err = json.Unmarshal([]byte(encoded), &publication); err != nil {
			return WorkItemRecord{}, fault(CodeInternal, "read retained task publication", err)
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if err = requireNoTaskPublication(ctx, tx, projectID); err != nil {
			return WorkItemRecord{}, err
		}
		project, readErr := readProject(ctx, tx, projectID)
		if readErr != nil {
			return WorkItemRecord{}, readErr
		}
		publication, err = prepare(tx, current, project)
		if err != nil {
			return WorkItemRecord{}, err
		}
		encoded, err = marshal(publication)
		if err != nil {
			return WorkItemRecord{}, fault(CodeInternal, "encode task publication", err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO work_item_writes(actor_id,operation,request_id,project_id,request_hash,plan_json,state) VALUES(?,?,?,?,?,?,?)`, current.UserID, operation, requestID, projectID, hash, encoded, "prepared"); err != nil {
			return WorkItemRecord{}, fault(CodeInternal, "retain task publication", err)
		}
	} else {
		return WorkItemRecord{}, fault(CodeInternal, "read task publication", err)
	}
	if err = tx.Commit(); err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "retain task publication intent", err)
	}
	return s.finishWorkItemPublication(ctx, current, current.UserID, operation, requestID, hash, publication)
}

// finishWorkItemPublication runs with the controller mutation lock held. A
// coordinator may reconcile another actor's retained operation; the original
// receipt and authored attribution remain attached to the original actor.
func (s *Service) finishWorkItemPublication(ctx context.Context, current Principal, receiptActor UserID, operation, requestID, hash string, publication workItemPublication) (WorkItemRecord, error) {
	projectID := publication.ProjectID
	if _, err := authorizeProject(ctx, s.state.db, current, projectID, CapabilityCoordinate); err != nil {
		return WorkItemRecord{}, err
	}
	var err error
	if err = publishWorkFiles(publication.Files); err != nil {
		return WorkItemRecord{}, detailFault(CodeStale, "task publication is incomplete; reconcile the pending save after resolving the changed file", map[string]any{"requestId": requestID, "workItemId": publication.WorkItemID}, err)
	}
	record := publication.Result
	if !record.Deleted {
		project, readErr := readProject(ctx, s.state.db, projectID)
		if readErr != nil {
			return record, readErr
		}
		result, readErr := s.orientProject(ctx, project)
		if readErr != nil {
			return record, readErr
		}
		work, readErr := selectWork(result, publication.WorkItemID)
		if readErr != nil {
			return record, readErr
		}
		record, err = s.workItemRecord(ctx, s.state.db, project, result, work)
		if err != nil {
			return record, err
		}
		// This publication itself is the only pending writer. The returned actions
		// describe the state once its final transaction commits.
		record.PublicationPending = false
		record.Frozen = false
		record.FrozenReason = ""
		if activityErr := requireNoTaskActivity(ctx, s.state.db, projectID, publication.WorkItemID); activityErr != nil {
			record.Frozen = true
			record.FrozenReason = activityErr.Error()
		}
		if publication.ClearHold {
			record.Stopped = false
		}
		record.DeleteReason = s.draftDeleteReason(ctx, s.state.db, project, result, work, record)
		record.CanDelete = record.DeleteReason == ""
		record.Actions = workItemActions(record)
	}
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return record, fault(CodeInternal, "finish task publication", err)
	}
	defer tx.Rollback()
	if _, err = authorizeProject(ctx, tx, current, projectID, CapabilityCoordinate); err != nil {
		return record, err
	}
	if publication.ClearConsent {
		if err = dropConsent(ctx, tx, projectID, publication.WorkItemID); err != nil {
			return record, err
		}
	}
	if publication.ClearHold {
		if _, err = tx.ExecContext(ctx, `DELETE FROM work_item_holds WHERE project_id=? AND work_item_id=?`, projectID, publication.WorkItemID); err != nil {
			return record, fault(CodeInternal, "clear task stop hold", err)
		}
	}
	if err = saveReceipt(ctx, tx, receiptActor, operation, requestID, hash, record, false, s.state.now().UTC()); err != nil {
		return record, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE work_item_writes SET state='complete' WHERE actor_id=? AND operation=? AND request_id=?`, receiptActor, operation, requestID); err != nil {
		return record, fault(CodeInternal, "finish task publication", err)
	}
	if err = tx.Commit(); err != nil {
		return record, fault(CodeInternal, "commit task publication", err)
	}
	return record, nil
}

func requireNoTaskPublication(ctx context.Context, q queryer, projectID ProjectID) error {
	var requestID string
	err := q.QueryRowContext(ctx, `SELECT request_id FROM work_item_writes WHERE project_id=? AND state='prepared' LIMIT 1`, projectID).Scan(&requestID)
	if err == nil {
		return detailFault(CodeConflict, "a task publication must be reconciled before more work can change or start", map[string]any{"requestId": requestID}, nil)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fault(CodeInternal, "inspect task publication", err)
	}
	return nil
}

func publishWorkFiles(changes []workFileChange) error {
	// Check every destination before the first effect. A later external edit still
	// fails its CAS; the retained plan makes that partial outcome recoverable.
	for _, change := range changes {
		if _, err := workFileState(change); err != nil {
			return err
		}
	}
	for _, change := range changes {
		done, err := workFileState(change)
		if err != nil {
			return err
		}
		if done {
			continue
		}
		switch {
		case change.Before == nil:
			if err = publishNewWorkFile(change.Path, change.After); err != nil {
				return err
			}
		case change.After == nil:
			if err = deleteWorkFileCAS(change.Path, change.Before); err != nil {
				return err
			}
		default:
			if err = writeFileCAS(change.Path, change.Before, change.After); err != nil {
				return err
			}
		}
		directory, err := os.Open(filepath.Dir(change.Path))
		if err != nil {
			return fault(CodeInternal, "open task directory for sync", err)
		}
		err = directory.Sync()
		_ = directory.Close()
		if err != nil {
			return fault(CodeInternal, "sync task publication", err)
		}
	}
	return nil
}
func workFileState(change workFileChange) (bool, error) {
	info, err := os.Lstat(change.Path)
	if errors.Is(err, os.ErrNotExist) {
		if change.After == nil {
			return true, nil
		}
		if change.Before == nil {
			return false, nil
		}
		return false, fault(CodeStale, "a task publication destination disappeared: "+change.Path, err)
	}
	if err != nil {
		return false, fault(CodeInternal, "inspect task publication destination", err)
	}
	if !info.Mode().IsRegular() {
		return false, fault(CodeInvalid, "task publication destination is not a regular file: "+change.Path, nil)
	}
	current, err := os.ReadFile(change.Path)
	if err != nil {
		return false, fault(CodeInternal, "read task publication destination", err)
	}
	if change.After != nil && bytes.Equal(current, change.After) {
		return true, nil
	}
	if change.Before != nil && bytes.Equal(current, change.Before) {
		return false, nil
	}
	return false, fault(CodeStale, "task publication destination changed: "+change.Path, nil)
}
func deleteWorkFileCAS(path string, expected []byte) error {
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fault(CodeInternal, "open task deletion lock", err)
	}
	defer lock.Close()
	if err = lockFile(lock, false); err != nil {
		return err
	}
	defer unlockFile(lock)
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(current, expected) {
		return fault(CodeStale, "task changed before deletion", err)
	}
	if err = os.Remove(path); err != nil {
		return fault(CodeInternal, "delete draft task", err)
	}
	return nil
}

// Link a synced temporary file into its final name without replacing an existing
// destination. A crash cannot expose a partially written authoritative record.
func publishNewWorkFile(path string, content []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".waymark-task-*")
	if err != nil {
		return fault(CodeInternal, "prepare new task file", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err = file.Chmod(0o644); err != nil {
		return err
	}
	if _, err = file.Write(content); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Link(file.Name(), path); err != nil {
		return fault(CodeConflict, "publish new task without replacing an existing file", err)
	}
	return nil
}

// ReconcileWorkItemPublications lets an authorized coordinator finish retained
// exact-byte saves after a reconnect. It is safe to repeat and does not require
// the browser to have retained the original request ID or payload.
func (s *Service) ReconcileWorkItemPublications(ctx context.Context, actor Principal, projectID ProjectID) ([]WorkItemRecord, error) {
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	current, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityCoordinate)
	if err != nil {
		return nil, err
	}
	pending, err := s.pendingWorkItemPublications(ctx, projectID)
	if err != nil {
		return nil, err
	}
	results := []WorkItemRecord{}
	for _, entry := range pending {
		record, err := s.finishWorkItemPublication(ctx, current, entry.actor, entry.operation, entry.requestID, entry.hash, entry.publication)
		if err != nil {
			return results, err
		}
		results = append(results, record)
	}
	return results, nil
}

type pendingWorkItemPublication struct {
	actor                      UserID
	operation, requestID, hash string
	publication                workItemPublication
}

func (s *Service) pendingWorkItemPublications(ctx context.Context, projectID ProjectID) ([]pendingWorkItemPublication, error) {
	rows, err := s.state.db.QueryContext(ctx, `SELECT actor_id,operation,request_id,request_hash,plan_json FROM work_item_writes WHERE state='prepared' AND (?='' OR project_id=?) ORDER BY rowid`, projectID, projectID)
	if err != nil {
		return nil, fault(CodeInternal, "read pending task saves", err)
	}
	defer rows.Close()
	pending := []pendingWorkItemPublication{}
	for rows.Next() {
		var entry pendingWorkItemPublication
		var encoded string
		if err := rows.Scan(&entry.actor, &entry.operation, &entry.requestID, &entry.hash, &encoded); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(encoded), &entry.publication); err != nil {
			return nil, fault(CodeInternal, "decode pending task save", err)
		}
		pending = append(pending, entry)
	}
	return pending, rows.Err()
}
func (s *Service) recoverWorkItemPublications(ctx context.Context) {
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	pending, err := s.pendingWorkItemPublications(ctx, "")
	if err != nil {
		return
	}
	for _, entry := range pending {
		// Divergent destinations or revoked authority remain explicitly pending.
		// Startup never substitutes new bytes or chooses an external edit's winner.
		_, _ = s.finishWorkItemPublication(ctx, Principal{UserID: entry.actor}, entry.actor, entry.operation, entry.requestID, entry.hash, entry.publication)
	}
}
