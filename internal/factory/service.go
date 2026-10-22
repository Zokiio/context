package factory

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Zokiio/context/internal/decision"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const defaultLeaseDuration = 30 * time.Second

type serviceState struct {
	db            *sql.DB
	lock          *os.File
	dataDir       string
	workspaceRoot string
	evidenceRoot  string
	gitBinary     string
	leaseDuration time.Duration
	maxFiles      int
	maxBytes      int64
	now           func() time.Time
	mutations     sync.Mutex
	closeOnce     sync.Once
	closeErr      error

	decision            *decision.Client
	decisionThreshold   float64
	decisionMu          sync.Mutex
	decisionModelDigest string
}

func Open(dataDir string, options Options) (*Service, error) {
	if dataDir == "" {
		return nil, fault(CodeInvalid, "data directory is required", nil)
	}
	absolute, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fault(CodeInvalid, "resolve data directory", err)
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fault(CodeInternal, "create data directory", err)
	}
	if err := os.Chmod(absolute, 0o700); err != nil {
		return nil, fault(CodeInternal, "protect data directory", err)
	}
	lock, err := os.OpenFile(filepath.Join(absolute, "controller.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fault(CodeInternal, "open controller lock", err)
	}
	if err := lockFile(lock, true); err != nil {
		_ = lock.Close()
		if isLockContended(err) {
			return nil, fault(CodeLocked, "another controller owns this data directory", err)
		}
		return nil, fault(CodeInternal, "lock controller data", err)
	}
	cleanupLock := true
	defer func() {
		if cleanupLock {
			_ = unlockFile(lock)
			_ = lock.Close()
		}
	}()
	workspaceRoot := filepath.Join(absolute, "workspaces")
	evidenceRoot := filepath.Join(absolute, "evidence")
	for _, path := range []string{workspaceRoot, evidenceRoot} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return nil, fault(CodeInternal, "create private factory directory", err)
		}
	}
	dbPath := filepath.Join(absolute, "factory.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fault(CodeInternal, "open factory database", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	closeDB := true
	defer func() {
		if closeDB {
			_ = db.Close()
		}
	}()
	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = FULL",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fault(CodeInternal, "configure factory database", err)
		}
	}
	if err := migrate(db); err != nil {
		return nil, fault(CodeInternal, "migrate factory database", err)
	}
	lease := options.LeaseDuration
	if lease <= 0 {
		lease = defaultLeaseDuration
	}
	gitBinary := options.GitBinary
	if gitBinary == "" {
		gitBinary = "git"
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	state := &serviceState{
		db: db, lock: lock, dataDir: absolute, workspaceRoot: workspaceRoot,
		evidenceRoot: evidenceRoot, gitBinary: gitBinary, leaseDuration: lease,
		maxFiles: options.MaxContextFiles, maxBytes: options.MaxContextBytes, now: now,
	}
	if options.DecisionModel != "" {
		state.decision = decision.New(options.DecisionURL, options.DecisionModel)
		state.decisionThreshold = options.DecisionThreshold
		if state.decisionThreshold <= 0 {
			state.decisionThreshold = DefaultDecisionThreshold
		}
	}
	if err := state.recoverInterruptedAttempts(context.Background()); err != nil {
		return nil, err
	}
	cleanupLock, closeDB = false, false
	return &Service{state: state}, nil
}

