package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Zokiio/context/internal/orientation"
)

type attentionCause struct {
	kind      string
	check     string
	code      string
	severity  string
	messages  []string
	sources   []attentionSource
	affected  []orientation.Reference
	affecteds map[string]bool
}

type attentionSource struct {
	path string
	from string
	link string
}

type compactOrientationView struct {
	recordsRoot   string
	rootKnown     bool
	ambiguousWork map[string]bool
}

func renderOrientation(output io.Writer, result orientation.Result) error {
	var text strings.Builder
	view := newCompactOrientationView(result)
	writeCompactProject(&text, result, view)
	writeCompactGoals(&text, result)
	writeCompactCommitments(&text, result, view)
	writeCompactAttention(&text, result, view)
	writeCompactWork(&text, "In progress", result.InProgress, view)
	if result.InventoryComplete {
		writeCompactWork(&text, "New pickup shortlist", result.Shortlist, view)
	} else {
		fmt.Fprintln(&text, "\nNew pickup shortlist: suppressed because inventory is partial")
	}
	fmt.Fprintln(&text, "Task context: run ctx context with a work item's source as --ticket, using the same scope and --allow-source arguments.")
	fmt.Fprintln(&text, "Continuation: run ctx resume --ticket <work-item-source> with the same scope and --allow-source arguments; direct --bundle access also requires --checkout <working-directory>.")
	fmt.Fprintln(&text, "Expanded detail: rerun ctx orient with the same scope and --detail.")
	_, err := io.WriteString(output, text.String())
	return err
}

func newCompactOrientationView(result orientation.Result) compactOrientationView {
	view := compactOrientationView{ambiguousWork: make(map[string]bool, len(result.WorkItems))}
	for _, item := range result.WorkItems {
		view.ambiguousWork[item.Source] = item.IdentityAmbiguous
	}
	if result.Project == nil || !filepath.IsAbs(result.Project.Source) || filepath.Base(result.Project.Source) != "project.md" {
		return view
	}
	view.recordsRoot = filepath.Dir(filepath.Clean(result.Project.Source))
	view.rootKnown = true
	return view
}

func writeCompactProject(output *strings.Builder, result orientation.Result, view compactOrientationView) {
	if result.Project == nil {
		fmt.Fprintln(output, "Project: unknown")
	} else {
		fmt.Fprintf(output, "Project: %s [%s]\n", result.Project.Title, result.Project.ID)
	}
	if view.rootKnown {
		fmt.Fprintf(output, "Record store: %s\n", view.recordsRoot)
	} else {
		fmt.Fprintln(output, "Record store: unknown")
	}
	fmt.Fprintf(output, "Evaluation: %s; inventory: %s\n", completeness(result.Complete), completeness(result.InventoryComplete))
}

func writeCompactGoals(output *strings.Builder, result orientation.Result) {
	fmt.Fprintln(output)
	switch {
	case result.Project == nil || !result.Project.GoalsKnown:
		fmt.Fprintln(output, "Goals: unknown")
	case len(result.Goals) == 0:
		fmt.Fprintln(output, "Goals: none")
	default:
		fmt.Fprintln(output, "Goals:")
	}
	for _, goal := range result.Goals {
		fmt.Fprint(output, goal.Text)
		if !strings.HasSuffix(goal.Text, "\n") {
			fmt.Fprintln(output)
		}
	}
}

func writeCompactCommitments(output *strings.Builder, result orientation.Result, view compactOrientationView) {
	fmt.Fprintln(output)
	known := result.Project != nil && result.Project.CommitmentsKnown
	if !known {
		fmt.Fprintln(output, "Current commitments: unknown")
		if len(result.CurrentCommitments) > 0 {
			fmt.Fprintln(output, "  Known references:")
		}
	} else if len(result.CurrentCommitments) == 0 {
		fmt.Fprintln(output, "Current commitments: none")
		return
	} else {
		fmt.Fprintln(output, "Current commitments:")
	}
	items := workItemsBySource(result.WorkItems)
	for _, ref := range result.CurrentCommitments {
		fmt.Fprintf(output, "  %s\n", view.workReference(ref, true))
		item, found := items[ref.Path]
		if !found {
			fmt.Fprintln(output, "    execution: unknown; readiness: unknown")
			continue
		}
		fmt.Fprintf(output, "    execution: %s; readiness: %s\n", knownString(item.Execution), knownStatus(item.Readiness))
	}
}

