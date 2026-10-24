package factory

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxAgentInstructionRunes bounds each agent's instructions, which travel in every prompt.
const MaxAgentInstructionRunes = 4000

// AgentInstructions are a project's own guidance for each agent. Each prompt adds them
// after the agent's built-in rules, which win where the two disagree, so the service's
// guarantees never depend on them. A run keeps the text in force when it started.
type AgentInstructions struct {
	Refiner    string     `json:"refiner,omitempty"`
	TestAuthor string     `json:"testAuthor,omitempty"`
	Builder    string     `json:"builder,omitempty"`
	SetBy      *UserID    `json:"setBy,omitempty"`
	SetAt      *time.Time `json:"setAt,omitempty"`
}

type SetAgentInstructionsRequest struct {
	RequestID  string    `json:"requestId"`
	ProjectID  ProjectID `json:"projectId"`
	Refiner    string    `json:"refiner"`
	TestAuthor string    `json:"testAuthor"`
	Builder    string    `json:"builder"`
}

// SetAgentInstructions replaces a project's instructions for its agents. Only a person
// who administers the project can change what its agents are told.
func (s *Service) SetAgentInstructions(ctx context.Context, actor Principal, request SetAgentInstructionsRequest) (Project, error) {
	instructions := AgentInstructions{Refiner: strings.TrimSpace(request.Refiner), TestAuthor: strings.TrimSpace(request.TestAuthor), Builder: strings.TrimSpace(request.Builder)}
	for _, text := range []string{instructions.Refiner, instructions.TestAuthor, instructions.Builder} {
		if utf8.RuneCountInString(text) > MaxAgentInstructionRunes {
			return Project{}, fault(CodeInvalid, fmt.Sprintf("each agent's instructions are limited to %d characters", MaxAgentInstructionRunes), nil)
		}
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, fault(CodeInternal, "start agent instructions", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, request.ProjectID, CapabilityProjectAdmin)
	if err != nil {
		return Project{}, err
	}
	if current.Kind != UserKindHuman {
		return Project{}, fault(CodeForbidden, "only a person can change what the agents are told", nil)
	}
	saved, ok, hash, err := receipt[Project](ctx, tx, current.UserID, "set_agent_instructions", request.RequestID, request)
	if err != nil || ok {
		return saved, err
	}
	project, err := readProject(ctx, tx, request.ProjectID)
	if err != nil {
		return Project{}, err
	}
	now := s.state.now().UTC()
	instructions.SetBy, instructions.SetAt = &current.UserID, &now
	project.Policy.Instructions = instructions
	if err := saveProjectPolicy(ctx, tx, project); err != nil {
		return Project{}, err
	}
	if err := saveReceipt(ctx, tx, current.UserID, "set_agent_instructions", request.RequestID, hash, project, false, now); err != nil {
		return Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return Project{}, fault(CodeInternal, "commit agent instructions", err)
	}
	return project, nil
}
