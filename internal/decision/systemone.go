// Package decision asks local decision models bounded, typed questions through
// Ollama's System One API. Answers are advisory observations, never gates.
package decision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CheckableQuestion versions the wording of the criterion question. Change it whenever
// the instructions or criteria below change, so cached answers are not reused.
const CheckableQuestion = "waymark.criterion-checkable.v1"

const DefaultURL = "http://127.0.0.1:11434"

type Client struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func New(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = DefaultURL
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Model: model, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

// Checkable returns the model's probability that the criterion states a concrete
// result someone could check as met or not met.
func (c *Client) Checkable(ctx context.Context, title, criterion string) (float64, error) {
	state := map[string]string{"criterion": criterion}
	if title != "" {
		state["ticket"] = title
	}
	body := map[string]any{
		"model":      c.Model,
		"state":      state,
		"keep_alive": "30m",
		"questions": map[string]any{
			"checkable": map[string]any{
				"type":         "noul",
				"instructions": "Does this acceptance criterion state a concrete result that someone could check as met or not met?",
				"criteria": map[string]string{
					"true":  "It names a concrete, checkable result.",
					"false": "It is vague or a matter of opinion, with no checkable result.",
				},
			},
		},
	}
	var response struct {
		Answers map[string]struct {
			Noul *float64 `json:"noul"`
		} `json:"answers"`
		Error string `json:"error"`
	}
	if err := c.post(ctx, "/v1/systemone", body, &response); err != nil {
		return 0, err
	}
	answer, ok := response.Answers["checkable"]
	if !ok || answer.Noul == nil {
		return 0, errors.New("decision model returned no checkable answer")
	}
	return *answer.Noul, nil
}

// ModelDigest identifies the exact local model build, so a recorded answer names the
// weights that produced it and not only a mutable tag.
func (c *Client) ModelDigest(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return "", err
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var tags struct {
		Models []struct {
			Name   string `json:"name"`
			Model  string `json:"model"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&tags); err != nil {
		return "", err
	}
	wanted := c.Model
	if !strings.Contains(wanted, ":") {
		wanted += ":latest"
	}
	for _, model := range tags.Models {
		if model.Name == wanted || model.Model == wanted {
			return model.Digest, nil
		}
	}
	return "", fmt.Errorf("model %s is not installed", c.Model)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("decision model returned %s: %s", response.Status, strings.TrimSpace(string(payload)))
	}
	return json.Unmarshal(payload, out)
}
