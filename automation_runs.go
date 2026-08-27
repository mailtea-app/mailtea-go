package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// AutomationRunsService is the `automation_runs` resource — one contact's
// journey through one automation. Reach it as client.AutomationRuns.
//
// Runs are nested under an automation and scoped to a publication. A run PINS
// the automation version it started on, so Get returns the graph the run is
// actually executing, not the live one.
type AutomationRunsService struct {
	client *Client
}

// List lists an automation's runs, cursor-paginated. Filters: publication_id
// (required), status (one status or a []string of them, joined for you),
// contact_id, is_test, limit, after. List items omit the pinned graph and the
// step runs; use Get for those.
//
// A Go bool renders as "true"/"false" here, which is what the server matches
// `is_test` against — it 400s on anything else.
func (s *AutomationRunsService) List(ctx context.Context, automationID string, params Params) (*List, error) {
	path := "/v1/automations/" + url.PathEscape(automationID) + "/runs" + query(params)
	return s.client.list(ctx, http.MethodGet, path, nil)
}

// Get retrieves one run in full. Requires publication_id. Returns the PINNED
// steps/connections, the per-step step_runs, and `waiting` (resume_at /
// waiting_event_name) — read this rather than an event ingest's resumed_runs
// counter to tell whether an event actually advanced the run.
func (s *AutomationRunsService) Get(ctx context.Context, automationID, runID string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(automationID) +
		"/runs/" + url.PathEscape(runID) + query(params)
	return s.client.object(ctx, http.MethodGet, path, nil)
}

// Cancel stops one in-flight run. Requires publication_id. A cancelled run
// cannot be resumed. Returns the run in full detail.
func (s *AutomationRunsService) Cancel(ctx context.Context, automationID, runID string, params Params) (Object, error) {
	path := "/v1/automations/" + url.PathEscape(automationID) +
		"/runs/" + url.PathEscape(runID) + "/cancel" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}
