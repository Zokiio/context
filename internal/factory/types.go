// Package factory owns durable workflow state for one Waymark installation.
package factory

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Zokiio/context/internal/orientation"
)

type ID string
type UserID ID
type ProjectID ID
type JobID ID
type AttemptID ID
type ArtifactID ID

type UserKind string

const (
	UserKindHuman  UserKind = "human"
	UserKindWorker UserKind = "worker"
)

type Capability string

const (
	CapabilityRead            Capability = "read"
	CapabilityCoordinate      Capability = "coordinate"
	CapabilityVerify          Capability = "verify"
	CapabilityReview          Capability = "review"
	CapabilityAccept          Capability = "accept"
	CapabilityProjectAdmin    Capability = "project_admin"
	CapabilityImplementWorker Capability = "worker_implement"
	CapabilityVerifyWorker    Capability = "worker_verify"
)

type Stage string

const (
	StageImplementation Stage = "implementation"
	StageVerification   Stage = "verification"
)

type JobStatus string

const (
	JobQueued                JobStatus = "queued"
	JobImplementing          JobStatus = "implementing"
	JobImplementationUnknown JobStatus = "implementation_unknown"
	JobVerifying             JobStatus = "verifying"
	JobVerificationFailed    JobStatus = "verification_failed"
	JobReadyForReview        JobStatus = "ready_for_review"
	JobReadyForAcceptance    JobStatus = "ready_for_acceptance"
	JobAccepting             JobStatus = "accepting"
	JobAccepted              JobStatus = "accepted"
	JobCancelRequested       JobStatus = "cancel_requested"
	JobCancellationUnknown   JobStatus = "cancellation_unknown"
	JobCancelled             JobStatus = "cancelled"
	JobFailed                JobStatus = "failed"
	JobStale                 JobStatus = "stale"
)

type AttemptStatus string

const (
	AttemptPending            AttemptStatus = "pending"
	AttemptAssigned           AttemptStatus = "assigned"
	AttemptStartRegistered    AttemptStatus = "start_registered"
	AttemptRunning            AttemptStatus = "running"
	AttemptCancelAcknowledged AttemptStatus = "cancel_acknowledged"
	AttemptSucceeded          AttemptStatus = "succeeded"
	AttemptFailed             AttemptStatus = "failed"
	AttemptCancelled          AttemptStatus = "cancelled"
	AttemptUnknown            AttemptStatus = "unknown"
)

type CheckStatus string

const (
	CheckPending     CheckStatus = "pending"
	CheckPassed      CheckStatus = "passed"
	CheckFailed      CheckStatus = "failed"
	CheckSkipped     CheckStatus = "skipped"
	CheckUnavailable CheckStatus = "unavailable"
	CheckUnknown     CheckStatus = "unknown"
)

type ReviewDecision string

const (
	ReviewApproved         ReviewDecision = "approved"
	ReviewChangesRequested ReviewDecision = "changes_requested"
)

type ActionKind string

const (
	ActionCancel    ActionKind = "cancel"
	ActionRepair    ActionKind = "repair"
	ActionReconcile ActionKind = "reconcile"
	ActionResolve   ActionKind = "resolve_execution"
)

type Resolution string

const (
	ResolutionStillRunning Resolution = "still_running"
	ResolutionNotStarted   Resolution = "not_started"
	ResolutionFailed       Resolution = "stopped_failed"
	ResolutionCancelled    Resolution = "stopped_cancelled"
)

type WorkerReportKind string

const (
	ReportStartRegistered    WorkerReportKind = "start_registered"
	ReportStarted            WorkerReportKind = "started"
	ReportCancelAcknowledged WorkerReportKind = "cancel_acknowledged"
	ReportTerminal           WorkerReportKind = "terminal"
	ReportUnknown            WorkerReportKind = "unknown"
)

type WorkerOutcome string

const (
	OutcomeSucceeded WorkerOutcome = "succeeded"
	OutcomeFailed    WorkerOutcome = "failed"
	OutcomeCancelled WorkerOutcome = "cancelled"
	OutcomeUnknown   WorkerOutcome = "unknown"
)

type WorkerDirective string

const (
	DirectiveContinue WorkerDirective = "continue"
	DirectiveCancel   WorkerDirective = "cancel"
)

type ErrorCode string

