package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// PostsService is the `posts` resource — newsletter posts and issues. Reach it
// as client.Posts.
type PostsService struct {
	client *Client
}

// CreatePostRequest is the body of POST /v1/posts.
//
// Seed the post from a published server template with TemplateID + Variables,
// or pass inline HTML. Kind selects the post type ("newsletter" or
// "broadcast"). Set Send to deliver right after creating (or add ScheduledAt to
// schedule) — that requires the `issues:send` scope.
type CreatePostRequest struct {
	PublicationID string `json:"publication_id"`
	Subject       string `json:"subject,omitempty"`
	Name          string `json:"name,omitempty"`
	Kind          string `json:"kind,omitempty"`

	HTML string `json:"html,omitempty"`
	Text string `json:"text,omitempty"`

	TemplateID string                 `json:"template_id,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`

	From    string `json:"from,omitempty"`
	ReplyTo string `json:"reply_to,omitempty"`

	// Send delivers the post to the audience as part of creating it.
	Send bool `json:"send,omitempty"`
	// ScheduledAt, with Send, queues that delivery for later (ISO 8601).
	ScheduledAt string `json:"scheduled_at,omitempty"`

	// Extra carries wire fields this SDK version does not name yet.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r CreatePostRequest) MarshalJSON() ([]byte, error) {
	type alias CreatePostRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// SendPostRequest is the body of POST /v1/posts/{id}/send. Leave it zero to
// send now; set ScheduledAt (ISO 8601) to schedule.
type SendPostRequest struct {
	ScheduledAt string `json:"scheduled_at,omitempty"`

	// Extra carries wire fields this SDK version does not name yet.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r SendPostRequest) MarshalJSON() ([]byte, error) {
	type alias SendPostRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// SendTestPostRequest is the body of POST /v1/posts/{id}/test. Up to 10
// recipients; From must use a verified domain.
type SendTestPostRequest struct {
	Recipients []string `json:"recipients"`
	From       string   `json:"from,omitempty"`
	ReplyTo    string   `json:"reply_to,omitempty"`

	// Extra carries wire fields this SDK version does not name yet.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r SendTestPostRequest) MarshalJSON() ([]byte, error) {
	type alias SendTestPostRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// SendTestPostResponse reports which test recipients were reached.
type SendTestPostResponse struct {
	SentTo   []string `json:"sent_to"`
	FailedTo []string `json:"failed_to"`
}

// Create adds a newsletter post — a draft unless Send is set.
func (s *PostsService) Create(ctx context.Context, request CreatePostRequest) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/posts", request)
}

// List lists posts, most recent first, offset-paginated. Takes publication_id
// (required) plus optional limit, offset, status and kind.
func (s *PostsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/posts"+query(params), nil)
}

// Get retrieves a post by id.
func (s *PostsService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/posts/"+url.PathEscape(id)+query(params), nil)
}

// Update changes a draft post — subject, html, text, from, reply_to, name. Sent
// posts are immutable.
func (s *PostsService) Update(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPatch, "/v1/posts/"+url.PathEscape(id), bodyOrNil(params))
}

// Delete removes a draft post. Sent posts cannot be deleted.
func (s *PostsService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/posts/"+url.PathEscape(id)+query(params), nil)
}

// Send delivers a draft post to the publication's audience — immediately, or at
// ScheduledAt. Requires the `issues:send` scope.
func (s *PostsService) Send(ctx context.Context, id string, request SendPostRequest) (Object, error) {
	// A zero request sends no body at all: the endpoint reads "send now" from
	// the absence of scheduled_at, not from an empty object.
	var body interface{}
	if request.ScheduledAt != "" || len(request.Extra) > 0 {
		body = request
	}
	return s.client.object(ctx, http.MethodPost, "/v1/posts/"+url.PathEscape(id)+"/send", body)
}

// SendTest sends a TEST copy of a post to specific recipients, to check it
// before subscribers see it. It renders the post exactly as a subscriber would
// receive it and delivers a one-shot [TEST] email — it does NOT send to the
// audience.
func (s *PostsService) SendTest(ctx context.Context, id string, request SendTestPostRequest) (*SendTestPostResponse, error) {
	var out SendTestPostResponse
	path := "/v1/posts/" + url.PathEscape(id) + "/test"
	if err := s.client.call(ctx, http.MethodPost, path, request, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
