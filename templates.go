package mailtea

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// TemplatesService is the `templates` resource — reusable server-side email
// templates. Reach it as client.Templates.
//
// Templates are scoped to a publication (except Render, which just renders a
// spec). Create one from raw html, a json-render spec, or an editor_doc (a
// Studio editor design), then Publish it before seeding posts or emails from it.
type TemplatesService struct {
	client *Client
}

// Render renders a json-render `spec` (with optional `variables`) to HTML
// without creating a template. Returns {"html": ..., "text": ...}.
func (s *TemplatesService) Render(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/templates/render", bodyOrNil(params))
}

// Create adds a template from html, a spec, OR an editor_doc — exactly one is
// required, and the server renders html from an editor_doc, so do not send
// both. Takes publication_id and name, plus optional style_profile,
// mailtea_theme, global_css, category, preview_image_url, tags, description,
// text, subject, from, reply_to and variables.
func (s *TemplatesService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/templates", bodyOrNil(params))
}

// List lists templates, cursor-paginated. Filters: publication_id (required),
// limit, after (a cursor from a previous next_cursor).
func (s *TemplatesService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/templates"+query(params), nil)
}

// Get retrieves one template. Requires publication_id.
func (s *TemplatesService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/templates/"+url.PathEscape(id)+query(params), nil)
}

// Update changes a template. An editor_doc re-renders html server-side, so do
// not send both. global_css, category, preview_image_url, tags, text, subject,
// from and reply_to accept an explicit nil to clear them. publication_id is
// required and travels in the query string as well as the body.
func (s *TemplatesService) Update(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/templates/" + url.PathEscape(id) + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPatch, path, bodyOrNil(params))
}

// Publish makes a template available to seed posts and emails. Requires
// publication_id.
func (s *TemplatesService) Publish(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/templates/" + url.PathEscape(id) + "/publish" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Unpublish returns a published template to draft. published_at is kept — it
// records that the template was published once, not that it still is. Requires
// publication_id.
func (s *TemplatesService) Unpublish(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/templates/" + url.PathEscape(id) + "/unpublish" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Versions lists a template's design history, newest first. Requires
// publication_id; optional limit (the server caps it at the retained maximum).
//
// Entries are metadata only — version, origin ("edit", "publish" or "restore"),
// restored_from_version, format, name, sealed, is_current, created_at,
// updated_at and author — never the design document, which one entry alone can
// carry half a megabyte of. is_current marks the design the template is serving
// right now, which is not always the newest entry: a metadata-only update
// touches the template without recording a version.
//
// The reply also carries `retention`: only the newest max_versions are kept,
// and consecutive edits by the same author within coalesce_window_seconds
// collapse into one entry.
func (s *TemplatesService) Versions(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/templates/" + url.PathEscape(id) + "/versions" + query(params)
	return s.client.object(ctx, http.MethodGet, path, nil)
}

// RestoreVersion puts an older design from Versions back onto the template.
// Requires publication_id.
//
// Restoring is a content write, so THE TEMPLATE RETURNS TO DRAFT — automations
// and the API stop sending it until Publish is called again. The reply's
// `unpublished` reports whether that just happened; re-publishing is the
// caller's job.
//
// History is forward-only: the design being replaced is recorded as its own
// version first, then the restored design is appended as the new newest one.
// Nothing is rewound or deleted, so a restore is itself undone by restoring the
// entry directly above it.
//
// Restoring the design that is already current writes nothing and returns
// restored: false with reason: "identical" and unpublished: false, so a no-op
// restore cannot unpublish a live template. A version that has aged out of
// retention returns a *Error with Code "template_version_not_found".
//
// version is an int or a string — whatever Versions reported.
func (s *TemplatesService) RestoreVersion(ctx context.Context, id string, version interface{}, params Params) (Object, error) {
	path := "/v1/templates/" + url.PathEscape(id) +
		"/versions/" + url.PathEscape(fmt.Sprintf("%v", version)) +
		"/restore" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Duplicate copies a template into a new draft. Requires publication_id.
func (s *TemplatesService) Duplicate(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/templates/" + url.PathEscape(id) + "/duplicate" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Delete removes a template. Requires publication_id.
func (s *TemplatesService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/templates/"+url.PathEscape(id)+query(params), nil)
}