const (
	CodeInvalid         ErrorCode = "invalid"
	CodeUnauthenticated ErrorCode = "unauthenticated"
	CodeForbidden       ErrorCode = "forbidden"
	CodeNotFound        ErrorCode = "not_found"
	CodeConflict        ErrorCode = "conflict"
	CodeStale           ErrorCode = "stale"
	CodeNotReady        ErrorCode = "not_ready"
	CodeNoWork          ErrorCode = "no_work"
	CodeUncertain       ErrorCode = "uncertain"
	CodeDeadline        ErrorCode = "deadline_exceeded"
	CodeExhausted       ErrorCode = "repair_exhausted"
	CodeLocked          ErrorCode = "installation_locked"
	CodeInternal        ErrorCode = "internal"
)

type Error struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Err     error          `json:"-"`
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func ErrorCodeOf(err error) ErrorCode {
	var target *Error
	if errors.As(err, &target) {
		return target.Code
	}
	return CodeInternal
}

type Options struct {
	LeaseDuration   time.Duration
	MaxContextFiles int
	MaxContextBytes int64
	GitBinary       string
	Now             func() time.Time
	// DecisionModel names an optional Ollama decision model, such as kev:0.8b-mxfp8,
	// that flags acceptance criteria that may not be checkable. Empty disables it.
	DecisionModel     string
	DecisionURL       string
	DecisionThreshold float64
	// RefinerCommand is the argv run to propose refined acceptance criteria. It reads a
	// RefinerInput on stdin and writes a RefinerOutput on stdout. Empty disables
	// refinement.
	RefinerCommand []string
}

// DefaultDecisionThreshold flags criteria whose checkable probability falls below it.
// It comes from Kev 0.8B trials on 37 real and 6 vague criteria and is uncalibrated.
const DefaultDecisionThreshold = 0.25

type Principal struct {
	UserID            UserID   `json:"userId"`
	Username          string   `json:"username"`
	Kind              UserKind `json:"kind"`
	InstallationAdmin bool     `json:"installationAdmin"`
}

type User struct {
	ID                UserID    `json:"id"`
	Username          string    `json:"username"`
	Kind              UserKind  `json:"kind"`
	InstallationAdmin bool      `json:"installationAdmin"`
	Disabled          bool      `json:"disabled"`
	CreatedAt         time.Time `json:"createdAt"`
}

type BootstrapRequest struct {
	Username string `json:"username"`
}

type BootstrapResult struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type CreateUserRequest struct {
	RequestID         string   `json:"requestId"`
	Username          string   `json:"username"`
	Kind              UserKind `json:"kind"`
	InstallationAdmin bool     `json:"installationAdmin"`
}