func (s *Service) Close() error {
	if s == nil || s.state == nil {
		return nil
	}
	s.state.closeOnce.Do(func() {
		dbErr := s.state.db.Close()
		unlockErr := unlockFile(s.state.lock)
		lockErr := s.state.lock.Close()
		s.state.closeErr = errors.Join(dbErr, unlockErr, lockErr)
	})
	return s.state.closeErr
}

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  kind TEXT NOT NULL,
  installation_admin INTEGER NOT NULL,
  disabled INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tokens (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  expires_at TEXT,
  revoked_at TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS executor_profiles (
  key TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  command_json TEXT NOT NULL,
  fixture INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS check_profiles (
  key TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  command_json TEXT NOT NULL,
  environment TEXT NOT NULL,
  fixture INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  bundle_path TEXT NOT NULL UNIQUE,
  repository_path TEXT NOT NULL,
  allowed_sources_json TEXT NOT NULL,
  reader_max_files INTEGER NOT NULL,
  reader_max_bytes INTEGER NOT NULL,
  policy_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS project_grants (
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  capability TEXT NOT NULL,
  PRIMARY KEY(project_id, user_id, capability)
);
CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id),
  work_item_id TEXT NOT NULL,
  coordinator_id TEXT NOT NULL REFERENCES users(id),
  implementer_id TEXT REFERENCES users(id),
  status TEXT NOT NULL,
  version INTEGER NOT NULL,
  admission_json TEXT NOT NULL,
  candidate_revision TEXT NOT NULL DEFAULT '',
  repair_count INTEGER NOT NULL DEFAULT 0,
  acceptance_record_path TEXT NOT NULL DEFAULT '',
  acceptance_status TEXT NOT NULL DEFAULT 'absent',
  active_claim INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS one_active_implementation_claim
  ON jobs(project_id, work_item_id) WHERE active_claim = 1;
CREATE TABLE IF NOT EXISTS attempts (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  stage TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  check_key TEXT NOT NULL DEFAULT '',
  dispatch_id TEXT NOT NULL UNIQUE,
  worker_id TEXT REFERENCES users(id),
  native_run_id TEXT NOT NULL DEFAULT '',
  workspace_path TEXT NOT NULL,
  input_revision TEXT NOT NULL,
  candidate_revision TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  outcome TEXT NOT NULL DEFAULT '',
  check_status TEXT NOT NULL DEFAULT '',
  lease_expires_at TEXT,
  last_heartbeat_at TEXT,
  started_at TEXT,
  finished_at TEXT,
  cancellation_requested INTEGER NOT NULL DEFAULT 0,
  summary TEXT NOT NULL DEFAULT '',
  terminal_receipt TEXT NOT NULL DEFAULT '',
  UNIQUE(job_id, stage, ordinal, check_key)
);
CREATE TABLE IF NOT EXISTS check_results (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  check_key TEXT NOT NULL,
  status TEXT NOT NULL,
  candidate_revision TEXT NOT NULL,
  requirement_fingerprint TEXT NOT NULL,
  observer_id TEXT REFERENCES users(id),
  attempt_id TEXT REFERENCES attempts(id),
  command_json TEXT NOT NULL,
  environment TEXT NOT NULL,
  observed_at TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reviews (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  reviewer_id TEXT NOT NULL REFERENCES users(id),
  candidate_revision TEXT NOT NULL,
  requirement_fingerprint TEXT NOT NULL,
  decision TEXT NOT NULL,
  summary TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS artifacts (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  attempt_id TEXT REFERENCES attempts(id),
  name TEXT NOT NULL,
  media_type TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  content TEXT NOT NULL,
  byte_count INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  actor_id TEXT REFERENCES users(id),
  kind TEXT NOT NULL,
  data_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS command_receipts (
  actor_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  request_id TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  response_json TEXT,
  secret INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  PRIMARY KEY(actor_id, operation, request_id)
);
CREATE TABLE IF NOT EXISTS acceptance_writes (
  job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
  plan_json TEXT NOT NULL,
  state TEXT NOT NULL,
  error TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS criterion_signals (
  model TEXT NOT NULL,
  question TEXT NOT NULL,
  input_sha256 TEXT NOT NULL,
  checkable REAL NOT NULL,
  model_digest TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  PRIMARY KEY(model, question, input_sha256)
);`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return ensureColumn(db, "attempts", "summary", `ALTER TABLE attempts ADD COLUMN summary TEXT NOT NULL DEFAULT ''`)
}

func ensureColumn(db *sql.DB, table, column, statement string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var ordinal int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&ordinal, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = db.Exec(statement)
	return err
}

func fault(code ErrorCode, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func detailFault(code ErrorCode, message string, details map[string]any, err error) *Error {
	return &Error{Code: code, Message: message, Details: details, Err: err}
}

func newID(prefix string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(raw), nil
}

func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "wm_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

func tokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func requestHash(operation string, request any) (string, error) {
	encoded, err := json.Marshal(struct {
		Operation string `json:"operation"`
		Request   any    `json:"request"`
	}{operation, request})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func nullableTime(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed, err := parseTime(value.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func requireRequestID(value string) error {
	if value == "" {
		return fault(CodeInvalid, "requestId is required", nil)
	}
	return nil
}

func receipt[T any](ctx context.Context, tx *sql.Tx, actor UserID, operation, requestID string, request any) (T, bool, string, error) {
	var zero T
	if err := requireRequestID(requestID); err != nil {
		return zero, false, "", err
	}
	hash, err := requestHash(operation, request)
	if err != nil {
		return zero, false, "", fault(CodeInternal, "hash request", err)
	}
	var storedHash string
	var response sql.NullString
	var secret int
	err = tx.QueryRowContext(ctx, `SELECT request_hash, response_json, secret FROM command_receipts WHERE actor_id=? AND operation=? AND request_id=?`, actor, operation, requestID).Scan(&storedHash, &response, &secret)
	switch {
	case err == nil && storedHash != hash:
		return zero, false, "", fault(CodeConflict, "requestId was already used with a different request", nil)
	case err == nil && secret != 0:
		return zero, false, "", fault(CodeConflict, "credential issue requests cannot be replayed", nil)
	case err == nil && response.Valid:
		if err := json.Unmarshal([]byte(response.String), &zero); err != nil {
			return zero, false, "", fault(CodeInternal, "decode command receipt", err)
		}
		return zero, true, hash, nil
	case err == nil:
		return zero, false, "", fault(CodeConflict, "request is already in progress", nil)
	case !errors.Is(err, sql.ErrNoRows):
		return zero, false, "", fault(CodeInternal, "read command receipt", err)
	default:
		return zero, false, hash, nil
	}
}

func saveReceipt(ctx context.Context, tx *sql.Tx, actor UserID, operation, requestID, hash string, value any, secret bool, now time.Time) error {
	var response any
	if !secret {
		encoded, err := json.Marshal(value)
		if err != nil {
			return fault(CodeInternal, "encode command receipt", err)
		}
		response = string(encoded)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO command_receipts(actor_id, operation, request_id, request_hash, response_json, secret, created_at) VALUES(?,?,?,?,?,?,?)`, actor, operation, requestID, hash, response, boolInt(secret), formatTime(now))
	if err != nil {
		return fault(CodeInternal, "save command receipt", err)
	}
	return nil
}

func marshal(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal value: %w", err)
	}
	return string(encoded), nil
}
