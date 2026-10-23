package factory

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
)

const (
	maxWorkItemTitleRunes       = 200
	maxWorkItemDescriptionRunes = 4000
)

// WorkItemDraft is a work item as a person writes it. The record keeps every other
// section as it is.
type WorkItemDraft struct {
	Title string `json:"title"`
	// Description is the prose before the record's first section; every agent reads it.
	Description string   `json:"description,omitempty"`
	Criteria    []string `json:"criteria"`
	// BlockedBy names work items in the same project that must be done first.
	BlockedBy []string `json:"blockedBy,omitempty"`
}

// WorkItemRecord is a work item as the person editing it sees it.
type WorkItemRecord struct {
	ProjectID  ProjectID `json:"projectId"`
	WorkItemID string    `json:"workItemId"`
	// Path is the record's file, relative to the project's records.
	Path         string        `json:"path"`
	TicketSHA256 string        `json:"ticketSHA256"`
	Draft        WorkItemDraft `json:"draft"`
	// Frozen means an active build holds the item, so it cannot change.
	Frozen bool `json:"frozen"`
}

type CreateWorkItemRequest struct {
	RequestID string        `json:"requestId"`
	ProjectID ProjectID     `json:"projectId"`
	Draft     WorkItemDraft `json:"draft"`
}

type UpdateWorkItemRequest struct {
	RequestID      string        `json:"requestId"`
	ProjectID      ProjectID     `json:"projectId"`
	WorkItemID     string        `json:"workItemId"`
	IfTicketSHA256 string        `json:"ifTicketSHA256"`
	Draft          WorkItemDraft `json:"draft"`
}

// GetWorkItem reads a work item's record into the fields a person edits.
func (s *Service) GetWorkItem(ctx context.Context, actor Principal, projectID ProjectID, workItemID string) (WorkItemRecord, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return WorkItemRecord{}, err
	}
	project, err := readProject(ctx, s.state.db, projectID)
	if err != nil {
		return WorkItemRecord{}, err
	}
	result, err := s.orientProject(ctx, project)
	if err != nil {
		return WorkItemRecord{}, err
	}
	work, err := selectWork(result, workItemID)
	if err != nil {
		return WorkItemRecord{}, err
	}
	raw, err := os.ReadFile(work.Source)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "read work item", err)
	}
	criteria, err := acceptanceCriteria(work.Source)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInvalid, "read acceptance criteria", err)
	}
	draft := WorkItemDraft{Description: ticketDescription(raw), Criteria: criteria}
	if work.Title != nil {
		draft.Title = *work.Title
	}
	for _, edge := range work.Dependencies {
		if edge.To.ID != nil {
			draft.BlockedBy = append(draft.BlockedBy, *edge.To.ID)
		}
	}
	frozen := requireNoActiveJob(ctx, s.state.db, projectID, workItemID) != nil
	return WorkItemRecord{ProjectID: projectID, WorkItemID: workItemID, Path: relativeRecord(project.BundlePath, work.Source), TicketSHA256: digestBytes(raw), Draft: draft, Frozen: frozen}, nil
}

