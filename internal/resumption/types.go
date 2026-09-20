// Package resumption refreshes current project and task facts and inspects
// recovery observations for one selected work item.
package resumption

import (
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

const (
	DefaultMaxCacheFiles       = 200
	DefaultMaxCacheBytes int64 = 4_194_304
)

type Request struct {
	RecordsDirectory  string
	WorkingDirectory  string
	TicketPath        string
	AllowedSourceDirs []string
	MaxFiles          int
	MaxBytes          int64
	MaxCacheFiles     int
	MaxCacheBytes     int64
}

type Result struct {
	SchemaVersion int                `json:"schemaVersion"`
	Kind          string             `json:"kind"`
	Complete      bool               `json:"complete"`
	Scope         Scope              `json:"scope"`
	Orientation   orientation.Result `json:"orientation"`
	Context       taskcontext.Result `json:"context"`
	Recovery      Recovery           `json:"recovery"`
	Comparison    Comparison         `json:"comparison"`
	Diagnostics   []Diagnostic       `json:"diagnostics"`
}

type Scope struct {
	RecordsDirectory string  `json:"recordsDirectory"`
	WorkingDirectory string  `json:"workingDirectory"`
	CacheRoot        string  `json:"cacheRoot"`
	ProjectID        *string `json:"projectId"`
	TaskID           *string `json:"taskId"`
}

type Recovery struct {
	Status            string        `json:"status"`
	InventoryComplete bool          `json:"inventoryComplete"`
	GraphStatus       string        `json:"graphStatus"`
	Observations      []Observation `json:"observations"`
	Candidates        []string      `json:"candidates"`
}

type Observation struct {
	ID               string         `json:"id"`
	ProjectID        string         `json:"projectId"`
	TaskID           string         `json:"taskId"`
	ObservedAt       string         `json:"observedAt"`
	Actor            string         `json:"actor"`
	Predecessors     []string       `json:"predecessors"`
	TicketPath       string         `json:"ticketPath"`
	CheckoutRevision *Revision      `json:"checkoutRevision"`
	Source           FileReference  `json:"source"`
	Body             string         `json:"body"`
	Metadata         map[string]any `json:"metadata"`
	Snapshot         SnapshotReport `json:"snapshot"`
}

type Revision struct {
	Origin   string `json:"origin"`
	Revision string `json:"revision"`
}

type FileReference struct {
	Path   string  `json:"path"`
	SHA256 *string `json:"sha256"`
}

type SnapshotReport struct {
	Path           string  `json:"path"`
	RecordedSHA256 string  `json:"recordedSHA256"`
	ObservedSHA256 *string `json:"observedSHA256"`
	Status         string  `json:"status"`
	SourceCount    *int    `json:"sourceCount"`
}

type Comparison struct {
	BaselineAvailable bool                  `json:"baselineAvailable"`
	Complete          bool                  `json:"complete"`
	Candidates        []CandidateComparison `json:"candidates"`
}

type CandidateComparison struct {
	ObservationID     string             `json:"observationId"`
	BaselineAvailable bool               `json:"baselineAvailable"`
	Complete          bool               `json:"complete"`
	Sources           []SourceDifference `json:"sources"`
}

type SourceDifference struct {
	Status   string          `json:"status"`
	Previous *ComparedSource `json:"previous"`
	Current  *ComparedSource `json:"current"`
}

type ComparedSource struct {
	Path         string  `json:"path"`
	SHA256       *string `json:"sha256"`
	Text         *string `json:"text"`
	Availability string  `json:"availability"`
}

type Diagnostic struct {
	Code          string  `json:"code"`
	Severity      string  `json:"severity"`
	Message       string  `json:"message"`
	Path          *string `json:"path"`
	From          *string `json:"from"`
	Link          *string `json:"link"`
	ObservationID *string `json:"observationId"`
}
