package mailtea

import (
	"context"
	"net/http"
	"testing"
)

// Params carries the body through untouched, so a nil value has to survive as a
// JSON null. Only the query string drops nils — if the body helper ever learned
// to do the same, "remove it" would silently become "leave it alone" and the
// caller would get a 200 saying nothing happened.
func TestDomainUpdateSendsNullToClearTheTrackingSubdomain(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.Update(context.Background(), "dom_1", Params{
		"publication_id":     "pub_1",
		"tracking_subdomain": nil,
	}); err != nil {
		t.Fatalf("Domains.Update: %v", err)
	}

	request := mock.last(t)
	assertRoute(t, request, http.MethodPatch, "/v1/domains/dom_1")
	assertQuery(t, request, "publication_id", "pub_1")
	value, present := request.Body["tracking_subdomain"]
	if !present {
		t.Fatalf("body has no tracking_subdomain (raw: %s)", request.RawBody)
	}
	if value != nil {
		t.Errorf("body[tracking_subdomain] = %#v, want nil", value)
	}
}

// Three states, not two: an absent key leaves the subdomain alone, nil removes
// it. A body that always carried the key would clear it on every update.
func TestDomainUpdateOmitsTheTrackingSubdomainWhenUnnamed(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.Update(context.Background(), "dom_1", Params{
		"publication_id": "pub_1",
		"tls":            "enforced",
	}); err != nil {
		t.Fatalf("Domains.Update: %v", err)
	}

	if _, present := mock.last(t).Body["tracking_subdomain"]; present {
		t.Errorf("body carries tracking_subdomain when it was never named")
	}
}

func TestDomainUpdateStillSendsANamedTrackingSubdomain(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.Update(context.Background(), "dom_1", Params{
		"publication_id":     "pub_1",
		"tracking_subdomain": "links",
	}); err != nil {
		t.Fatalf("Domains.Update: %v", err)
	}

	assertBodyField(t, mock.last(t), "tracking_subdomain", "links")
}