// CreateWorkItem writes a new work item record next to the project's others and lists
// it under the project's current commitments, so it is ready work once it is clear.
func (s *Service) CreateWorkItem(ctx context.Context, actor Principal, request CreateWorkItemRequest) (WorkItemRecord, error) {
	if err := requireRequestID(request.RequestID); err != nil {
		return WorkItemRecord{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "start work item", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, request.ProjectID, CapabilityCoordinate)
	if err != nil {
		return WorkItemRecord{}, err
	}
	saved, ok, hash, err := receipt[WorkItemRecord](ctx, tx, current.UserID, "create_work_item", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	project, err := readProject(ctx, tx, request.ProjectID)
	if err != nil {
		return WorkItemRecord{}, err
	}
	result, err := s.orientProject(ctx, project)
	if err != nil {
		return WorkItemRecord{}, err
	}
	if result.Project == nil {
		return WorkItemRecord{}, fault(CodeNotReady, "the project record is unavailable, so new work cannot be listed in it", nil)
	}
	works := workItemsByID(result)
	draft, err := normalizeDraft(request.Draft, works, "")
	if err != nil {
		return WorkItemRecord{}, err
	}
	id, err := newRecordID()
	if err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "create work item identity", err)
	}
	directory := project.BundlePath
	if len(result.WorkItems) > 0 {
		directory = filepath.Dir(result.WorkItems[0].Source)
	}
	path, err := freeRecordPath(directory, slugOf(draft.Title))
	if err != nil {
		return WorkItemRecord{}, err
	}
	content := newWorkItemRecord(id, draft, path, works)
	if err := writeNewFile(path, content); err != nil {
		return WorkItemRecord{}, err
	}
	if err := listCommitment(result.Project.Source, draft.Title, path); err != nil {
		_ = os.Remove(path)
		return WorkItemRecord{}, err
	}
	record := WorkItemRecord{ProjectID: project.ID, WorkItemID: id, Path: relativeRecord(project.BundlePath, path), TicketSHA256: digestBytes(content), Draft: draft}
	now := s.state.now().UTC()
	if err := saveReceipt(ctx, tx, current.UserID, "create_work_item", request.RequestID, hash, record, false, now); err != nil {
		return WorkItemRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "commit work item", err)
	}
	return record, nil
}

// UpdateWorkItem rewrites a work item's title, description, requirements and
// dependencies through an expected-revision write. A build freezes the item.
func (s *Service) UpdateWorkItem(ctx context.Context, actor Principal, request UpdateWorkItemRequest) (WorkItemRecord, error) {
	if err := requireRequestID(request.RequestID); err != nil {
		return WorkItemRecord{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "start work item edit", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, request.ProjectID, CapabilityCoordinate)
	if err != nil {
		return WorkItemRecord{}, err
	}
	saved, ok, hash, err := receipt[WorkItemRecord](ctx, tx, current.UserID, "update_work_item", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	project, err := readProject(ctx, tx, request.ProjectID)
	if err != nil {
		return WorkItemRecord{}, err
	}
	result, err := s.orientProject(ctx, project)
	if err != nil {
		return WorkItemRecord{}, err
	}
	work, err := selectWork(result, request.WorkItemID)
	if err != nil {
		return WorkItemRecord{}, err
	}
	if err := requireNoActiveJob(ctx, tx, project.ID, request.WorkItemID); err != nil {
		return WorkItemRecord{}, err
	}
	works := workItemsByID(result)
	draft, err := normalizeDraft(request.Draft, works, request.WorkItemID)
	if err != nil {
		return WorkItemRecord{}, err
	}
	raw, err := os.ReadFile(work.Source)
	if err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "read work item", err)
	}
	if digestBytes(raw) != request.IfTicketSHA256 {
		return WorkItemRecord{}, fault(CodeStale, "the work item changed since you opened it; reload it and edit again", nil)
	}
	now := s.state.now().UTC()
	updated, err := rewriteWorkItem(raw, work.Source, draft, works)
	if err != nil {
		return WorkItemRecord{}, err
	}
	if updated, err = appendComment(updated, work.Source, fmt.Sprintf("- %s: %s edited the work item in Waymark.\n", now.Format("2006-01-02"), current.Username)); err != nil {
		return WorkItemRecord{}, err
	}
	if err := writeFileCAS(work.Source, raw, updated); err != nil {
		return WorkItemRecord{}, err
	}
	record := WorkItemRecord{ProjectID: project.ID, WorkItemID: request.WorkItemID, Path: relativeRecord(project.BundlePath, work.Source), TicketSHA256: digestBytes(updated), Draft: draft}
	if err := saveReceipt(ctx, tx, current.UserID, "update_work_item", request.RequestID, hash, record, false, now); err != nil {
		return WorkItemRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return WorkItemRecord{}, fault(CodeInternal, "commit work item edit", err)
	}
	return record, nil
}

