package orientation

import (
	"sort"
	"strings"
)

type dependencyNode struct {
	record      *record
	declaration Check
	edges       []*dependencyLink
	closure     []*dependencyLink
	reachable   map[string]bool
	check       Check
}

type dependencyLink struct {
	selected *selection
	target   *dependencyNode
	basis    Check
	result   DependencyEdge
}

// Resolve authored edges from captured records, then collect finite closures.
// Independent closures keep shared prerequisites consistent across starting work.
func (e *evaluator) prepareDependencies() {
	e.dependencies = map[string]*dependencyNode{}
	for _, r := range e.recordOrder {
		if r.kind != "WorkItem" {
			continue
		}
		node := &dependencyNode{record: r, declaration: Check{Name: "dependencies", Status: "pass", Reasons: []Finding{}}}
		if !e.declaration(r, "Blocked by", true) {
			node.declaration.Status = "unknown"
			node.declaration.Reasons = append(node.declaration.Reasons, Finding{Code: "incomplete_relationship_declaration", Message: "Blocked by declaration is missing, malformed, or unavailable", Path: r.source.Path})
		}
		e.dependencies[r.source.Path] = node
	}
	facts, seenEdges := map[string]Check{}, map[string]bool{}
	for _, selected := range e.selections {
		if selected.link.Kind != "blocked_by" {
			continue
		}
		node := e.dependencies[selected.from]
		if node == nil {
			continue
		}
		key := selected.from + "\x00" + selected.reference.Path + "\x00" + selected.link.Link
		if seenEdges[key] {
			continue
		}
		seenEdges[key] = true
		link := &dependencyLink{selected: selected, basis: Check{Status: "pass", Reasons: []Finding{}}, result: DependencyEdge{From: Reference{Path: node.record.source.Path, ID: node.record.id, Title: node.record.title}, To: selected.reference, Reasons: []Finding{}}}
		node.edges = append(node.edges, link)
		target := selected.target
		if selected.source == nil || target == nil || target.kind != "WorkItem" || target.id == nil || target.title == nil || target.ambiguous {
			link.basis.Status = "unknown"
			link.basis.Reasons = append(link.basis.Reasons, Finding{Code: "dependency_unavailable", Message: "prerequisite must identify an available, unambiguous WorkItem", Path: selected.reference.Path, From: selected.from, Link: selected.link.Link})
			continue
		}
		link.target = e.dependencies[target.source.Path]
		if _, known := facts[target.source.Path]; !known {
			facts[target.source.Path] = e.prerequisiteFacts(target)
		}
		combineDependencyChecks(&link.basis, facts[target.source.Path], link.target.declaration)
		for index := range link.basis.Reasons {
			reason := &link.basis.Reasons[index]
			if reason.From == "" {
				reason.From, reason.Link = selected.from, selected.link.Link
			}
		}
	}
	for _, r := range e.recordOrder {
		node := e.dependencies[r.source.Path]
		if node == nil {
			continue
		}
		node.reachable = map[string]bool{}
		var visit func(*dependencyNode)
		visit = func(current *dependencyNode) {
			if node.reachable[current.record.source.Path] {
				return
			}
			node.reachable[current.record.source.Path] = true
			for _, link := range current.edges {
				node.closure = append(node.closure, link)
				if link.target != nil {
					visit(link.target)
				}
			}
		}
		visit(node)
	}
	e.markDependencyCycles()
	for _, r := range e.recordOrder {
		node := e.dependencies[r.source.Path]
		if node == nil {
			continue
		}
		node.check = Check{Name: "dependencies", Status: "pass", Reasons: []Finding{}}
		combineDependencyChecks(&node.check, node.declaration)
		for _, link := range node.closure {
			combineDependencyChecks(&node.check, link.basis)
		}
		if len(node.check.Reasons) == 0 {
			node.check.Reasons = append(node.check.Reasons, Finding{Code: "no_dependencies", Message: "Blocked by explicitly declares none", Path: r.source.Path})
		}
	}
	for _, r := range e.recordOrder {
		if node := e.dependencies[r.source.Path]; node != nil {
			for _, link := range node.edges {
				check := Check{Status: "pass", Reasons: []Finding{}}
				combineDependencyChecks(&check, link.basis)
				if link.target != nil {
					combineDependencyChecks(&check, link.target.check)
				}
				link.result.Status, link.result.Reasons = check.Status, check.Reasons
			}
		}
	}
}

