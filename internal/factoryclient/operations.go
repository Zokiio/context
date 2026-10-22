package factoryclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Zokiio/context/internal/factory"
)

func request[T any](ctx context.Context, c *Client, method, path string, input any) (T, error) {
	var value T
	err := c.DoJSON(ctx, method, path, input, &value)
	return value, err
}

func (c *Client) Me(ctx context.Context) (factory.Principal, error) {
	return request[factory.Principal](ctx, c, http.MethodGet, "/v1/me", nil)
}
func (c *Client) ListUsers(ctx context.Context) ([]factory.User, error) {
	return request[[]factory.User](ctx, c, http.MethodGet, "/v1/users", nil)
}
func (c *Client) CreateUser(ctx context.Context, q factory.CreateUserRequest) (factory.User, error) {
	return request[factory.User](ctx, c, http.MethodPost, "/v1/users", q)
}
func (c *Client) IssueToken(ctx context.Context, q factory.IssueTokenRequest) (factory.IssuedToken, error) {
	return request[factory.IssuedToken](ctx, c, http.MethodPost, "/v1/users/"+url.PathEscape(string(q.UserID))+"/tokens", q)
}
func (c *Client) SetUserDisabled(ctx context.Context, q factory.SetUserDisabledRequest) (factory.User, error) {
	return request[factory.User](ctx, c, http.MethodPut, "/v1/users/"+url.PathEscape(string(q.UserID))+"/disabled", q)
}
func (c *Client) RevokeToken(ctx context.Context, q factory.RevokeTokenRequest) error {
	return c.DoJSON(ctx, http.MethodPost, "/v1/tokens/"+url.PathEscape(string(q.TokenID))+"/revoke", q, nil)
}
func (c *Client) GrantProject(ctx context.Context, q factory.GrantProjectRequest) (factory.Grant, error) {
	return request[factory.Grant](ctx, c, http.MethodPut, "/v1/projects/"+url.PathEscape(string(q.ProjectID))+"/grants/"+url.PathEscape(string(q.UserID)), q)
}
func (c *Client) ListProjectGrants(ctx context.Context, id factory.ProjectID) ([]factory.Grant, error) {
	return request[[]factory.Grant](ctx, c, http.MethodGet, "/v1/projects/"+url.PathEscape(string(id))+"/grants", nil)
}
func (c *Client) Capabilities(ctx context.Context, id factory.ProjectID) ([]factory.Capability, error) {
	return request[[]factory.Capability](ctx, c, http.MethodGet, "/v1/projects/"+url.PathEscape(string(id))+"/capabilities", nil)
}
func (c *Client) RegisterExecutor(ctx context.Context, q factory.RegisterExecutorRequest) (factory.ExecutorProfile, error) {
	return request[factory.ExecutorProfile](ctx, c, http.MethodPost, "/v1/executors", q)
}
func (c *Client) ListExecutors(ctx context.Context) ([]factory.ExecutorProfile, error) {
	return request[[]factory.ExecutorProfile](ctx, c, http.MethodGet, "/v1/executors", nil)
}
func (c *Client) RegisterCheck(ctx context.Context, q factory.RegisterCheckRequest) (factory.CheckProfile, error) {
	return request[factory.CheckProfile](ctx, c, http.MethodPost, "/v1/checks", q)
}
func (c *Client) ListChecks(ctx context.Context) ([]factory.CheckProfile, error) {
	return request[[]factory.CheckProfile](ctx, c, http.MethodGet, "/v1/checks", nil)
}
func (c *Client) RegisterProject(ctx context.Context, q factory.RegisterProjectRequest) (factory.Project, error) {
	return request[factory.Project](ctx, c, http.MethodPost, "/v1/projects", q)
}
func (c *Client) ListProjects(ctx context.Context) ([]factory.Project, error) {
	return request[[]factory.Project](ctx, c, http.MethodGet, "/v1/projects", nil)
}
func (c *Client) Orientation(ctx context.Context, id factory.ProjectID) (factory.ProjectOrientation, error) {
	return request[factory.ProjectOrientation](ctx, c, http.MethodGet, "/v1/projects/"+url.PathEscape(string(id))+"/orientation", nil)
}
func (c *Client) Submit(ctx context.Context, q factory.SubmitRequest) (factory.Job, error) {
	return request[factory.Job](ctx, c, http.MethodPost, "/v1/jobs", q)
}
func (c *Client) Jobs(ctx context.Context, q factory.JobFilter) ([]factory.JobSummary, error) {
	query := url.Values{}
	if q.ProjectID != "" {
		query.Set("projectId", string(q.ProjectID))
	}
	if q.Status != "" {
		query.Set("status", string(q.Status))
	}
	if q.Limit != 0 {
		query.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.After != "" {
		query.Set("after", q.After)
	}
	return request[[]factory.JobSummary](ctx, c, http.MethodGet, "/v1/jobs?"+query.Encode(), nil)
}
func (c *Client) GetJob(ctx context.Context, id factory.JobID) (factory.Job, error) {
	return request[factory.Job](ctx, c, http.MethodGet, "/v1/jobs/"+url.PathEscape(string(id)), nil)
}
func (c *Client) Claim(ctx context.Context, q factory.WorkerClaimRequest) (factory.Assignment, error) {
	return request[factory.Assignment](ctx, c, http.MethodPost, "/v1/worker/claims", q)
}
func (c *Client) Heartbeat(ctx context.Context, q factory.HeartbeatRequest) (factory.HeartbeatResult, error) {
	return request[factory.HeartbeatResult](ctx, c, http.MethodPost, "/v1/worker/heartbeat", q)
}
func (c *Client) Report(ctx context.Context, q factory.WorkerReportRequest) (factory.Job, error) {
	return request[factory.Job](ctx, c, http.MethodPost, "/v1/worker/reports", q)
}
func (c *Client) Review(ctx context.Context, q factory.ReviewRequest) (factory.Job, error) {
	return request[factory.Job](ctx, c, http.MethodPost, "/v1/jobs/"+url.PathEscape(string(q.JobID))+"/reviews", q)
}
func (c *Client) Action(ctx context.Context, q factory.JobActionRequest) (factory.Job, error) {
	return request[factory.Job](ctx, c, http.MethodPost, "/v1/jobs/"+url.PathEscape(string(q.JobID))+"/actions", q)
}
func (c *Client) Accept(ctx context.Context, q factory.AcceptRequest) (factory.Job, error) {
	return request[factory.Job](ctx, c, http.MethodPost, "/v1/jobs/"+url.PathEscape(string(q.JobID))+"/acceptance", q)
}
func (c *Client) Events(ctx context.Context, id factory.JobID, after int64) ([]factory.Event, error) {
	return request[[]factory.Event](ctx, c, http.MethodGet, "/v1/jobs/"+url.PathEscape(string(id))+"/events?after="+strconv.FormatInt(after, 10), nil)
}