// ScoreDraftCriteria asks the decision model whether requirements a person is still
// writing name checkable results. It is advisory, like every decision-model answer.
func (s *Service) ScoreDraftCriteria(ctx context.Context, actor Principal, projectID ProjectID, title string, criteria []string) (*CriterionSignals, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return nil, err
	}
	cleaned := []string{}
	for _, criterion := range criteria {
		if criterion = oneLine(criterion); criterion != "" && utf8.RuneCountInString(criterion) <= maxRefinementRunes {
			cleaned = append(cleaned, criterion)
		}
	}
	if len(cleaned) > maxRefinedCriteria {
		cleaned = cleaned[:maxRefinedCriteria]
	}
	return s.scoreCriteria(ctx, oneLine(title), cleaned, true), nil
}

func (s *Service) orientProject(ctx context.Context, project Project) (orientation.Result, error) {
	result, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return orientation.Result{}, fault(CodeInternal, "read the project's records", err)
	}
	return result, nil
}

func workItemsByID(result orientation.Result) map[string]orientation.WorkItem {
	works := map[string]orientation.WorkItem{}
	for _, work := range result.WorkItems {
		if work.ID != nil && !work.IdentityAmbiguous {
			works[*work.ID] = work
		}
	}
	return works
}

func normalizeDraft(draft WorkItemDraft, works map[string]orientation.WorkItem, self string) (WorkItemDraft, error) {
	cleaned := WorkItemDraft{Title: oneLine(draft.Title), Description: strings.TrimSpace(draft.Description), Criteria: []string{}, BlockedBy: []string{}}
	if cleaned.Title == "" || utf8.RuneCountInString(cleaned.Title) > maxWorkItemTitleRunes {
		return WorkItemDraft{}, fault(CodeInvalid, fmt.Sprintf("a work item needs a title of at most %d characters", maxWorkItemTitleRunes), nil)
	}
	if utf8.RuneCountInString(cleaned.Description) > maxWorkItemDescriptionRunes {
		return WorkItemDraft{}, fault(CodeInvalid, fmt.Sprintf("the description is limited to %d characters", maxWorkItemDescriptionRunes), nil)
	}
	for _, line := range strings.Split(cleaned.Description, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return WorkItemDraft{}, fault(CodeInvalid, "the description cannot contain headings, which would start sections of the record", nil)
		}
	}
	for _, criterion := range draft.Criteria {
		if criterion = joinLines(criterion); criterion != "" {
			if utf8.RuneCountInString(criterion) > maxRefinementRunes {
				return WorkItemDraft{}, fault(CodeInvalid, "a requirement is too long", nil)
			}
			cleaned.Criteria = append(cleaned.Criteria, criterion)
		}
	}
	if len(cleaned.Criteria) == 0 || len(cleaned.Criteria) > maxRefinedCriteria {
		return WorkItemDraft{}, fault(CodeInvalid, fmt.Sprintf("a work item needs 1 to %d requirements", maxRefinedCriteria), nil)
	}
	for _, id := range draft.BlockedBy {
		if _, known := works[id]; !known || id == self {
			return WorkItemDraft{}, fault(CodeInvalid, "a work item can only depend on other work items in its project: "+id, nil)
		}
		if !slices.Contains(cleaned.BlockedBy, id) {
			cleaned.BlockedBy = append(cleaned.BlockedBy, id)
		}
	}
	return cleaned, nil
}

// joinLines puts a requirement on one line without collapsing the spaces inside it,
// which an example such as ReverseWords("a  b") depends on.
func joinLines(text string) string {
	return strings.TrimSpace(strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(text))
}

