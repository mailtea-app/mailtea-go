package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// SendersService is the `senders` resource — named From identities. Reach it as
// client.Senders.
//
// Senders are scoped to a publication. Create takes name and email (the address
// must live on a verified, DKIM-verified email domain), plus optional reply_to
// and is_default. The email is immutable, so Update only changes name,
// reply_to and is_default.
type SendersService struct {
	client *Client
}

// Create adds a sender.
func (s *SendersService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/senders", bodyOrNil(params))
}

// List lists senders, cursor-paginated. Filters: publication_id (required),
// limit, after (a cursor from a previous next_cursor).
func (s *SendersService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/senders"+query(params), nil)
}

// Get retrieves one sender. Requires publication_id.
func (s *SendersService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/senders/"+url.PathEscape(id)+query(params), nil)
}

// Update changes a sender's name, reply_to or is_default. publication_id is
// required in the body — unlike most updates, this one reads nothing from the
// query string.
func (s *SendersService) Update(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPatch, "/v1/senders/"+url.PathEscape(id), bodyOrNil(params))
}

// Delete removes a sender. Requires publication_id.
func (s *SendersService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/senders/"+url.PathEscape(id)+query(params), nil)
}