// An edge participates in a cycle exactly when its target can reach its source.
// Mutual reachability also identifies the complete participating component,
// including edges through unfinished records and overlapping cycles.
func (e *evaluator) markDependencyCycles() {
	for _, r := range e.recordOrder {
		node := e.dependencies[r.source.Path]
		if node == nil {
			continue
		}
		message := ""
		for _, link := range node.edges {
			if link.target == nil || !link.target.reachable[r.source.Path] {
				continue
			}
			if message == "" {
				members := []*record{}
				for path := range node.reachable {
					if member := e.dependencies[path]; member != nil && member.reachable[r.source.Path] {
						members = append(members, member.record)
					}
				}
				sort.Slice(members, func(i, j int) bool {
					if stringValue(members[i].id) != stringValue(members[j].id) {
						return stringValue(members[i].id) < stringValue(members[j].id)
					}
					return members[i].source.Path < members[j].source.Path
				})
				identities := []string{}
				for _, member := range members {
					identities = append(identities, stringValue(member.id)+" ("+member.source.Path+")")
				}
				message = "dependency cycle involving " + strings.Join(identities, ", ")
			}
			link.result.Cycle = true
			link.basis.Status = "fail"
			link.basis.Reasons = append(link.basis.Reasons, Finding{Code: "dependency_cycle", Message: message, Path: link.result.To.Path, From: r.source.Path, Link: link.selected.link.Link})
			e.diagnose(Diagnostic{Code: "dependency_cycle", Severity: "warning", Message: message, Path: link.result.To.Path, From: r.source.Path, Link: link.selected.link.Link})
		}
	}
}

func (e *evaluator) prerequisiteFacts(r *record) Check {
	check := Check{Status: "pass", Reasons: []Finding{}}
	execution := e.enumField(r, "execution", "unstarted", "in-progress", "completed", "cancelled")
	code, message := "completed_dependency", "prerequisite is recorded as completed"
	switch stringValue(execution) {
	case "unstarted", "in-progress":
		check.Status, code, message = "fail", "unfinished_dependency", "prerequisite is not completed"
	case "cancelled":
		check.Status, code, message = "fail", "cancelled_dependency", "cancelled prerequisite cannot satisfy an edge"
	case "completed":
		accepted := e.evaluateAcceptance(r)
		combineDependencyChecks(&check, Check{Status: accepted.CheckStatus, Reasons: accepted.Reasons})
	default:
		check.Status, code, message = "unknown", "dependency_execution_unknown", "prerequisite execution state is unavailable"
	}
	check.Reasons = append(check.Reasons, Finding{Code: code, Message: message, Path: r.source.Path})
	combineDependencyChecks(&check, e.blockingDecisionCheck(r))
	return check
}

func combineDependencyChecks(result *Check, checks ...Check) {
	seen := map[Finding]bool{}
	for _, reason := range result.Reasons {
		seen[reason] = true
	}
	for _, check := range checks {
		result.Status = combineCheckStatus(result.Status, check.Status)
		for _, reason := range check.Reasons {
			if !seen[reason] {
				result.Reasons = append(result.Reasons, reason)
				seen[reason] = true
			}
		}
	}
}

func (e *evaluator) dependencyCheck(r *record) Check { return e.dependencies[r.source.Path].check }

func (e *evaluator) dependencyEdges(r *record) []DependencyEdge {
	edges := []DependencyEdge{}
	for _, link := range e.dependencies[r.source.Path].closure {
		edges = append(edges, link.result)
	}
	return edges
}
