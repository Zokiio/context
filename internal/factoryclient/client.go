// Package factoryclient calls the authenticated factory service used by the CLI
// and enrolled workers. Domain request and response types belong to factory.
package factoryclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const responseLimit = 16 << 20

// Client keeps credentials in request headers. HTTPClient may be replaced by
// callers that need a custom TLS transport. Redirects are always refused.
type Client struct {
	HTTPClient *http.Client
	base       *url.URL
	token      string
}

// APIError is a service rejection. It deliberately excludes request bodies and
// authorization headers from its diagnostic representation.
type APIError struct {
	Status  int            `json:"-"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *APIError) Error() string { return fmt.Sprintf("factory: %s: %s", e.Code, e.Message) }

// New accepts HTTPS endpoints and loopback HTTP endpoints. HTTP on a shared
// network would expose the bearer credential and is rejected.
func New(baseURL, token string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("factory endpoint must be an HTTP(S) URL without credentials, query, or fragment")
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return nil, errors.New("factory endpoints outside loopback require HTTPS")
		}
	}
	if strings.TrimSpace(token) == "" || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("factory bearer token is required")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	return &Client{base: u, token: token, HTTPClient: &http.Client{Timeout: 30 * time.Second}}, nil
}

// DoJSON sends a request to a /v1 path on the configured installation. Input and
// output are domain structs, not CLI-specific transport copies.
func (c *Client) DoJSON(ctx context.Context, method, path string, input, output any) error {
	if !strings.HasPrefix(path, "/v1/") || strings.Contains(path, "#") {
		return errors.New("factory request path must start with /v1/")
	}
	relative, err := url.Parse(path)
	if err != nil || relative.Host != "" || relative.Scheme != "" {
		return errors.New("invalid factory request path")
	}
	for key := range relative.Query() {
		if strings.Contains(strings.ToLower(key), "token") {
			return errors.New("factory tokens cannot be sent in URLs")
		}
	}
	u := *c.base
	u.Path += relative.Path
	u.RawPath = c.base.EscapedPath() + relative.EscapedPath()
	u.RawQuery = relative.RawQuery
	var body bytes.Buffer
	if input != nil {
		if err := json.NewEncoder(&body).Encode(input); err != nil {
			return fmt.Errorf("encode factory request: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), &body)
	if err != nil {
		return errors.New("cannot construct factory request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	transport := c.HTTPClient
	if transport == nil {
		transport = &http.Client{Timeout: 30 * time.Second}
	}
	client := *transport
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		// net/http errors can contain the URL. It never contains the credential.
		return fmt.Errorf("call factory service: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return fmt.Errorf("read factory response: %w", err)
	}
	if len(raw) > responseLimit {
		return errors.New("factory response exceeds 16 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rejection := &APIError{Status: resp.StatusCode}
		if json.Unmarshal(raw, rejection) != nil || rejection.Code == "" {
			rejection.Code = "http_error"
			rejection.Message = http.StatusText(resp.StatusCode)
		}
		rejection.Message = strings.ReplaceAll(rejection.Message, c.token, "[redacted]")
		return rejection
	}
	if output == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return fmt.Errorf("decode factory response: %w", err)
	}
	return nil
}