func writeCompactAttention(output *strings.Builder, result orientation.Result, view compactOrientationView) {
	fmt.Fprintln(output, "\nNeeds attention:")
	causes := compactAttentionCauses(result)
	if len(causes) == 0 {
		fmt.Fprintln(output, "  none")
		return
	}
	for _, cause := range causes {
		label := cause.code
		if cause.kind == "check" {
			label = cause.check + "/" + cause.code
		} else if cause.severity != "" {
			label = cause.severity + " " + cause.code
		}
		fmt.Fprintf(output, "  %s:\n", label)
		for _, message := range cause.messages {
			fmt.Fprintf(output, "    %s\n", message)
		}
		writeCauseSource(output, cause, view)
		writeCompactReferences(output, "    Affected work", cause.affected, view, false)
	}
}

func compactAttentionCauses(result orientation.Result) []*attentionCause {
	causes := []*attentionCause{}
	byKey := map[string]*attentionCause{}
	items := workItemsBySource(result.WorkItems)
	orderedWork := make([]orientation.WorkItem, 0, len(result.CurrentCommitments))
	for _, commitment := range result.CurrentCommitments {
		if item, ok := items[commitment.Path]; ok {
			orderedWork = append(orderedWork, item)
		}
	}
	for _, item := range orderedWork {
		ref := orientation.Reference{Path: item.Source, ID: item.ID, Title: item.Title}
		for _, check := range item.Checks {
			if check.Status == "pass" {
				continue
			}
			if len(check.Reasons) == 0 {
				finding := orientation.Finding{Code: "check_" + knownStatus(check.Status), Message: "check is " + knownStatus(check.Status), Path: item.Source}
				addAttentionCause(&causes, byKey, "check", check.Name, "", finding, &ref)
				continue
			}
			for _, finding := range check.Reasons {
				if routinePassedFinding(finding.Code) {
					continue
				}
				addAttentionCause(&causes, byKey, "check", check.Name, "", finding, &ref)
			}
		}
		for _, finding := range item.ExclusionReasons {
			if eligibilityFindingNeedsAttention(finding.Code) {
				addAttentionCause(&causes, byKey, "check", "eligibility", "", finding, &ref)
			}
		}
		if item.Acceptance != nil && item.Acceptance.CheckStatus != "pass" {
			if len(item.Acceptance.Reasons) == 0 {
				finding := orientation.Finding{Code: "acceptance_" + knownStatus(item.Acceptance.CheckStatus), Message: "acceptance check is " + knownStatus(item.Acceptance.CheckStatus), Path: item.Source}
				addAttentionCause(&causes, byKey, "check", "acceptance", "", finding, &ref)
			} else {
				for _, finding := range item.Acceptance.Reasons {
					addAttentionCause(&causes, byKey, "check", "acceptance", "", finding, &ref)
				}
			}
		}
	}
	for _, decision := range result.Decisions {
		if decision.CheckStatus == "pass" {
			continue
		}
		projectReference := false
		for _, ref := range decision.References {
			if result.Project != nil && ref.From == result.Project.Source {
				projectReference = true
				break
			}
		}
		affected := make([]orientation.Reference, 0, len(decision.AffectedWork))
		decisionWork := make(map[string]orientation.Reference, len(decision.AffectedWork))
		for _, ref := range decision.AffectedWork {
			decisionWork[ref.Path] = ref
		}
		for _, commitment := range result.CurrentCommitments {
			if ref, found := decisionWork[commitment.Path]; found {
				affected = append(affected, ref)
			}
		}
		if !projectReference && len(affected) == 0 {
			continue
		}
		for _, finding := range decision.Reasons {
			if routinePassedFinding(finding.Code) {
				continue
			}
			if len(affected) == 0 {
				addAttentionCause(&causes, byKey, "check", "decision", "", finding, nil)
				continue
			}
			for index := range affected {
				addAttentionCause(&causes, byKey, "check", "decision", "", finding, &affected[index])
			}
		}
	}
	for _, diagnostic := range result.Diagnostics {
		finding := orientation.Finding{Code: diagnostic.Code, Message: diagnostic.Message, Path: diagnostic.Path, From: diagnostic.From, Link: diagnostic.Link}
		var affected *orientation.Reference
		if item, ok := items[firstNonempty(diagnostic.From, diagnostic.Path)]; ok {
			ref := orientation.Reference{Path: item.Source, ID: item.ID, Title: item.Title}
			affected = &ref
		}
		addAttentionCause(&causes, byKey, "diagnostic", "", diagnostic.Severity, finding, affected)
	}
	return causes
}