func newWorkItemRecord(id string, draft WorkItemDraft, path string, works map[string]orientation.WorkItem) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntype: WorkItem\nid: %s\ntitle: %s\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n", id, yamlString(draft.Title))
	if draft.Description != "" {
		b.WriteString(draft.Description + "\n\n")
	}
	b.WriteString("## Acceptance criteria\n\n")
	for _, criterion := range draft.Criteria {
		b.WriteString("- " + criterion + "\n")
	}
	b.WriteString("\n## Blocked by\n\n" + strings.Join(blockerLines(draft.BlockedBy, path, works), "\n") + "\n")
	b.WriteString("\n## Blocked by decisions\n\nNone\n\n## Acceptance\n")
	return []byte(b.String())
}

// blockerLines links each prerequisite by its title, relative to the record's folder.
func blockerLines(ids []string, path string, works map[string]orientation.WorkItem) []string {
	if len(ids) == 0 {
		return []string{"None"}
	}
	lines := []string{}
	for _, id := range ids {
		work := works[id]
		title := id
		if work.Title != nil {
			title = *work.Title
		}
		lines = append(lines, "- ["+escapeLinkText(title)+"]("+relativeLink(path, work.Source)+")")
	}
	return lines
}

// rewriteWorkItem replaces the title, the description and the two sections a person
// edits, and keeps the rest of the record as it was.
func rewriteWorkItem(raw []byte, path string, draft WorkItemDraft, works map[string]orientation.WorkItem) ([]byte, error) {
	updated, err := setFrontmatterTitle(raw, draft.Title)
	if err != nil {
		return nil, err
	}
	updated, err = setDescription(updated, draft.Description)
	if err != nil {
		return nil, err
	}
	// Keep checkbox items when the record uses them, as refinement does.
	bullet := "- "
	if document, err := recordread.ParseDocument(updated); err == nil {
		for _, section := range recordread.Sections(document.Body, path, map[string]string{"Acceptance criteria": "criteria"}).Sections {
			if strings.Contains(strings.ToLower(section.Content), "- [ ]") || strings.Contains(strings.ToLower(section.Content), "- [x]") {
				bullet = "- [ ] "
			}
		}
	}
	criteria := make([]string, len(draft.Criteria))
	for index, criterion := range draft.Criteria {
		criteria[index] = bullet + criterion
	}
	if updated, err = replaceSection(updated, path, "Acceptance criteria", criteria); err != nil {
		return nil, err
	}
	return replaceSection(updated, path, "Blocked by", blockerLines(draft.BlockedBy, path, works))
}

func setFrontmatterTitle(raw []byte, title string) ([]byte, error) {
	document, err := recordread.ParseDocument(raw)
	if err != nil || !bytes.HasSuffix(raw, document.Body) {
		return nil, fault(CodeInvalid, "work item cannot be parsed for rewriting", err)
	}
	front := raw[:len(raw)-len(document.Body)]
	lines := strings.SplitAfter(string(front), "\n")
	for index, line := range lines {
		if strings.HasPrefix(line, "title:") {
			lines[index] = "title: " + yamlString(title) + "\n"
			return append([]byte(strings.Join(lines, "")), document.Body...), nil
		}
	}
	return nil, fault(CodeInvalid, "work item has no title to rewrite; edit it by hand", nil)
}

// ticketDescription is the prose between a record's front matter and its first heading.
func ticketDescription(raw []byte) string {
	document, err := recordread.ParseDocument(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(document.Body[:firstHeading(document.Body)]))
}

func setDescription(raw []byte, description string) ([]byte, error) {
	document, err := recordread.ParseDocument(raw)
	if err != nil || !bytes.HasSuffix(raw, document.Body) {
		return nil, fault(CodeInvalid, "work item cannot be parsed for rewriting", err)
	}
	offset := len(raw) - len(document.Body)
	end := offset + firstHeading(document.Body)
	prose := "\n"
	if description != "" {
		prose = "\n" + description + "\n\n"
	}
	return append(append(append([]byte{}, raw[:offset]...), prose...), raw[end:]...), nil
}

func firstHeading(body []byte) int {
	position := 0
	for _, line := range bytes.SplitAfter(body, []byte("\n")) {
		if bytes.HasPrefix(bytes.TrimLeft(line, " "), []byte("#")) {
			return position
		}
		position += len(line)
	}
	return len(body)
}

