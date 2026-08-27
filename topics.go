package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// TopicsService is the `topics` resource — topic definitions. Reach it as
// client.Topics.
//
// Topics are scoped to a publication. This manages topic definitions only;
// assigning topics to contacts is not yet exposed by the API.
type TopicsService struct {
	client *Client
}

// CreateTopicRequest is the body of POST /v1/topics.
//
// DefaultSubscription is required and is one of "opt_in" or "opt_out".
// Visibility defaults to "private"; "public" makes the topic appear on the
// reader preference page as its own subscription.
type CreateTopicRequest struct {
	PublicationID       string `json:"publication_id"`
	Name                string `json:"name"`
	DefaultSubscription string `json:"default_subscription"`
	Description         string `json:"description,omitempty"`
	Visibility          string `json:"visibility,omitempty"`

	// Extra carries wire fields this SDK version does not name yet.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r CreateTopicRequest) MarshalJSON() ([]byte, error) {
	type alias CreateTopicRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// Create adds a topic definition.
func (s *TopicsService) Create(ctx context.Context, request CreateTopicRequest) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/topics", request)
}

// List lists topics. Requires publication_id.
func (s *TopicsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/topics"+query(params), nil)
}

// Get retrieves one topic. Requires publication_id.
func (s *TopicsService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/topics/"+url.PathEscape(id)+query(params), nil)
}

// Update changes a topic. publication_id travels in the query string and the
// body alike.
func (s *TopicsService) Update(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/topics/" + url.PathEscape(id) + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPatch, path, bodyOrNil(params))
}

// Delete removes a topic. Requires publication_id.
func (s *TopicsService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/topics/"+url.PathEscape(id)+query(params), nil)
}
