package mailtea

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AutomationsService is the `automations` resource — multi-step contact
// journeys. Reach it as client.Automations.
//
// Automations are scoped to a publication. An automation is a graph: `steps`
// (each {"key", "type", "label", "config"}) plus optional `connections` (each
// {"from", "to", "branch"}).
//
// `connections` is optional: omit it and the server links the steps in array
// order with branch "next", rooted at the trigger. A graph containing a
// `condition` or `wait_for_event` step cannot be inferred that way and is
// rejected with `connections_required_for_branching` — send its connections
// explicitly.
//
// Failures come back as coded `issues[]` rather than schema errors, and for a
// draft/paused/archived automation they ride along informationally instead of
// blocking the save.
type AutomationsService struct {
	client *Client
}

// Validate dry-runs a graph without creating anything. Takes publication_id and
// steps, plus optional connections. Returns
// {"object": "automation_validation", "valid": ..., "issues": [...]}.
func (s *AutomationsService) Validate(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/automations/validate", bodyOrNil(params))
}

// Create adds an automation. Takes publication_id, name and steps, plus
// optional description, connections, reentry_policy
// (once/once_per_window/always — once_per_window requires
// reentry_window_seconds), on_step_failure and validate_only. With
// validate_only nothing is written and an automation_validation comes back
// instead. New automations start as draft — Activate starts them.
func (s *AutomationsService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/automations", bodyOrNil(params))
}

// List lists automations, cursor-paginated. Filters: publication_id (required),
// status (draft/active/paused/archived), limit, after. List items omit steps,
// connections, valid and issues — use Get for the full graph.
func (s *AutomationsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/automations"+query(params), nil)
}

// Get retrieves one automation with its live graph and current issues[].
// Requires publication_id.
func (s *AutomationsService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/automations/"+url.PathEscape(id)+query(params), nil)
}

// Update changes an automation's name, description, steps, connections,
// reentry_policy, reentry_window_seconds or on_step_failure. publication_id is
// required and is sent as a query parameter ONLY — this endpoint rejects it in
// the body. The graph is replaced wholesale and cuts a new version.
//
// validate_only returns an automation_validation and writes nothing. A graph
// change carrying errors saves anyway while the automation is
// draft/paused/archived; on an `active` one it is a 422 — pause, save, then
// start again.
func (s *AutomationsService) Update(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(id) + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPatch, path, withoutPublicationID(params))
}

// Delete removes an automation. Requires publication_id. Deleting an `active`
// automation is a 409 `automation_active` — pause or archive it first so its
// in-flight runs are not dropped silently.
func (s *AutomationsService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/automations/"+url.PathEscape(id)+query(params), nil)
}

// Activate starts the automation so new contacts enroll. Requires
// publication_id. A graph with errors is refused with 422 `automation_invalid`
// and the blocking issues[].
func (s *AutomationsService) Activate(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(id) + "/activate" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Pause stops new enrollments. Requires publication_id (query). Optional
// cancel_runs — it DEFAULTS TO FALSE here, so in-flight runs keep going; pass
// cancel_runs true to exit them. Returns the automation plus canceled_runs.
func (s *AutomationsService) Pause(ctx context.Context, id string, params Params) (Object, error) {
	return s.lifecycle(ctx, id, "/pause", params)
}

// Archive archives the automation. Requires publication_id (query). Optional
// cancel_runs — it DEFAULTS TO TRUE here, the opposite of Pause, so in-flight
// runs exit with `automation_archived`. Returns the automation plus
// canceled_runs.
func (s *AutomationsService) Archive(ctx context.Context, id string, params Params) (Object, error) {
	return s.lifecycle(ctx, id, "/archive", params)
}

// lifecycle splits the payload the way pause and archive read it:
// publication_id from the query, cancel_runs from the body. No body is sent
// when the caller omitted cancel_runs, so the per-verb default applies —
// sending an empty object instead would be rejected by the endpoint's schema.
func (s *AutomationsService) lifecycle(ctx context.Context, id, suffix string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(id) + suffix + query(withPublicationID(params))
	var body interface{}
	if cancelRuns, ok := params["cancel_runs"]; ok && cancelRuns != nil {
		body = Params{"cancel_runs": cancelRuns}
	}
	return s.client.object(ctx, http.MethodPost, path, body)
}

// Versions lists an automation's versions, cursor-paginated. Filters:
// publication_id (required), limit, after. List items carry no
// steps/connections — use Version for a stored graph.
func (s *AutomationsService) Versions(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(id) + "/versions" + query(params)
	return s.client.object(ctx, http.MethodGet, path, nil)
}

// Version retrieves one stored version, including its steps and connections.
// Requires publication_id. This is the graph a run of that version is pinned
// to — editing the automation never rewrites it.
func (s *AutomationsService) Version(ctx context.Context, id string, version interface{}, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(id) +
		"/versions/" + url.PathEscape(fmt.Sprintf("%v", version)) + query(params)
	return s.client.object(ctx, http.MethodGet, path, nil)
}

// Metrics returns per-step funnel counts. Filters: publication_id (required),
// version (omit to aggregate across ALL versions), since, until (ISO 8601).
// Test runs are always excluded (excludes_test_runs: true). Condition steps
// report branches {condition_met, condition_not_met}; wait_for_event steps
// {event_received, timeout}.
func (s *AutomationsService) Metrics(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(id) + "/metrics" + query(params)
	return s.client.object(ctx, http.MethodGet, path, nil)
}

// Test runs the automation once against a real contact. publication_id is
// required and is sent as a query parameter; the body takes one of contact_id
// or email, plus optional event_properties to seed the run's `event.*`
// namespace.
//
// A test run SENDS REAL, BILLED EMAIL to that inbox — it does not bypass any
// send gate. It is flagged is_test and excluded from Metrics. Returns 202 with
// the queued run.
func (s *AutomationsService) Test(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(id) + "/test" + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPost, path, withoutPublicationID(params))
}