type IssueTokenRequest struct {
	RequestID string     `json:"requestId"`
	UserID    UserID     `json:"userId"`
	Label     string     `json:"label"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type SetUserDisabledRequest struct {
	RequestID string `json:"requestId"`
	UserID    UserID `json:"userId"`
	Disabled  bool   `json:"disabled"`
}

type RevokeTokenRequest struct {
	RequestID string `json:"requestId"`
	TokenID   ID     `json:"tokenId"`
}

type IssuedToken struct {
	ID        ID         `json:"id"`
	UserID    UserID     `json:"userId"`
	Label     string     `json:"label"`
	Token     string     `json:"token"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Grant struct {
	ProjectID    ProjectID    `json:"projectId"`
	UserID       UserID       `json:"userId"`
	Capabilities []Capability `json:"capabilities"`
}

type GrantProjectRequest struct {
	RequestID    string       `json:"requestId"`
	ProjectID    ProjectID    `json:"projectId"`
	UserID       UserID       `json:"userId"`
	Capabilities []Capability `json:"capabilities"`
}

type ExecutorProfile struct {
	Key         string    `json:"key"`
	DisplayName string    `json:"displayName"`
	Command     []string  `json:"command"`
	Fixture     bool      `json:"fixture"`
	CreatedAt   time.Time `json:"createdAt"`
}

type RegisterExecutorRequest struct {
	RequestID   string   `json:"requestId"`
	Key         string   `json:"key"`
	DisplayName string   `json:"displayName"`
	Command     []string `json:"command"`
	Fixture     bool     `json:"fixture"`
}

type CheckProfile struct {
	Key         string    `json:"key"`
	DisplayName string    `json:"displayName"`
	Command     []string  `json:"command"`
	Environment string    `json:"environment"`
	Fixture     bool      `json:"fixture"`
	CreatedAt   time.Time `json:"createdAt"`
}

type RegisterCheckRequest struct {
	RequestID   string   `json:"requestId"`
	Key         string   `json:"key"`
	DisplayName string   `json:"displayName"`
	Command     []string `json:"command"`
	Environment string   `json:"environment"`
	Fixture     bool     `json:"fixture"`
}

type ProjectPolicy struct {
	RequiredChecks        []string `json:"requiredChecks"`
	AllowedExecutors      []string `json:"allowedExecutors"`
	MaxRepairAttempts     int      `json:"maxRepairAttempts"`
	MaxJobDurationSeconds int64    `json:"maxJobDurationSeconds"`
	Fixture               bool     `json:"fixture"`
	// ProtectedPaths are repository path patterns an implementation candidate may not
	// change. They keep the tests and expectations that verification runs out of the
	// implementer's reach.
	ProtectedPaths []string `json:"protectedPaths,omitempty"`
	// AllowCoordinatorReview lets the human who submitted a job also review it. It suits
	// a single-person project; the reviewer must still be human and not the implementer.
	AllowCoordinatorReview bool `json:"allowCoordinatorReview,omitempty"`
}

type ReaderLimits struct {
	MaxFiles int   `json:"maxFiles"`
	MaxBytes int64 `json:"maxBytes"`
}

type Project struct {
	ID                 ProjectID     `json:"id"`
	Title              string        `json:"title"`
	BundlePath         string        `json:"bundlePath"`
	RepositoryPath     string        `json:"repositoryPath"`
	AllowedSourceRoots []string      `json:"allowedSourceRoots"`
	ReaderLimits       ReaderLimits  `json:"readerLimits"`
	Policy             ProjectPolicy `json:"policy"`
	CreatedAt          time.Time     `json:"createdAt"`
}

type RegisterProjectRequest struct {
	RequestID          string        `json:"requestId"`
	BundlePath         string        `json:"bundlePath"`
	RepositoryPath     string        `json:"repositoryPath"`
	AllowedSourceRoots []string      `json:"allowedSourceRoots"`
	ReaderLimits       ReaderLimits  `json:"readerLimits"`
	Policy             ProjectPolicy `json:"policy"`
}

type ProjectOrientation struct {
	Project          Project                      `json:"project"`
	Result           orientation.Result           `json:"result"`
	CriterionSignals map[string]*CriterionSignals `json:"criterionSignals,omitempty"`
}

type ContextSource struct {
	Path    string          `json:"path"`
	Text    string          `json:"text"`
	SHA256  string          `json:"sha256"`
	Reasons json.RawMessage `json:"reasons"`
}

type ContextDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
}

type Admission struct {
	TicketPath         string              `json:"ticketPath"`
	BaseRevision       string              `json:"baseRevision"`
	FingerprintVersion int                 `json:"fingerprintVersion"`
	TicketSHA256       string              `json:"ticketSHA256"`
	CriteriaSHA256     string              `json:"criteriaSHA256"`
	ContextSHA256      string              `json:"contextSHA256"`
	ContextComplete    bool                `json:"contextComplete"`
	Sources            []ContextSource     `json:"sources"`
	Diagnostics        []ContextDiagnostic `json:"diagnostics"`
	ExecutorKey        string              `json:"executorKey"`
	RequiredChecks     []string            `json:"requiredChecks"`
	RepairLimit        int                 `json:"repairLimit"`
	Deadline           time.Time           `json:"deadline"`
	AdmittedAt         time.Time           `json:"admittedAt"`
	// ProtectedPaths and AllowCoordinatorReview freeze the project policy in force at
	// admission, so a later policy edit cannot change the rules for running work.
	ProtectedPaths         []string `json:"protectedPaths,omitempty"`
	AllowCoordinatorReview bool     `json:"allowCoordinatorReview,omitempty"`
	// CriterionSignals are advisory decision-model observations taken at admission.
	// They never affect readiness, admission, or acceptance.
	CriterionSignals *CriterionSignals `json:"criterionSignals,omitempty"`
}

type SubmitRequest struct {
	RequestID    string    `json:"requestId"`
	ProjectID    ProjectID `json:"projectId"`
	WorkItemID   string    `json:"workItemId"`
	BaseRevision string    `json:"baseRevision"`
	ExecutorKey  string    `json:"executorKey"`
	RepairLimit  int       `json:"repairLimit"`
	Deadline     time.Time `json:"deadline"`
}

type ArtifactInput struct {
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
	Content   string `json:"content"`
}

