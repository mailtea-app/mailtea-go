package mailtea

import (
	"context"
	"net/http"
	"testing"
)

func TestDomainClaimCreatePostsTheClaim(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.Claims.Create(context.Background(), Params{
		"publication_id": "pub_1",
		"name":           "acme.com",
		"region":         "eu-west-1",
	}); err != nil {
		t.Fatalf("Claims.Create: %v", err)
	}

	request := mock.last(t)
	assertRoute(t, request, http.MethodPost, "/v1/domains/claim")
	assertBodyField(t, request, "publication_id", "pub_1")
	assertBodyField(t, request, "name", "acme.com")
	assertBodyField(t, request, "region", "eu-west-1")
}

func TestDomainClaimGetVerifyCancelReachTheirRoutes(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()
	params := Params{"publication_id": "pub_1"}

	if _, err := client.Domains.Claims.Get(ctx, "clm_1", params); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/domains/claims/clm_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Domains.Claims.Verify(ctx, "clm_1", params); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/domains/claims/clm_1/verify")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Domains.Claims.Cancel(ctx, "clm_1", params); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/domains/claims/clm_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
}

// The claim id lands in a path segment, so it has to be escaped the way every
// other id-taking method in this SDK escapes one.
func TestDomainClaimIDIsPathEscaped(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.Claims.Get(context.Background(), "clm/1", Params{
		"publication_id": "pub_1",
	}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/domains/claims/clm/1")
}

// The resource forwards whatever Params it is handed, so the multi-region
// fields need no code. This is the test that says so, and that fails if anyone
// ever adds a whitelist.
func TestDomainCreateForwardsRegionTLSAndTrackingSubdomain(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.Create(context.Background(), Params{
		"publication_id":     "pub_1",
		"name":               "acme.com",
		"region":             "ap-southeast-1",
		"tls":                "enforced",
		"tracking_subdomain": "links",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	request := mock.last(t)
	assertBodyField(t, request, "region", "ap-southeast-1")
	assertBodyField(t, request, "tls", "enforced")
	assertBodyField(t, request, "tracking_subdomain", "links")
}

func TestDomainListForwardsTheRegionAndStatusFilters(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.List(context.Background(), Params{
		"publication_id": "pub_1",
		"region":         "eu-west-1",
		"status":         "verified",
	}); err != nil {
		t.Fatalf("List: %v", err)
	}
	request := mock.last(t)
	assertQuery(t, request, "region", "eu-west-1")
	assertQuery(t, request, "status", "verified")
}

func TestDomainUpdateForwardsTLSAndTrackingSubdomain(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Domains.Update(context.Background(), "dom_1", Params{
		"publication_id":     "pub_1",
		"tls":                "enforced",
		"tracking_subdomain": "links",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	request := mock.last(t)
	assertRoute(t, request, http.MethodPatch, "/v1/domains/dom_1")
	assertBodyField(t, request, "tls", "enforced")
	assertBodyField(t, request, "tracking_subdomain", "links")
}
