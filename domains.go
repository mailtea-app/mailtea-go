package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// DomainsService is the `domains` resource — email and site sending domains.
// Reach it as client.Domains.
//
// Scoped to a publication. Register a domain, add the DNS `records` the
// response lists, then Verify it before sending from it.
type DomainsService struct {
	client *Client

	// Tracking covers the CNAME tracking sub-domains under a domain.
	Tracking *TrackingDomainsService

	// Claims covers taking a domain back from the team that currently holds it.
	Claims *DomainClaimsService
}

// Create registers a domain. The response's `records` lists the DNS records to
// add.
func (s *DomainsService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/domains", bodyOrNil(params))
}

// List lists domains. Requires publication_id.
func (s *DomainsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/domains"+query(params), nil)
}

// Get retrieves one domain. Requires publication_id.
func (s *DomainsService) Get(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/domains/"+url.PathEscape(id)+query(params), nil)
}

// Verify checks a domain's DNS records; on success its status becomes
// "verified".
func (s *DomainsService) Verify(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/domains/" + url.PathEscape(id) + "/verify" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Update changes a domain — open_tracking, click_tracking, custom_return_path
// and the like. publication_id travels in the query string and the body alike.
//
// custom_return_path delegates a subdomain as the envelope sender so SPF aligns
// with your own domain. Mail keeps sending on the default return-path until the
// delegated DNS resolves.
//
// tracking_subdomain set to nil removes a tracking subdomain: the domain's
// links go back to being served from the Mailtea host, and links in mail
// already sent point at the old hostname and stop resolving. Params reaches
// the encoder as given, so the nil travels as a JSON null — leaving the key
// out (leave the subdomain alone) and setting it to nil (remove it) are
// different requests. An empty string is neither; it is refused with
// tracking_subdomain_invalid. nil is an update-only value: a create has
// nothing to clear.
func (s *DomainsService) Update(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/domains/" + url.PathEscape(id) + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPatch, path, bodyOrNil(params))
}

// Delete removes a domain. Requires publication_id.
func (s *DomainsService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/domains/"+url.PathEscape(id)+query(params), nil)
}

// TrackingDomainsService covers tracking sub-domains (CNAME) under a domain —
// used to serve open-pixel and click-tracking links from your own domain. Reach
// it as client.Domains.Tracking.
type TrackingDomainsService struct {
	client *Client
}

// Create adds a tracking sub-domain. Takes publication_id and subdomain. The
// response's `records` lists the CNAME to add.
//
// publication_id goes in the query string and only `subdomain` in the body,
// which is what this endpoint reads.
func (s *TrackingDomainsService) Create(ctx context.Context, domainID string, params Params) (Object, error) {
	path := "/v1/domains/" + url.PathEscape(domainID) + "/tracking-domains" + query(withPublicationID(params))
	return s.client.object(ctx, http.MethodPost, path, Params{"subdomain": params["subdomain"]})
}

// List lists a domain's tracking sub-domains. Requires publication_id.
func (s *TrackingDomainsService) List(ctx context.Context, domainID string, params Params) (Object, error) {
	path := "/v1/domains/" + url.PathEscape(domainID) + "/tracking-domains" + query(params)
	return s.client.object(ctx, http.MethodGet, path, nil)
}

// Verify checks a tracking sub-domain's CNAME. Requires publication_id.
func (s *TrackingDomainsService) Verify(ctx context.Context, domainID, trackingDomainID string, params Params) (Object, error) {
	path := "/v1/domains/" + url.PathEscape(domainID) +
		"/tracking-domains/" + url.PathEscape(trackingDomainID) + "/verify" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Delete removes a tracking sub-domain. Requires publication_id.
func (s *TrackingDomainsService) Delete(ctx context.Context, domainID, trackingDomainID string, params Params) (Object, error) {
	path := "/v1/domains/" + url.PathEscape(domainID) +
		"/tracking-domains/" + url.PathEscape(trackingDomainID) + query(params)
	return s.client.object(ctx, http.MethodDelete, path, nil)
}

// DomainClaimsService covers domain claims — taking a domain back from
// whichever publication currently holds it. Reach it as
// client.Domains.Claims.
//
// Use it when Create is refused because the host is connected to another
// publication: open a claim, publish the TXT record the response lists to prove
// you control the DNS, then Verify. On success the other team's domain is
// released and a fresh one is created for you.
type DomainClaimsService struct {
	client *Client
}

// Create opens a claim. Takes publication_id, name, and an optional region.
// The response's `records` lists the TXT record to publish.
func (s *DomainClaimsService) Create(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/domains/claim", bodyOrNil(params))
}

// Get polls a claim. Requires publication_id.
func (s *DomainClaimsService) Get(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/domains/claims/" + url.PathEscape(id) + query(params)
	return s.client.object(ctx, http.MethodGet, path, nil)
}

// Verify checks the TXT record and completes the claim if it is there.
//
// Safe to call repeatedly: a record that has not propagated yet leaves the
// claim pending with the same record, so nothing has to be republished. A
// completed claim answers with the fresh `domain` beside the claim.
func (s *DomainClaimsService) Verify(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/domains/claims/" + url.PathEscape(id) + "/verify" + query(params)
	return s.client.object(ctx, http.MethodPost, path, nil)
}

// Cancel withdraws a pending claim. Requires publication_id.
func (s *DomainClaimsService) Cancel(ctx context.Context, id string, params Params) (Object, error) {
	path := "/v1/domains/claims/" + url.PathEscape(id) + query(params)
	return s.client.object(ctx, http.MethodDelete, path, nil)
}