type Artifact struct {
	ID        ArtifactID `json:"id"`
	Name      string     `json:"name"`
	MediaType string     `json:"mediaType"`
	SHA256    string     `json:"sha256"`
	Bytes     int        `json:"bytes"`
	Content   string     `json:"content,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Attempt struct {
	ID                    AttemptID     `json:"id"`
	JobID                 JobID         `json:"jobId"`
	Stage                 Stage         `json:"stage"`
	Ordinal               int           `json:"ordinal"`
	CheckKey              string        `json:"checkKey,omitempty"`
	DispatchID            string        `json:"dispatchId"`
	WorkerID              *UserID       `json:"workerId,omitempty"`
	NativeRunID           string        `json:"nativeRunId,omitempty"`
	WorkspacePath         string        `json:"workspacePath"`
	InputRevision         string        `json:"inputRevision"`
	CandidateRevision     string        `json:"candidateRevision,omitempty"`
	Status                AttemptStatus `json:"status"`
	Outcome               WorkerOutcome `json:"outcome,omitempty"`
	CheckStatus           CheckStatus   `json:"checkStatus,omitempty"`
	LeaseExpiresAt        *time.Time    `json:"leaseExpiresAt,omitempty"`
	LastHeartbeatAt       *time.Time    `json:"lastHeartbeatAt,omitempty"`
	StartedAt             *time.Time    `json:"startedAt,omitempty"`
	FinishedAt            *time.Time    `json:"finishedAt,omitempty"`
	CancellationRequested bool          `json:"cancellationRequested"`
	Summary               string        `json:"summary,omitempty"`
	TerminalReceipt       string        `json:"terminalReceipt,omitempty"`
	Artifacts             []Artifact    `json:"artifacts"`
}

type CheckResult struct {
	CheckKey               string      `json:"checkKey"`
	Status                 CheckStatus `json:"status"`
	CandidateRevision      string      `json:"candidateRevision"`
	RequirementFingerprint string      `json:"requirementFingerprint"`
	ObserverID             *UserID     `json:"observerId,omitempty"`
	AttemptID              *AttemptID  `json:"attemptId,omitempty"`
	Command                []string    `json:"command"`
	Environment            string      `json:"environment"`
	ObservedAt             *time.Time  `json:"observedAt,omitempty"`
	Artifacts              []Artifact  `json:"artifacts"`
}

type Review struct {
	ID                     ID             `json:"id"`
	ReviewerID             UserID         `json:"reviewerId"`
	CandidateRevision      string         `json:"candidateRevision"`
	RequirementFingerprint string         `json:"requirementFingerprint"`
	Decision               ReviewDecision `json:"decision"`
	Summary                string         `json:"summary"`
	CreatedAt              time.Time      `json:"createdAt"`
}

type Event struct {
	Sequence  int64           `json:"sequence"`
	JobID     JobID           `json:"jobId"`
	ActorID   *UserID         `json:"actorId,omitempty"`
	Kind      string          `json:"kind"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"createdAt"`
}

type Job struct {
	ID                   JobID         `json:"id"`
	Version              int64         `json:"version"`
	ProjectID            ProjectID     `json:"projectId"`
	WorkItemID           string        `json:"workItemId"`
	CoordinatorID        UserID        `json:"coordinatorId"`
	ImplementerID        *UserID       `json:"implementerId,omitempty"`
	Status               JobStatus     `json:"status"`
	Admission            Admission     `json:"admission"`
	CandidateRevision    string        `json:"candidateRevision,omitempty"`
	RepairCount          int           `json:"repairCount"`
	Attempts             []Attempt     `json:"attempts"`
	Checks               []CheckResult `json:"checks"`
	Reviews              []Review      `json:"reviews"`
	Events               []Event       `json:"events"`
	AvailableActions     []ActionKind  `json:"availableActions"`
	AcceptanceRecordPath string        `json:"acceptanceRecordPath,omitempty"`
	AcceptanceStatus     string        `json:"acceptanceStatus"`
	CreatedAt            time.Time     `json:"createdAt"`
	UpdatedAt            time.Time     `json:"updatedAt"`
}

type JobSummary struct {
	ID                JobID     `json:"id"`
	Version           int64     `json:"version"`
	ProjectID         ProjectID `json:"projectId"`
	WorkItemID        string    `json:"workItemId"`
	Status            JobStatus `json:"status"`
	CandidateRevision string    `json:"candidateRevision,omitempty"`
	Deadline          time.Time `json:"deadline"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type JobFilter struct {
	ProjectID ProjectID `json:"projectId,omitempty"`
	Status    JobStatus `json:"status,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	After     string    `json:"after,omitempty"`
}

type WorkerClaimRequest struct {
	RequestID string     `json:"requestId"`
	Stages    []Stage    `json:"stages"`
	ProjectID *ProjectID `json:"projectId,omitempty"`
}

type Assignment struct {
	AttemptID              AttemptID       `json:"attemptId"`
	JobID                  JobID           `json:"jobId"`
	ProjectID              ProjectID       `json:"projectId"`
	WorkItemID             string          `json:"workItemId"`
	Stage                  Stage           `json:"stage"`
	WorkerID               UserID          `json:"workerId"`
	DispatchID             string          `json:"dispatchId"`
	WorkspacePath          string          `json:"workspacePath"`
	BaseRevision           string          `json:"baseRevision"`
	CandidateRevision      string          `json:"candidateRevision,omitempty"`
	RequirementFingerprint string          `json:"requirementFingerprint"`
	ContextSHA256          string          `json:"contextSHA256"`
	ContextSources         []ContextSource `json:"contextSources"`
	Instructions           string          `json:"instructions"`
	ProfileKey             string          `json:"profileKey"`
	Command                []string        `json:"command"`
	CheckEnvironment       string          `json:"checkEnvironment,omitempty"`
	Fixture                bool            `json:"fixture"`
	Deadline               time.Time       `json:"deadline"`
	LeaseExpiresAt         time.Time       `json:"leaseExpiresAt"`
	ProtectedPaths         []string        `json:"protectedPaths,omitempty"`
}

type HeartbeatRequest struct {
	RequestID  string    `json:"requestId"`
	AttemptID  AttemptID `json:"attemptId"`
	DispatchID string    `json:"dispatchId"`
	// Progress is the latest note the running command wrote, such as a finished
	// checkpoint. It is an observation for operators, not evidence.
	Progress string `json:"progress,omitempty"`
}

type HeartbeatResult struct {
	AttemptID      AttemptID       `json:"attemptId"`
	Directive      WorkerDirective `json:"directive"`
	LeaseExpiresAt time.Time       `json:"leaseExpiresAt"`
}

type WorkerReportRequest struct {
	RequestID         string           `json:"requestId"`
	AttemptID         AttemptID        `json:"attemptId"`
	DispatchID        string           `json:"dispatchId"`
	Kind              WorkerReportKind `json:"kind"`
	ObservedAt        time.Time        `json:"observedAt"`
	NativeRunID       string           `json:"nativeRunId,omitempty"`
	Outcome           WorkerOutcome    `json:"outcome,omitempty"`
	CandidateRevision string           `json:"candidateRevision,omitempty"`
	CheckStatus       CheckStatus      `json:"checkStatus,omitempty"`
	Summary           string           `json:"summary,omitempty"`
	TerminalReceipt   string           `json:"terminalReceipt,omitempty"`
	Artifacts         []ArtifactInput  `json:"artifacts,omitempty"`
}

type ReviewRequest struct {
	RequestID              string         `json:"requestId"`
	JobID                  JobID          `json:"jobId"`
	IfVersion              int64          `json:"ifVersion"`
	CandidateRevision      string         `json:"candidateRevision"`
	RequirementFingerprint string         `json:"requirementFingerprint"`
	Decision               ReviewDecision `json:"decision"`
	Summary                string         `json:"summary"`
}

type JobActionRequest struct {
	RequestID  string     `json:"requestId"`
	JobID      JobID      `json:"jobId"`
	IfVersion  int64      `json:"ifVersion"`
	Action     ActionKind `json:"action"`
	AttemptID  *AttemptID `json:"attemptId,omitempty"`
	Resolution Resolution `json:"resolution,omitempty"`
	Evidence   string     `json:"evidence,omitempty"`
}

type AcceptRequest struct {
	RequestID              string `json:"requestId"`
	JobID                  JobID  `json:"jobId"`
	IfVersion              int64  `json:"ifVersion"`
	CandidateRevision      string `json:"candidateRevision"`
	RequirementFingerprint string `json:"requirementFingerprint"`
	Summary                string `json:"summary"`
}

// Service has one durable controller and one workspace owner for its data directory.
type Service struct {
	state *serviceState
}
