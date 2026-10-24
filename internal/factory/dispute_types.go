package factory

import (
	"strings"
	"time"
)

type DisputeInput struct {
	Summary         string   `json:"summary"`
	CriterionRefs   []int    `json:"criterionRefs,omitempty"`
	TestPaths       []string `json:"testPaths,omitempty"`
	ObservedFailure string   `json:"observedFailure"`
	EvidenceRefs    []string `json:"evidenceRefs"`
	ProposedChange  string   `json:"proposedChange,omitempty"`
}

func ValidateDisputeInput(input DisputeInput) error {
	if strings.TrimSpace(input.Summary) == "" || strings.TrimSpace(input.ObservedFailure) == "" || len(input.EvidenceRefs) == 0 || len(input.CriterionRefs)+len(input.TestPaths) == 0 {
		return fault(CodeInvalid, "a dispute requires summary, observedFailure, evidenceRefs, and affected criterionRefs or testPaths", nil)
	}
	for _, value := range append(append([]string{input.Summary, input.ObservedFailure, input.ProposedChange}, input.EvidenceRefs...), input.TestPaths...) {
		if len(value) > 16000 {
			return fault(CodeInvalid, "dispute text exceeds 16000 bytes", nil)
		}
	}
	if len(input.EvidenceRefs) > 20 || len(input.TestPaths) > 20 || len(input.CriterionRefs) > 20 {
		return fault(CodeInvalid, "dispute references exceed 20 entries", nil)
	}
	for _, n := range input.CriterionRefs {
		if n < 1 {
			return fault(CodeInvalid, "criterionRefs must be positive", nil)
		}
	}
	for _, value := range append(append([]string{}, input.EvidenceRefs...), input.TestPaths...) {
		if strings.TrimSpace(value) == "" {
			return fault(CodeInvalid, "dispute references must not be empty", nil)
		}
	}
	return nil
}

type DisputeDecisionKind string

const (
	DisputeRepair      DisputeDecisionKind = "repair_implementation"
	DisputeReviseTests DisputeDecisionKind = "revise_tests"
	DisputeAddTests    DisputeDecisionKind = "add_tests"
	DisputeNeedsHuman  DisputeDecisionKind = "needs_human"
	DisputeAcceptTests DisputeDecisionKind = "accept_tests"
)

type JudgeDecision struct {
	Kind              DisputeDecisionKind `json:"kind"`
	Rationale         string              `json:"rationale"`
	EvidenceRefs      []string            `json:"evidenceRefs"`
	PreservesIntent   bool                `json:"preservesIntent"`
	WeakensAcceptance bool                `json:"weakensAcceptance"`
	Directive         string              `json:"directive,omitempty"`
}

// DisputeTestReview binds a retained judge invocation to one immutable test proposal.
type DisputeTestReview struct {
	ExpectationID ID             `json:"expectationId"`
	Commit        string         `json:"commit"`
	Round         int            `json:"round"`
	Decision      *JudgeDecision `json:"decision,omitempty"`
	Run           *RefinerRun    `json:"run,omitempty"`
	Log           string         `json:"log,omitempty"`
	LogTruncated  bool           `json:"logTruncated,omitempty"`
	Error         string         `json:"error,omitempty"`
	NextOwner     string         `json:"nextOwner"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type Dispute struct {
	Delegated         string              `json:"delegated,omitempty"`
	TestProposalCount int                 `json:"testProposalCount"`
	TestReviews       []DisputeTestReview `json:"testReviews,omitempty"`
	ID                ID                  `json:"id"`
	ProjectID         ProjectID           `json:"projectId"`
	WorkItemID        string              `json:"workItemId"`
	JobID             JobID               `json:"jobId"`
	LineageID         JobID               `json:"lineageId"`
	Version           int64               `json:"version"`
	Status            string              `json:"status"`
	Input             DisputeInput        `json:"input"`
	Decision          *JudgeDecision      `json:"decision,omitempty"`
	NextOwner         string              `json:"nextOwner"`
	ExpectationID     ID                  `json:"expectationId,omitempty"`
	ReplacementJobID  JobID               `json:"replacementJobId,omitempty"`
	RequestedBy       UserID              `json:"requestedBy"`
	ResolvedBy        *UserID             `json:"resolvedBy,omitempty"`
	Resolution        string              `json:"resolution,omitempty"`
	Rationale         string              `json:"rationale,omitempty"`
	Error             string              `json:"error,omitempty"`
	Run               *RefinerRun         `json:"run,omitempty"`
	Log               string              `json:"log,omitempty"`
	LogTruncated      bool                `json:"logTruncated,omitempty"`
	AvailableActions  []string            `json:"availableActions"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
}
type ReportDisputeRequest struct {
	RequestID string       `json:"requestId"`
	JobID     JobID        `json:"jobId"`
	IfVersion int64        `json:"ifVersion"`
	Dispute   DisputeInput `json:"dispute"`
}
type ResolveDisputeRequest struct {
	RequestID string `json:"requestId"`
	DisputeID ID     `json:"disputeId"`
	IfVersion int64  `json:"ifVersion"`
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
}
type JudgeInput struct {
	ProposedTests       *Expectation    `json:"proposedTests,omitempty"`
	CandidatePatch      string          `json:"candidatePatch,omitempty"`
	Dispute             Dispute         `json:"dispute"`
	Job                 Job             `json:"job"`
	OriginalTests       *Expectation    `json:"originalTests,omitempty"`
	TestSources         []ContextSource `json:"testSources"`
	ProtectedPaths      []string        `json:"protectedPaths"`
	TestAuthorAvailable bool            `json:"testAuthorAvailable"`
	Instructions        string          `json:"instructions,omitempty"`
}
type JudgeOutput struct {
	Decision JudgeDecision `json:"decision"`
	Run      RefinerRun    `json:"run"`
}
