// Package factoryworker invokes configured external commands for immutable assignments.
// Receipts survive worker restarts. An unfinished receipt is never permission to rerun.
package factoryworker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Command is an administrator supplied argv. No argument is interpreted by a shell.
type Command struct {
	Name string   `json:"name"`
	Argv []string `json:"argv"`
}

// Assignment is the immutable input retained beside the process receipt.
// The controller, rather than this adapter, creates Workspace.
type Assignment struct {
	DispatchID     string          `json:"dispatchId"`
	Raw            json.RawMessage `json:"assignment,omitempty"`
	AttemptID      string          `json:"attemptId"`
	JobID          string          `json:"jobId"`
	WorkerID       string          `json:"workerId"`
	Role           string          `json:"role"`
	Workspace      string          `json:"workspace"`
	Base           string          `json:"base"`
	Candidate      string          `json:"candidate,omitempty"`
	Requirements   string          `json:"requirements"`
	Context        json.RawMessage `json:"context"`
	Command        Command         `json:"command"`
	Checks         []Command       `json:"checks,omitempty"`
	Mode           string          `json:"mode"`
	FixtureProject bool            `json:"fixtureProject"`
}

type Observation struct {
	Name       string    `json:"name"`
	Argv       []string  `json:"argv"`
	Status     string    `json:"status"`
	ExitCode   int       `json:"exitCode"`
	Log        string    `json:"log"`
	Truncated  bool      `json:"truncated,omitempty"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
}

type Receipt struct {
	AttemptID        string        `json:"attemptId"`
	AssignmentSHA256 string        `json:"assignmentSHA256"`
	State            string        `json:"state"`
	Candidate        string        `json:"candidate,omitempty"`
	PID              int           `json:"pid,omitempty"`
	StartedAt        time.Time     `json:"startedAt"`
	FinishedAt       time.Time     `json:"finishedAt,omitempty"`
	Observations     []Observation `json:"observations,omitempty"`
	Error            string        `json:"error,omitempty"`
}

// Process owns only processes it starts in this invocation. Stored PIDs are
// diagnostic facts and are never used to signal a process after restart.
type Process struct {
	ReceiptDir  string
	MaxLogBytes int
	StopGrace   time.Duration
	mu          sync.Mutex
	active      map[string]*running
}
type running struct{ cancel context.CancelFunc }

// Hooks records the service acknowledgement before launch and the observed start.
type Hooks struct {
	Register func(context.Context, Receipt) error
	Started  func(context.Context, int) error
}

// Execute retains intent before starting. A repeated delivery returns the saved
// outcome; an unfinished prior invocation returns unknown without starting again.
func (p *Process) Execute(ctx context.Context, a Assignment) (Receipt, error) {
	return p.execute(ctx, a, Hooks{})
}

func (p *Process) execute(ctx context.Context, a Assignment, hooks Hooks) (Receipt, error) {
	if strings.TrimSpace(p.ReceiptDir) == "" {
		return Receipt{}, errors.New("durable receipt directory is required")
	}
	if err := validate(a); err != nil {
		return Receipt{}, err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return Receipt{}, err
	}
	identity := a
	if len(a.Raw) > 0 {
		var body map[string]json.RawMessage
		if err = json.Unmarshal(a.Raw, &body); err != nil {
			return Receipt{}, err
		}
		delete(body, "leaseExpiresAt")
		identity.Raw, err = json.Marshal(body)
		if err != nil {
			return Receipt{}, err
		}
	}
	immutable, err := json.Marshal(identity)
	if err != nil {
		return Receipt{}, err
	}
	digest := sha256.Sum256(immutable)
	expected := hex.EncodeToString(digest[:])
	path := p.receiptPath(a.AttemptID)
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{AttemptID: a.AttemptID, AssignmentSHA256: expected, State: "prepared", StartedAt: time.Now().UTC()}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		old, e := p.Read(a.AttemptID)
		if e != nil {
			observed := receipt.StartedAt
			if info, statErr := os.Stat(path); statErr == nil {
				observed = info.ModTime().UTC()
			}
			return Receipt{AttemptID: a.AttemptID, AssignmentSHA256: expected, State: "unknown", StartedAt: observed, FinishedAt: observed, Error: "existing receipt is unreadable; reconciliation required"}, nil
		}
		if old.AssignmentSHA256 != expected {
			return Receipt{}, errors.New("attempt receipt belongs to different assignment")
		}
		if !terminal(old.State) {
			old.State = "unknown"
			old.Error = "prior execution has no terminal receipt; reconciliation required"
			old.FinishedAt = time.Now().UTC()
			if e = p.save(path, old); e != nil {
				return old, e
			}
		}
		return old, nil
	}
	if err != nil {
		return Receipt{}, err
	}
	if err = writeSync(f, receipt); err != nil {
		f.Close()
		return Receipt{}, err
	}
	if err = f.Close(); err != nil {
		return Receipt{}, err
	}
	if err = syncDir(filepath.Dir(path)); err != nil {
		return Receipt{}, err
	}
	assignmentPath := filepath.Join(filepath.Dir(path), "assignment.json")
	contextPath := filepath.Join(filepath.Dir(path), "context.json")
	if err = writeExclusive(assignmentPath, data, 0400); err != nil {
		return receipt, err
	}
	if err = writeExclusive(contextPath, a.Context, 0400); err != nil {
		return receipt, err
	}
	if hooks.Register != nil {
		if err = hooks.Register(ctx, receipt); err != nil {
			return receipt, err
		}
	}
	input := data
	if len(a.Raw) > 0 {
		input = a.Raw
	}
	runctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p.mu.Lock()
	if p.active == nil {
		p.active = map[string]*running{}
	}
	p.active[a.AttemptID] = &running{cancel: cancel}
	p.mu.Unlock()
	defer func() { p.mu.Lock(); delete(p.active, a.AttemptID); p.mu.Unlock() }()
	if a.Role == "verifier" {
		if err = verifyCandidate(a); err != nil {
			receipt.State = "failed"
			receipt.Error = err.Error()
		} else {
			receipt.State = "succeeded"
			receipt.Candidate = a.Candidate
			for _, check := range a.Checks {
				o, pid, e := p.invoke(runctx, a, check, input, assignmentPath, contextPath, hooks)
				receipt.PID = pid
				receipt.Observations = append(receipt.Observations, o)
				if e != nil {
					receipt.State = "unknown"
					receipt.Error = e.Error()
					break
				}
				if o.Status != "passed" {
					if o.Status == "cancelled" || o.Status == "unknown" {
						receipt.State = o.Status
						break
					}
					receipt.State = "failed"
				}
				if e = verifyCandidate(a); e != nil {
					receipt.State = "failed"
					receipt.Error = e.Error()
					break
				}
			}
		}
	} else {
		o, pid, e := p.invoke(runctx, a, a.Command, input, assignmentPath, contextPath, hooks)
		receipt.PID = pid
		receipt.Observations = append(receipt.Observations, o)
		switch {
		case e != nil:
			receipt.State = "unknown"
			receipt.Error = e.Error()
		case o.Status == "passed":
			receipt.State = "succeeded"
		case o.Status == "cancelled":
			receipt.State = "cancelled"
		default:
			receipt.State = "failed"
		}
		if receipt.State == "succeeded" {
			receipt.Candidate, err = git(a.Workspace, "rev-parse", "HEAD")
			if err == nil {
				var status string
				status, err = git(a.Workspace, "status", "--porcelain")
				if err == nil && status != "" {
					err = errors.New("worker left uncommitted changes")
				}
			}
			if err == nil && receipt.Candidate == a.Base {
				err = errors.New("worker produced no candidate commit")
			}
			if err != nil {
				receipt.State = "failed"
				receipt.Error = err.Error()
			}
		}
	}
	receipt.FinishedAt = time.Now().UTC()
	if err = p.save(path, receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}

// Cancel signals only the process group still owned by this Process object.
// It requests cancellation; the eventual receipt establishes the outcome.
func (p *Process) Cancel(attemptID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	r := p.active[attemptID]
	if r == nil {
		return false
	}
	r.cancel()
	return true
}
func (p *Process) Read(id string) (Receipt, error) {
	var r Receipt
	b, e := os.ReadFile(p.receiptPath(id))
	if e != nil {
		return r, e
	}
	e = json.Unmarshal(b, &r)
	return r, e
}

// progressPath is where a running command may append progress notes. The worker
// forwards the latest line with each heartbeat.
func (p *Process) progressPath(id string) string {
	return filepath.Join(filepath.Dir(p.receiptPath(id)), "progress.txt")
}

// resultPath is where an adapter may write its own observations, such as an agent's
// cost. The worker attaches the file to the attempt as agent-result.json.
func (p *Process) resultPath(id string) string {
	return filepath.Join(filepath.Dir(p.receiptPath(id)), "result.json")
}
func (p *Process) receiptPath(id string) string {
	h := sha256.Sum256([]byte(id))
	return filepath.Join(p.ReceiptDir, hex.EncodeToString(h[:]), "receipt.json")
}
func (p *Process) save(path string, r Receipt) error {
	f, e := os.CreateTemp(filepath.Dir(path), "receipt-*.tmp")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e == nil {
		e = writeSync(f, r)
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e == nil {
		e = os.Rename(name, path)
	}
	if e == nil {
		e = syncDir(filepath.Dir(path))
	}
	return e
}

func (p *Process) invoke(ctx context.Context, a Assignment, c Command, input []byte, assignmentPath, contextPath string, hooks Hooks) (Observation, int, error) {
	o := Observation{Name: c.Name, Argv: append([]string(nil), c.Argv...), ExitCode: -1, StartedAt: time.Now().UTC()}
	if len(c.Argv) == 0 {
		o.Status = "unavailable"
		o.Log = "command has no executable"
		o.FinishedAt = time.Now().UTC()
		return o, 0, nil
	}
	if ctx.Err() != nil {
		o.Status = "cancelled"
		o.Log = "cancelled before process start"
		o.FinishedAt = time.Now().UTC()
		return o, 0, nil
	}
	limit := p.MaxLogBytes
	if limit <= 0 {
		limit = 64 << 10
	}
	logFile, logErr := os.OpenFile(filepath.Join(filepath.Dir(assignmentPath), fmt.Sprintf("command-%d.log", o.StartedAt.UnixNano())), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if logErr != nil {
		o.Status = "unavailable"
		o.Log = logErr.Error()
		o.FinishedAt = time.Now().UTC()
		return o, 0, nil
	}
	defer logFile.Close()
	log := &boundedLog{limit: limit, file: logFile}
	cmd := exec.Command(c.Argv[0], c.Argv[1:]...)
	cmd.Dir = a.Workspace
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.Env = append(os.Environ(), "CTX_FACTORY_ASSIGNMENT="+assignmentPath, "CTX_FACTORY_CONTEXT="+contextPath, "CTX_FACTORY_ATTEMPT="+a.AttemptID, "CTX_FACTORY_PROGRESS="+p.progressPath(a.AttemptID), "CTX_FACTORY_RESULT="+p.resultPath(a.AttemptID))
	if err := configureProcess(cmd); err != nil {
		o.Status = "unavailable"
		o.Log = err.Error()
		o.FinishedAt = time.Now().UTC()
		return o, 0, nil
	}
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Start(); err != nil {
		o.Status = "unavailable"
		o.Log = err.Error()
		o.FinishedAt = time.Now().UTC()
		return o, 0, nil
	}
	pid := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	if hooks.Started != nil {
		_ = hooks.Started(ctx, pid)
	}
	cancelled := false
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		cancelled = true
		_ = stopOwnedProcess(cmd, false)
		grace := p.StopGrace
		if grace <= 0 {
			grace = 2 * time.Second
		}
		timer := time.NewTimer(grace)
		select {
		case err = <-done:
			timer.Stop()
		case <-timer.C:
			_ = stopOwnedProcess(cmd, true)
			finalWait := time.NewTimer(grace + cmd.WaitDelay)
			select {
			case err = <-done:
				finalWait.Stop()
			case <-finalWait.C:
				o.Status = "unknown"
				o.FinishedAt = time.Now().UTC()
				o.Log, o.Truncated = log.result()
				return o, pid, errors.New("no terminal observation after signalling owned process group; stoppage unproven")
			}
		}
	}
	o.FinishedAt = time.Now().UTC()
	o.Log, o.Truncated = log.result()
	o.ExitCode = cmd.ProcessState.ExitCode()
	// Descendants which outlive the observed child make the outcome uncertain.
	if ownedGroupAlive(cmd) || errors.Is(err, exec.ErrWaitDelay) {
		o.Status = "unknown"
		return o, pid, errors.New("spawned process group still exists; stoppage unproven")
	}
	switch {
	case cancelled:
		if a.Mode != "fixture" {
			o.Status = "unknown"
			return o, pid, errors.New("spawned process group stopped; external native execution cannot be reconciled by the generic command adapter")
		}
		o.Status = "cancelled"
	case err != nil:
		o.Status = "failed"
	default:
		o.Status = "passed"
	}
	return o, pid, nil
}
func validate(a Assignment) error {
	if a.AttemptID == "" || a.JobID == "" || a.WorkerID == "" || a.Workspace == "" {
		return errors.New("assignment requires attempt, job, worker and workspace")
	}
	if a.Role != "implementer" && a.Role != "verifier" {
		return errors.New("unknown worker role")
	}
	if !filepath.IsAbs(a.Workspace) {
		return errors.New("workspace must be absolute")
	}
	if a.Mode == "fixture" && !a.FixtureProject {
		return errors.New("fixture executor requires a fixture project")
	}
	if a.Mode != "fixture" && a.Mode != "command" {
		return errors.New("unknown executor mode")
	}
	if a.Role == "verifier" && (a.Candidate == "" || len(a.Checks) == 0) {
		return errors.New("verifier assignment requires candidate and checks")
	}
	if a.Role == "implementer" && len(a.Command.Argv) == 0 {
		return errors.New("implementation command is empty")
	}
	return nil
}
func terminal(s string) bool {
	return s == "succeeded" || s == "failed" || s == "cancelled" || s == "unknown"
}
func verifyCandidate(a Assignment) error {
	head, e := git(a.Workspace, "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	if head != a.Candidate {
		return errors.New("verifier workspace does not match sealed candidate")
	}
	status, e := git(a.Workspace, "status", "--porcelain")
	if e != nil {
		return e
	}
	if status != "" {
		return errors.New("verifier workspace is not clean")
	}
	return nil
}
func git(dir string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	b, e := c.CombinedOutput()
	if e != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), e, b)
	}
	return strings.TrimSpace(string(b)), nil
}
func writeSync(w *os.File, v any) error {
	if e := json.NewEncoder(w).Encode(v); e != nil {
		return e
	}
	return w.Sync()
}
func writeExclusive(path string, data []byte, mode os.FileMode) error {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if e != nil {
		return e
	}
	_, e = f.Write(data)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	return e
}
func syncDir(path string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}

type boundedLog struct {
	file      *os.File
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedLog) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	left := b.limit - b.buf.Len()
	if left < len(p) {
		b.truncated = true
		if left < 0 {
			left = 0
		}
		p = p[:left]
	}
	_, _ = b.buf.Write(p)
	if b.file != nil {
		_, _ = b.file.Write(p)
		_ = b.file.Sync()
	}
	return n, nil
}
func (b *boundedLog) result() (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String(), b.truncated
}

var _ io.Writer = (*boundedLog)(nil)
