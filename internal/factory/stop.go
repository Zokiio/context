package factory

import (
	"context"
	"database/sql"
	"time"
)

// holdWorkItem persists a person's stop independently of a job's eventual outcome.
// Call it within the same transaction as cancellation so a restart cannot lose the
// hold or turn a late successful observation into permission to run more work.
func holdWorkItem(ctx context.Context, tx *sql.Tx, projectID ProjectID, workItemID string, now time.Time) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO work_item_holds(project_id,work_item_id,created_at) VALUES(?,?,?) ON CONFLICT(project_id,work_item_id) DO NOTHING`, projectID, workItemID, formatTime(now)); err != nil {
		return fault(CodeInternal, "hold stopped work item", err)
	}
	return dropConsent(ctx, tx, projectID, workItemID)
}

func workItemHeld(ctx context.Context, q queryer, projectID ProjectID, workItemID string) (bool, error) {
	var held bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM work_item_holds WHERE project_id=? AND work_item_id=?)`, projectID, workItemID).Scan(&held); err != nil {
		return false, fault(CodeInternal, "read work item stop hold", err)
	}
	return held, nil
}