func addAttentionCause(causes *[]*attentionCause, byKey map[string]*attentionCause, kind, check, severity string, finding orientation.Finding, affected *orientation.Reference) {
	identity := attentionIdentity(kind, finding)
	key := strings.Join([]string{kind, check, severity, finding.Code, identity}, "\x00")
	cause := byKey[key]
	if cause == nil {
		cause = &attentionCause{kind: kind, check: check, code: finding.Code, severity: severity, affecteds: map[string]bool{}}
		byKey[key] = cause
		*causes = append(*causes, cause)
	}
	if !contains(cause.messages, finding.Message) {
		cause.messages = append(cause.messages, finding.Message)
	}
	if affected != nil && !cause.affecteds[affected.Path] {
		cause.affected = append(cause.affected, *affected)
		cause.affecteds[affected.Path] = true
	}
	source := attentionSource{path: finding.Path, from: finding.From, link: finding.Link}
	if source.path != "" || source.from != "" || source.link != "" {
		for _, existing := range cause.sources {
			if existing == source {
				return
			}
		}
		cause.sources = append(cause.sources, source)
	}
}

func attentionIdentity(kind string, finding orientation.Finding) string {
	// Some relationship diagnostics use the referring record as Path because no
	// target can be resolved. In that case the authored relationship is the only
	// stable cause identity. Resolved targets continue to group by their path.
	if kind == "diagnostic" && finding.Path == finding.From && finding.Link != "" {
		switch finding.Code {
		case "unsupported_source", "unresolved_reference":
			return "relationship\x00" + finding.From + "\x00" + finding.Link
		}
	}
	if finding.Path != "" {
		return "source\x00" + finding.Path
	}
	return "relationship\x00" + finding.From + "\x00" + finding.Link
}

func writeCauseSource(output *strings.Builder, cause *attentionCause, view compactOrientationView) {
	written := map[string]bool{}
	for _, source := range cause.sources {
		line := ""
		if source.path != "" {
			line = "    source: " + view.path(source.path)
			if source.path == source.from && source.link != "" {
				line += relationshipDetails(view.path(source.from), source.link)
			}
		} else {
			line = "    relationship" + relationshipDetails(view.path(source.from), source.link)
		}
		if !written[line] {
			fmt.Fprintln(output, line)
			written[line] = true
		}
	}
}

func writeCompactWork(output *strings.Builder, heading string, refs []orientation.Reference, view compactOrientationView) {
	fmt.Fprintln(output)
	writeCompactReferences(output, heading, refs, view, true)
}

func writeCompactReferences(output *strings.Builder, heading string, refs []orientation.Reference, view compactOrientationView, includePath bool) {
	if len(refs) == 0 {
		fmt.Fprintf(output, "%s: none\n", heading)
		return
	}
	fmt.Fprintf(output, "%s:\n", heading)
	indent := strings.Repeat(" ", len(heading)-len(strings.TrimLeft(heading, " "))+2)
	for _, ref := range refs {
		fmt.Fprintf(output, "%s%s\n", indent, view.workReference(ref, includePath))
	}
}

func (view compactOrientationView) workReference(ref orientation.Reference, includePath bool) string {
	value := fmt.Sprintf("%s [%s]", knownString(ref.Title), knownString(ref.ID))
	if ref.Path != "" && (includePath || missingReferenceIdentity(ref) || view.ambiguousWork[ref.Path]) {
		value += " " + view.path(ref.Path)
	}
	return value
}

func missingReferenceIdentity(ref orientation.Reference) bool {
	return ref.ID == nil || strings.TrimSpace(*ref.ID) == "" || ref.Title == nil || strings.TrimSpace(*ref.Title) == ""
}

func (view compactOrientationView) path(path string) string {
	if path == "" || !view.rootKnown || !filepath.IsAbs(path) {
		return path
	}
	relative, err := filepath.Rel(view.recordsRoot, filepath.Clean(path))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return path
	}
	return relative
}

func workItemsBySource(items []orientation.WorkItem) map[string]orientation.WorkItem {
	result := make(map[string]orientation.WorkItem, len(items))
	for _, item := range items {
		result[item.Source] = item
	}
	return result
}

func knownStatus(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func routinePassedFinding(code string) bool {
	switch code {
	case "acceptance_criteria_present", "required_context_available", "no_dependencies", "completed_dependency", "no_blocking_decisions", "decision_resolved", "acceptance_valid":
		return true
	default:
		return false
	}
}

func eligibilityFindingNeedsAttention(code string) bool {
	switch code {
	case "invalid_identity", "duplicate_identity", "project_identity_unknown", "inventory_incomplete", "execution_unknown", "commitment_unknown", "triage_unknown", "readiness_unknown":
		return true
	default:
		return false
	}
}
