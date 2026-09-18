package resumption

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
)

var errNoteIdentity = errors.New("recovery note identity mismatch")

var (
	noteUUIDPattern      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	noteDigestPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	noteTimestampPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?(?:Z|[+-][0-9]{2}:[0-9]{2})$`)
)

var requiredNoteSections = []string{
	"Approach",
	"Completed",
	"Remaining",
	"Checks",
	"Questions",
	"Failed approaches",
	"Next step",
}

// parseNote validates a finalized recovery note and returns the observation it
// describes. Snapshot bytes are deliberately left unloaded until graph
// selection establishes that this observation is a candidate.
func parseNote(data []byte, notePath, directoryID, projectID, taskID string) (Observation, error) {
	if !utf8.Valid(data) {
		return Observation{}, errors.New("recovery note must be valid UTF-8")
	}
	document, err := recordread.ParseDocument(data)
	if err != nil {
		return Observation{}, fmt.Errorf("parse recovery note: %w", err)
	}
	metadata := document.Metadata

	noteType, err := requiredNoteString(metadata, "type", false)
	if err != nil {
		return Observation{}, err
	}
	if noteType != "RecoveryNote" {
		return Observation{}, fmt.Errorf("recovery note field %q must be exactly RecoveryNote", "type")
	}
	if err := requireNoteVersion(metadata); err != nil {
		return Observation{}, err
	}

	id, err := requiredNoteString(metadata, "id", true)
	if err != nil {
		return Observation{}, err
	}
	if !noteUUIDPattern.MatchString(id) {
		return Observation{}, fmt.Errorf("recovery note field %q must be a lowercase hyphenated UUID", "id")
	}
	project, err := requiredNoteString(metadata, "projectId", true)
	if err != nil {
		return Observation{}, err
	}
	task, err := requiredNoteString(metadata, "taskId", true)
	if err != nil {
		return Observation{}, err
	}
	actor, err := requiredNoteString(metadata, "actor", true)
	if err != nil {
		return Observation{}, err
	}
	ticketPath, err := requiredNoteString(metadata, "ticketPath", true)
	if err != nil {
		return Observation{}, err
	}

	if id != directoryID {
		return Observation{}, fmt.Errorf("%w: note ID %q does not match observation directory %q", errNoteIdentity, id, directoryID)
	}
	if project != strings.TrimSpace(projectID) {
		return Observation{}, fmt.Errorf("%w: project ID %q does not match selected project %q", errNoteIdentity, project, strings.TrimSpace(projectID))
	}
	if task != strings.TrimSpace(taskID) {
		return Observation{}, fmt.Errorf("%w: task ID %q does not match selected task %q", errNoteIdentity, task, strings.TrimSpace(taskID))
	}

	observedAt, err := requiredNoteString(metadata, "observedAt", false)
	if err != nil {
		return Observation{}, err
	}
	if !validNoteTimestamp(observedAt) {
		return Observation{}, fmt.Errorf("recovery note field %q must be an RFC 3339 timestamp", "observedAt")
	}
	if _, err := time.Parse(time.RFC3339, observedAt); err != nil {
		return Observation{}, fmt.Errorf("recovery note field %q must be an RFC 3339 timestamp: %w", "observedAt", err)
	}

	predecessors, err := notePredecessors(metadata, id)
	if err != nil {
		return Observation{}, err
	}
	checkoutRevision, err := noteRevision(metadata)
	if err != nil {
		return Observation{}, err
	}
	contextFile, err := requiredNoteString(metadata, "contextFile", false)
	if err != nil {
		return Observation{}, err
	}
	if contextFile != "context.json" {
		return Observation{}, fmt.Errorf("recovery note field %q must be exactly context.json", "contextFile")
	}
	contextDigest, err := requiredNoteString(metadata, "contextSHA256", false)
	if err != nil {
		return Observation{}, err
	}
	if !noteDigestPattern.MatchString(contextDigest) {
		return Observation{}, fmt.Errorf("recovery note field %q must be a lowercase 64-character SHA-256 digest", "contextSHA256")
	}
	if err := validateNoteSections(document.Body, notePath); err != nil {
		return Observation{}, err
	}

	noteDigest := fmt.Sprintf("%x", sha256.Sum256(data))
	return Observation{
		ID:               id,
		ProjectID:        project,
		TaskID:           task,
		ObservedAt:       observedAt,
		Actor:            actor,
		Predecessors:     predecessors,
		TicketPath:       ticketPath,
		CheckoutRevision: checkoutRevision,
		Source:           FileReference{Path: notePath, SHA256: &noteDigest},
		Body:             string(document.Body),
		Metadata:         orientation.MetadataForOutput(metadata),
		Snapshot: SnapshotReport{
			Path:           filepath.Join(filepath.Dir(notePath), contextFile),
			RecordedSHA256: contextDigest,
			ObservedSHA256: nil,
			Status:         "not_loaded",
			SourceCount:    nil,
		},
	}, nil
}

func validNoteTimestamp(value string) bool {
	if !noteTimestampPattern.MatchString(value) {
		return false
	}
	if value[len(value)-1] == 'Z' {
		return true
	}
	offset := value[len(value)-6:]
	hour := int(offset[1]-'0')*10 + int(offset[2]-'0')
	minute := int(offset[4]-'0')*10 + int(offset[5]-'0')
	return hour <= 23 && minute <= 59
}

func requiredNoteString(metadata map[string]any, name string, trim bool) (string, error) {
	value, exists := metadata[name]
	if !exists || value == nil {
		return "", fmt.Errorf("recovery note requires non-null field %q", name)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("recovery note field %q must be a string", name)
	}
	if trim {
		text = strings.TrimSpace(text)
	}
	if text == "" {
		return "", fmt.Errorf("recovery note field %q must be nonempty", name)
	}
	return text, nil
}

func requireNoteVersion(metadata map[string]any) error {
	value, exists := metadata["version"]
	if !exists || value == nil {
		return errors.New("recovery note requires non-null field \"version\"")
	}
	typeOf := reflect.TypeOf(value)
	integer := false
	one := false
	if typeOf != nil {
		switch typeOf.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			integer, one = true, reflect.ValueOf(value).Int() == 1
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			integer, one = true, reflect.ValueOf(value).Uint() == 1
		}
	}
	if !integer || !one {
		return errors.New("recovery note field \"version\" must be integer 1")
	}
	return nil
}

func notePredecessors(metadata map[string]any, id string) ([]string, error) {
	value, exists := metadata["predecessors"]
	if !exists || value == nil {
		return nil, errors.New("recovery note requires non-null field \"predecessors\"")
	}
	items, ok := value.([]any)
	if !ok {
		return nil, errors.New("recovery note field \"predecessors\" must be an array")
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for index, item := range items {
		predecessor, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("recovery note field \"predecessors[%d]\" must be a string", index)
		}
		if !noteUUIDPattern.MatchString(predecessor) {
			return nil, fmt.Errorf("recovery note field \"predecessors[%d]\" must be a lowercase hyphenated UUID", index)
		}
		if predecessor == id {
			return nil, fmt.Errorf("recovery note predecessor %q must not refer to the note itself", predecessor)
		}
		if _, duplicate := seen[predecessor]; duplicate {
			return nil, fmt.Errorf("recovery note predecessor %q is duplicated", predecessor)
		}
		seen[predecessor] = struct{}{}
		result = append(result, predecessor)
	}
	return result, nil
}

func noteRevision(metadata map[string]any) (*Revision, error) {
	value, exists := metadata["checkoutRevision"]
	if !exists {
		return nil, errors.New("recovery note requires field \"checkoutRevision\"")
	}
	if value == nil {
		return nil, nil
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("recovery note field \"checkoutRevision\" must be null or an object")
	}
	origin, err := requiredNoteString(fields, "origin", true)
	if err != nil {
		return nil, fmt.Errorf("recovery note checkoutRevision: %w", err)
	}
	revision, err := requiredNoteString(fields, "revision", true)
	if err != nil {
		return nil, fmt.Errorf("recovery note checkoutRevision: %w", err)
	}
	return &Revision{Origin: origin, Revision: revision}, nil
}

func validateNoteSections(body []byte, notePath string) error {
	names := make(map[string]string, len(requiredNoteSections))
	for _, name := range requiredNoteSections {
		names[name] = "recovery_note"
	}
	parsed := recordread.Sections(body, notePath, names)
	counts := make(map[string]int, len(requiredNoteSections))
	for _, section := range parsed.Sections {
		counts[section.Name]++
	}
	for _, name := range requiredNoteSections {
		switch counts[name] {
		case 0:
			return fmt.Errorf("recovery note is missing required level-two section %q", name)
		case 1:
		default:
			return fmt.Errorf("recovery note repeats required level-two section %q", name)
		}
	}
	return nil
}
