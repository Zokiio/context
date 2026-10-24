package factoryclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Zokiio/context/internal/factory"
)

func (c *Client) ReportDispute(ctx context.Context, q factory.ReportDisputeRequest) (factory.Dispute, error) {
	return request[factory.Dispute](ctx, c, http.MethodPost, "/v1/jobs/"+url.PathEscape(string(q.JobID))+"/disputes", q)
}

func (c *Client) GetDispute(ctx context.Context, id factory.ID) (factory.Dispute, error) {
	return request[factory.Dispute](ctx, c, http.MethodGet, "/v1/disputes/"+url.PathEscape(string(id)), nil)
}

func (c *Client) ListDisputes(ctx context.Context, projectID factory.ProjectID, workItemID string) ([]factory.Dispute, error) {
	return request[[]factory.Dispute](ctx, c, http.MethodGet, "/v1/projects/"+url.PathEscape(string(projectID))+"/work-items/"+url.PathEscape(workItemID)+"/disputes", nil)
}

func (c *Client) ResolveDispute(ctx context.Context, q factory.ResolveDisputeRequest) (factory.Dispute, error) {
	return request[factory.Dispute](ctx, c, http.MethodPost, "/v1/disputes/"+url.PathEscape(string(q.DisputeID))+"/resolve", q)
}
