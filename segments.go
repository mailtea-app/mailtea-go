package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// SegmentsService is the `segments` resource. Reach it as client.Segments.
//
// Audience segments are scoped to a publication — pass publication_id. To clear
// a nullable filter on update, set it to nil explicitly (Params{"status_filter":
// nil} is dropped from a query but kept in a body); omit the key to leave it
// unchanged.
type SegmentsService struct {
	client *Client
}

// Create adds a segment.
func (s *SegmentsService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/segments", bodyOrNil(params))
}

// List lists segments. Requires publication_id.
func (s *SegmentsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/segments"+query(params), nil)
}

// Get retrieves one segment. Requires publication_id.
func (s *SegmentsService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/segments/"+url.PathEscape(id)+query(params), nil)
}

// Update changes a segment. publication_id travels in the query string and the
// body alike.
func (s *SegmentsService) Update(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/segments/" + url.PathEscape(id) + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPatch, path, bodyOrNil(params))
}

// Delete removes a segment. Requires publication_id.
func (s *SegmentsService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/segments/"+url.PathEscape(id)+query(params), nil)
}
