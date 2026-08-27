package mailtea

import (
	"context"
	"net/http"
)

// SuppressionsService is the `suppressions` resource — the team-wide
// do-not-send list. Reach it as client.Suppressions.
//
// Suppressions are team-scoped: there is no publication_id here.
type SuppressionsService struct {
	client *Client
}

// List lists suppression entries, cursor-paginated. Optional filters: reason,
// q (email search), created_after, created_before, limit, starting_after (a
// cursor from a previous next_cursor).
func (s *SuppressionsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/suppressions"+query(params), nil)
}

// Add adds addresses to the suppression list. Takes emails (up to 1000) and an
// optional reason. Returns {"added": n}.
func (s *SuppressionsService) Add(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/suppressions", bodyOrNil(params))
}

// Remove takes addresses off the suppression list. Takes emails. Returns
// {"removed": n}. The body travels on a DELETE, which is what this endpoint
// reads.
func (s *SuppressionsService) Remove(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/suppressions", bodyOrNil(params))
}

// Export returns the whole suppression list as CSV — the raw text/csv body
// (email,reason,source,created_at with a header row), not JSON.
func (s *SuppressionsService) Export(ctx context.Context) (string, error) {
	return s.client.text(ctx, http.MethodGet, "/v1/suppressions/export", nil)
}
