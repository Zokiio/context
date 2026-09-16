// Package orientation returns an attributed, read-only overview of a project.
package orientation

const (
	DefaultMaxFiles       = 100
	DefaultMaxBytes int64 = 1_048_576
)

type Request struct {
	ProjectDir        string
	AllowedSourceDirs []string
	// Zero uses the default. Negative limits are invalid.
	MaxFiles int
	MaxBytes int64
}

type Result struct {
	SchemaVersion      int          `json:"schemaVersion"`
	Complete           bool         `json:"complete"`
	InventoryComplete  bool         `json:"inventoryComplete"`
	Project            *Project     `json:"project"`
	Goals              []Goal       `json:"goals"`
	CurrentCommitments []Reference  `json:"currentCommitments"`
	WorkItems          []WorkItem   `json:"workItems"`
	Shortlist          []Reference  `json:"shortlist"`
	InProgress         []Reference  `json:"inProgress"`
	Backlog            []Reference  `json:"backlog"`
	Decisions          []Decision   `json:"decisions"`
	Sources            []Source     `json:"sources"`
	Diagnostics        []Diagnostic `json:"diagnostics"`
}

type Project struct {
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Source             string         `json:"source"`
	GoalsKnown         bool           `json:"goalsKnown"`
	CommitmentsKnown   bool           `json:"commitmentsKnown"`
	OpenDecisionsKnown bool           `json:"openDecisionsKnown"`
	Metadata           map[string]any `json:"metadata"`
}

type Goal struct {
	Text       string      `json:"text"`
	Source     string      `json:"source"`
	References []Reference `json:"references"`
}

// Reference retains the authored relationship even when its target is unavailable.
type Reference struct {
	Path  string  `json:"path"`
	ID    *string `json:"id"`
	Title *string `json:"title"`
	From  string  `json:"from,omitempty"`
	Link  string  `json:"link,omitempty"`
}

type WorkItem struct {
	ID                 *string            `json:"id"`
	Title              *string            `json:"title"`
	Source             string             `json:"source"`
	IdentityAmbiguous  bool               `json:"identityAmbiguous"`
	Triage             *string            `json:"triage"`
	Execution          *string            `json:"execution"`
	Lifecycle          *string            `json:"lifecycle"`
	Committed          *bool              `json:"committed"`
	Specifications     []Reference        `json:"specifications"`
	Checks             []Check            `json:"checks"`
	Readiness          string             `json:"readiness"`
	Eligible           bool               `json:"eligible"`
	ExclusionReasons   []Finding          `json:"exclusionReasons"`
	Acceptance         *AcceptanceSummary `json:"acceptance"`
	FingerprintVersion *int               `json:"fingerprintVersion"`
	TicketSHA256       *string            `json:"ticketSHA256"`
	CriteriaSHA256     *string            `json:"criteriaSHA256"`
	Metadata           map[string]any     `json:"metadata"`
}

type Check struct {
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Reasons []Finding `json:"reasons"`
}

type Finding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	From    string `json:"from,omitempty"`
	Link    string `json:"link,omitempty"`
}

type AcceptanceSummary struct {
	Status string     `json:"status"`
	Record *Reference `json:"record"`
}

type Decision struct {
	ID                *string        `json:"id"`
	Title             *string        `json:"title"`
	Source            string         `json:"source"`
	IdentityAmbiguous bool           `json:"identityAmbiguous"`
	State             *string        `json:"state"`
	Resolution        *string        `json:"resolution"`
	CheckStatus       string         `json:"checkStatus"`
	Reasons           []Finding      `json:"reasons"`
	AffectedWork      []Reference    `json:"affectedWork"`
	References        []Reference    `json:"references"`
	Metadata          map[string]any `json:"metadata"`
}

type Source struct {
	Path    string   `json:"path"`
	SHA256  string   `json:"sha256"`
	Reasons []Reason `json:"reasons"`
}

type Reason struct {
	Kind string `json:"kind"`
	From string `json:"from,omitempty"`
	Link string `json:"link,omitempty"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	From     string `json:"from,omitempty"`
	Link     string `json:"link,omitempty"`
}
