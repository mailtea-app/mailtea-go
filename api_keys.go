package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// APIKeysService is the `api_keys` resource. Reach it as client.APIKeys.
//
// Requires a token with `settings:write`. A key can never be granted scopes the
// calling token does not already hold.
type APIKeysService struct {
	client *Client
}

// Create mints an API key. The `token` is returned ONCE — store it securely.
//
// Takes name, optional permission ("full_access" or "sending_access"), and
// optional domain_id.
func (s *APIKeysService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/api-keys", bodyOrNil(params))
}

// List lists API keys. Token values are never returned.
func (s *APIKeysService) List(ctx context.Context) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/api-keys", nil)
}

// Revoke deletes an API key by id.
func (s *APIKeysService) Revoke(ctx context.Context, id string) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/api-keys/"+url.PathEscape(id), nil)
}
