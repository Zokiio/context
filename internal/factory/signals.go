package factory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/decision"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
)

// Signal states. Observed means every criterion has an answer; partial means some
// answers are missing; unavailable means the decision model could not be reached;
// not_observed means admission found no cached answers to copy.
const (
	SignalsObserved    = "observed"
	SignalsPartial     = "partial"
	SignalsUnavailable = "unavailable"
	SignalsNotObserved = "not_observed"
)

type CriterionSignals struct {
	Model        string            `json:"model"`
	ModelDigest  string            `json:"modelDigest,omitempty"`
	Question     string            `json:"question"`
	Threshold    float64           `json:"threshold"`
	Status       string            `json:"status"`
	Message      string            `json:"message,omitempty"`
	Criteria     []CriterionSignal `json:"criteria"`
	FlaggedCount int               `json:"flaggedCount"`
	ObservedAt   time.Time         `json:"observedAt"`
	AdvisoryOnly bool              `json:"advisoryOnly"`
}

type CriterionSignal struct {
	Text string `json:"text"`
	// Checkable is the model's probability that the criterion names a checkable result.
	// It is uncalibrated: compare criteria with each other rather than trusting the
	// absolute value.
	Checkable *float64 `json:"checkable,omitempty"`
	Flagged   bool     `json:"flagged"`
}

const maxSignalWorkItems = 50

var criterionItem = regexp.MustCompile(`^\s{0,3}(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?(.*\S)\s*$`)

// acceptanceCriteria reads the authored list items under a work item's Acceptance
// criteria section. Indented continuation lines belong to the item above them.
func acceptanceCriteria(source string) ([]string, error) {
	raw, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	document, err := recordread.ParseDocument(raw)
	if err != nil {
		return nil, err
	}
	sections := recordread.Sections(document.Body, source, map[string]string{"Acceptance criteria": "criteria"})
	var criteria []string
	for _, section := range sections.Sections {
		for _, line := range strings.Split(section.Content, "\n") {
			if match := criterionItem.FindStringSubmatch(line); match != nil {
				criteria = append(criteria, strings.TrimSpace(match[1]))
			} else if len(criteria) > 0 && strings.TrimSpace(line) != "" && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
				criteria[len(criteria)-1] += " " + strings.TrimSpace(line)
			}
		}
	}
	return criteria, nil
}

func signalKey(title, criterion string) string {
	sum := sha256.Sum256([]byte(title + "\x00" + criterion))
	return hex.EncodeToString(sum[:])
}

// criterionSignals answers the checkability question for each criterion of one work
// item. With observe false it only copies cached answers and never calls the model.
func (s *Service) criterionSignals(ctx context.Context, title *string, source string, observe bool) *CriterionSignals {
	client := s.state.decision
	if client == nil {
		return nil
	}
	name := ""
	if title != nil {
		name = *title
	}
	result := &CriterionSignals{Model: client.Model, Question: decision.CheckableQuestion, Threshold: s.state.decisionThreshold, Criteria: []CriterionSignal{}, ObservedAt: s.state.now().UTC(), AdvisoryOnly: true}
	criteria, err := acceptanceCriteria(source)
	if err != nil {
		result.Status, result.Message = SignalsUnavailable, "acceptance criteria could not be read"
		return result
	}
	observed, failed := 0, 0
	var lastErr error
	for _, text := range criteria {
		signal := CriterionSignal{Text: text}
		key := signalKey(name, text)
		var probability float64
		var digest string
		err := s.state.db.QueryRowContext(ctx, `SELECT checkable, model_digest FROM criterion_signals WHERE model=? AND question=? AND input_sha256=?`, client.Model, decision.CheckableQuestion, key).Scan(&probability, &digest)
		switch {
		case err == nil:
		case errors.Is(err, sql.ErrNoRows) && observe && failed == 0:
			probability, err = client.Checkable(ctx, name, text)
			if err == nil {
				digest = s.decisionDigest(ctx)
				_, _ = s.state.db.ExecContext(ctx, `INSERT OR REPLACE INTO criterion_signals(model, question, input_sha256, checkable, model_digest, observed_at) VALUES(?,?,?,?,?,?)`, client.Model, decision.CheckableQuestion, key, probability, digest, formatTime(s.state.now()))
			} else {
				failed++
				lastErr = err
			}
		default:
			failed++
		}
		if err == nil {
			value := probability
			signal.Checkable = &value
			signal.Flagged = probability < result.Threshold
			if signal.Flagged {
				result.FlaggedCount++
			}
			if result.ModelDigest == "" {
				result.ModelDigest = digest
			}
			observed++
		}
		result.Criteria = append(result.Criteria, signal)
	}
	switch {
	case observed == len(criteria):
		result.Status = SignalsObserved
	case observed > 0:
		result.Status = SignalsPartial
	case observe && lastErr != nil:
		result.Status, result.Message = SignalsUnavailable, "decision model unavailable: "+lastErr.Error()
	default:
		result.Status = SignalsNotObserved
	}
	return result
}

func (s *Service) cachedCriterionSignals(ctx context.Context, title *string, source string) *CriterionSignals {
	return s.criterionSignals(ctx, title, source, false)
}

func (s *Service) decisionDigest(ctx context.Context) string {
	s.state.decisionMu.Lock()
	defer s.state.decisionMu.Unlock()
	if s.state.decisionModelDigest == "" {
		if digest, err := s.state.decision.ModelDigest(ctx); err == nil {
			s.state.decisionModelDigest = digest
		}
	}
	return s.state.decisionModelDigest
}

// orientationSignals observes the open work items in an orientation result, so a
// person can rewrite vague criteria before admission freezes them.
func (s *Service) orientationSignals(ctx context.Context, result orientation.Result) map[string]*CriterionSignals {
	if s.state.decision == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	signals := map[string]*CriterionSignals{}
	for _, work := range result.WorkItems {
		if len(signals) >= maxSignalWorkItems {
			break
		}
		if work.ID == nil || (work.Execution != nil && (*work.Execution == "completed" || *work.Execution == "cancelled")) {
			continue
		}
		signals[*work.ID] = s.criterionSignals(ctx, work.Title, work.Source, true)
	}
	return signals
}
