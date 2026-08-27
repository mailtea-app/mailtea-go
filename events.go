package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// EventsService is the `events` resource — custom product events that trigger
// automations and resume `wait_for_event` steps. Reach it as client.Events.
//
// Events are scoped to a publication.
type EventsService struct {
	client *Client
}

// Send records an event for a contact. Takes publication_id, name, and exactly
// one of contact_id or email (both is a 400 `contact_reference_conflict`,
// neither a 400 `contact_reference_required`). Optional: create_contact,
// properties, occurred_at, idempotency_key.
//
// create_contact is OPT-IN — without it an unresolvable address is a 404
// `contact_not_found` rather than a new contact.
//
// Returns 202 with enrolled_automations and resumed_runs. A replay of the same
// idempotency_key returns the ORIGINAL event id with replayed: true and always
// reports enrolled_automations: 0, resumed_runs: 0. Note that resumed_runs: 0
// on a FRESH ingest does not prove nothing matched — a run being advanced
// concurrently is invisible for that instant, so read the run itself
// (client.AutomationRuns.Get) rather than the counter.
func (s *EventsService) Send(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/events", bodyOrNil(params))
}

// List lists recorded events, cursor-paginated. Filters: publication_id
// (required), name, contact_id, limit, after.
func (s *EventsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/events"+query(params), nil)
}

// EventDefinitionsService is the `event_definitions` resource — the catalog of
// event names a publication expects, with optional property schemas. Reach it
// as client.EventDefinitions.
//
// Definitions are scoped to a publication. They are documentation and tooling,
// not a gate: Events.Send accepts an event with no definition.
type EventDefinitionsService struct {
	client *Client
}

// Create adds an event definition. Takes publication_id and name, plus optional
// description and schema_json. The name is immutable once created.
func (s *EventDefinitionsService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/event-definitions", bodyOrNil(params))
}

// List lists event definitions, cursor-paginated. Filters: publication_id
// (required), limit, after. List items carry no inferred_properties — use Get
// for those.
func (s *EventDefinitionsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/event-definitions"+query(params), nil)
}

// Get retrieves one definition. Requires publication_id. Adds schema_properties
// and inferred_properties — the latter computed on read over the last 500
// events, reporting each key's type, sample count and COVERAGE. Low coverage is
// the trap: a condition on a key present in 3% of events will almost never
// match.
func (s *EventDefinitionsService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/event-definitions/"+url.PathEscape(id)+query(params), nil)
}

// Update changes a definition's description or schema_json (an explicit nil
// clears the schema back to free-form). publication_id is required and is sent
// as a query parameter ONLY. `name` is immutable — sending it is a 400
// `event_name_immutable`, not a silently dropped rename.
func (s *EventDefinitionsService) Update(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/event-definitions/" + url.PathEscape(id) + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPatch, path, withoutPublicationID(params))
}

// Delete removes an event definition. Requires publication_id. Events already
// recorded under that name are untouched.
func (s *EventDefinitionsService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/event-definitions/"+url.PathEscape(id)+query(params), nil)
}
