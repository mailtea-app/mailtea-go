package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// ContactPropertiesService is the `contact_properties` resource — custom
// contact fields. Reach it as client.ContactProperties.
//
// Definitions are team-scoped: there is no publication_id here. Create takes
// key and type ("string" or "number").
type ContactPropertiesService struct {
	client *Client
}

// Create adds a contact property definition.
func (s *ContactPropertiesService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/contact-properties", bodyOrNil(params))
}

// List lists contact property definitions.
func (s *ContactPropertiesService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/contact-properties"+query(params), nil)
}

// Update changes a contact property definition.
func (s *ContactPropertiesService) Update(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPatch, "/v1/contact-properties/"+url.PathEscape(id), bodyOrNil(params))
}

// Delete removes a contact property definition.
func (s *ContactPropertiesService) Delete(ctx context.Context, id string) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/contact-properties/"+url.PathEscape(id), nil)
}