// replaceSection replaces the content of the record's single section with that name.
func replaceSection(raw []byte, path, name string, lines []string) ([]byte, error) {
	document, err := recordread.ParseDocument(raw)
	if err != nil || !bytes.HasSuffix(raw, document.Body) {
		return nil, fault(CodeInvalid, "work item cannot be parsed for rewriting", err)
	}
	offset := len(raw) - len(document.Body)
	sections := recordread.Sections(document.Body, path, map[string]string{name: "edited"}).Sections
	if len(sections) != 1 {
		return nil, fault(CodeInvalid, "work item must have exactly one "+name+" section to rewrite; edit it by hand", nil)
	}
	start, end := offset+sections[0].ContentStart, offset+sections[0].End
	block := "\n" + strings.Join(lines, "\n") + "\n\n"
	return append(append(append([]byte{}, raw[:start]...), block...), raw[end:]...), nil
}

// listCommitment adds the new record to the project's Current commitments section,
// creating the section when the project has none.
func listCommitment(manifest, title, path string) error {
	raw, err := os.ReadFile(manifest)
	if err != nil {
		return fault(CodeInternal, "read the project record", err)
	}
	document, err := recordread.ParseDocument(raw)
	if err != nil || !bytes.HasSuffix(raw, document.Body) {
		return fault(CodeInvalid, "the project record cannot be parsed", err)
	}
	line := "- [" + escapeLinkText(title) + "](" + relativeLink(manifest, path) + ")"
	offset := len(raw) - len(document.Body)
	sections := recordread.Sections(document.Body, manifest, map[string]string{"Current commitments": "commitment"}).Sections
	var updated []byte
	switch len(sections) {
	case 0:
		updated = append(append([]byte{}, bytes.TrimRight(raw, "\n")...), "\n\n## Current commitments\n\n"+line+"\n"...)
	case 1:
		section := sections[0]
		content := strings.TrimSpace(section.Content)
		block := line
		if content != "" && !strings.EqualFold(content, "none") {
			block = content + "\n" + line
		}
		start, end := offset+section.ContentStart, offset+section.End
		updated = append(append(append([]byte{}, raw[:start]...), "\n"+block+"\n\n"...), raw[end:]...)
	default:
		return fault(CodeInvalid, "the project record has more than one Current commitments section; list the work by hand", nil)
	}
	return writeFileCAS(manifest, raw, updated)
}

func writeNewFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fault(CodeConflict, "write the new work item", err)
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		_ = os.Remove(path)
		return fault(CodeInternal, "write the new work item", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fault(CodeInternal, "write the new work item", err)
	}
	return nil
}

func freeRecordPath(directory, slug string) (string, error) {
	for attempt := 1; attempt <= 100; attempt++ {
		name := slug + ".md"
		if attempt > 1 {
			name = fmt.Sprintf("%s-%d.md", slug, attempt)
		}
		path := filepath.Join(directory, name)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			return path, nil
		}
	}
	return "", fault(CodeConflict, "no free file name for the new work item", nil)
}

func slugOf(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
		if b.Len() >= 60 {
			break
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		return "work-item"
	}
	return slug
}

// yamlString writes a title as a YAML scalar, quoting it when YAML would misread it.
func yamlString(value string) string {
	if value == "" || strings.ContainsAny(value, ":#'\"{}[],&*!|>%@`\\") || strings.ContainsAny(value[:1], "-?") {
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
	return value
}

func escapeLinkText(text string) string {
	return strings.NewReplacer("[", "\\[", "]", "\\]").Replace(text)
}

func relativeLink(from, to string) string {
	link, err := filepath.Rel(filepath.Dir(from), to)
	if err != nil {
		return filepath.ToSlash(to)
	}
	return filepath.ToSlash(link)
}

func relativeRecord(bundle, path string) string {
	link, err := filepath.Rel(bundle, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(link)
}

// newRecordID makes a random version 4 UUID, the identity form project records use.
func newRecordID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
