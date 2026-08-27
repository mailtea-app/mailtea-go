package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// webhookEndpoints is the base path of the webhook-endpoint routes. They live
// under /v1/webhooks/, not /v1/webhook-endpoints.
const webhookEndpoints = "/v1/webhooks/endpoints"

// WebhooksService is the `webhooks` resource — outbound event subscriptions.
// Reach it as client.Webhooks.
//
// Scoped to a publication. Create returns the signing_secret ONCE; store it and
// verify deliveries with VerifyWebhookSignature.
type WebhooksService struct {
	client *Client
}

// Create registers a webhook endpoint.
func (s *WebhooksService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, webhookEndpoints, bodyOrNil(params))
}

// List lists webhook endpoints. Requires publication_id.
func (s *WebhooksService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, webhookEndpoints+query(params), nil)
}

// Get retrieves one webhook endpoint. Requires publication_id.
func (s *WebhooksService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, webhookEndpoints+"/"+url.PathEscape(id)+query(params), nil)
}

// Update changes a webhook endpoint. publication_id travels in the query string
// and the body alike.
func (s *WebhooksService) Update(ctx context.Context, id string, params Params) (Object, error) {
	path := webhookEndpoints + "/" + url.PathEscape(id) + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPatch, path, bodyOrNil(params))
}

// Delete removes a webhook endpoint. Requires publication_id.
func (s *WebhooksService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, webhookEndpoints+"/"+url.PathEscape(id)+query(params), nil)
}
