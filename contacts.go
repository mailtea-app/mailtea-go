package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// ContactsService is the `contacts` resource. Reach it as client.Contacts.
//
// Audience resources are scoped to a publication — every call takes a
// publication_id.
type ContactsService struct {
	client *Client
}

// CreateContactRequest is the body of POST /v1/contacts.
//
// Status is one of "active", "unsubscribed" or "suppressed"; omit it and the
// server picks the default.
type CreateContactRequest struct {
	PublicationID string `json:"publication_id"`
	Email         string `json:"email"`
	Status        string `json:"status,omitempty"`

	// Extra carries wire fields this SDK version does not name yet.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r CreateContactRequest) MarshalJSON() ([]byte, error) {
	type alias CreateContactRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// UpdateContactRequest is the body of PATCH /v1/contacts/{id_or_email}.
// PublicationID is required — it goes in the query string and the body alike.
type UpdateContactRequest struct {
	PublicationID string `json:"publication_id"`
	Status        string `json:"status,omitempty"`

	// Extra carries wire fields this SDK version does not name yet.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r UpdateContactRequest) MarshalJSON() ([]byte, error) {
	type alias UpdateContactRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// Create adds a contact — or updates it if the email already exists in the
// publication, because the endpoint upserts. Upsert is the same call under the
// name of what it actually does.
func (s *ContactsService) Create(ctx context.Context, request CreateContactRequest) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/contacts", request)
}

// Upsert creates the contact or updates it in place — an alias of Create, named
// for what POST /v1/contacts really does.
func (s *ContactsService) Upsert(ctx context.Context, request CreateContactRequest) (Object, error) {
	return s.Create(ctx, request)
}

// List lists contacts, cursor-paginated. Filters: publication_id (required),
// status (active/unsubscribed/suppressed), search (matches the email address),
// limit, after (a cursor from a previous next_cursor).
func (s *ContactsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/contacts"+query(params), nil)
}

// Get retrieves one contact by id or by email address.
func (s *ContactsService) Get(ctx context.Context, idOrEmail string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/contacts/"+url.PathEscape(idOrEmail)+query(params), nil)
}

// Update changes a contact. The publication is sent in the query string as well
// as the body, which is what this endpoint reads.
func (s *ContactsService) Update(ctx context.Context, idOrEmail string, request UpdateContactRequest) (Object, error) {
	path := "/v1/contacts/" + url.PathEscape(idOrEmail) + publicationQuery(request.PublicationID)
	return s.client.object(ctx, http.MethodPatch, path, request)
}

// Delete removes a contact. Requires publication_id.
func (s *ContactsService) Delete(ctx context.Context, idOrEmail string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/contacts/"+url.PathEscape(idOrEmail)+query(params), nil)
}
