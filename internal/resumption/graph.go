package resumption

import (
	"fmt"
	"sort"
)

// Graph validation remains useful after an incomplete read: a known cycle is
// still invalid, but an unread predecessor is not necessarily absent.
func evaluateRecoveryGraph(recovery *Recovery, names []string, enumerationComplete bool, findings *[]Diagnostic) {
	sort.SliceStable(recovery.Observations, func(i, j int) bool { return recovery.Observations[i].ID < recovery.Observations[j].ID })
	nodes := make(map[string]int, len(recovery.Observations))
	directories := make(map[string]bool, len(names))
	referenced := map[string]bool{}
	for _, name := range names {
		directories[name] = true
	}
	invalid := func(code, message string, observation *Observation, predecessor string) {
		recovery.GraphStatus = "invalid"
		*findings = append(*findings, Diagnostic{Code: code, Severity: "error", Message: message, Path: &observation.Source.Path, Link: optionalString(predecessor), ObservationID: &observation.ID})
	}
	for index := range recovery.Observations {
		observation := &recovery.Observations[index]
		if _, exists := nodes[observation.ID]; exists {
			invalid("recovery_identity_mismatch", "observation identity is duplicated", observation, "")
		}
		nodes[observation.ID] = index
	}
	for index := range recovery.Observations {
		observation := &recovery.Observations[index]
		for _, predecessor := range observation.Predecessors {
			if _, exists := nodes[predecessor]; exists {
				referenced[predecessor] = true
				continue
			}
			if enumerationComplete && !directories[predecessor] {
				invalid("recovery_predecessor_missing", fmt.Sprintf("predecessor %s has no observation directory", predecessor), observation, predecessor)
			}
		}
	}
	// An explicit stack avoids making the caller's history limit a recursion limit.
	type frame struct{ node, next int }
	state := make([]uint8, len(recovery.Observations))
	for start := range recovery.Observations {
		if state[start] != 0 {
			continue
		}
		state[start] = 1
		stack := []frame{{node: start}}
		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			observation := &recovery.Observations[top.node]
			if top.next == len(observation.Predecessors) {
				state[top.node] = 2
				stack = stack[:len(stack)-1]
				continue
			}
			predecessor := observation.Predecessors[top.next]
			top.next++
			next, exists := nodes[predecessor]
			if !exists {
				continue
			}
			switch state[next] {
			case 0:
				state[next] = 1
				stack = append(stack, frame{node: next})
			case 1:
				invalid("recovery_cycle", fmt.Sprintf("predecessor %s closes a recovery cycle", predecessor), observation, predecessor)
			}
		}
	}
	if !recovery.InventoryComplete || recovery.GraphStatus != "valid" {
		return
	}
	for _, observation := range recovery.Observations {
		if !referenced[observation.ID] {
			recovery.Candidates = append(recovery.Candidates, observation.ID)
		}
	}
}

// Messages are presentation, not diagnostic identity. Nil attribution stays
// distinct from an authored empty value in the normalized report.
func appendUniqueDiagnostics(existing, additional []Diagnostic) []Diagnostic {
	type scalar struct {
		value   string
		present bool
	}
	value := func(pointer *string) scalar {
		if pointer == nil {
			return scalar{}
		}
		return scalar{*pointer, true}
	}
	type key struct {
		code, severity                string
		path, from, link, observation scalar
	}
	seen := map[key]bool{}
	identity := func(d Diagnostic) key {
		return key{d.Code, d.Severity, value(d.Path), value(d.From), value(d.Link), value(d.ObservationID)}
	}
	for _, diagnostic := range existing {
		seen[identity(diagnostic)] = true
	}
	for _, diagnostic := range additional {
		key := identity(diagnostic)
		if !seen[key] {
			existing = append(existing, diagnostic)
			seen[key] = true
		}
	}
	return existing
}
