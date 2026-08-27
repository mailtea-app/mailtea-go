package mailtea

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// One test per resource: the method, the path, the bearer token, the body or
// query the API actually reads, and the field the caller came for.

func TestContacts(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	created, err := client.Contacts.Create(ctx, CreateContactRequest{
		PublicationID: "pub_1",
		Email:         "reader@example.com",
		Status:        "active",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/contacts")
	assertBodyField(t, mock.last(t), "publication_id", "pub_1")
	assertBodyField(t, mock.last(t), "email", "reader@example.com")
	if created.String("id") != "obj_1" {
		t.Errorf("id = %q", created.String("id"))
	}

	// Upsert is Create under the name of what POST /v1/contacts really does.
	if _, err := client.Contacts.Upsert(ctx, CreateContactRequest{PublicationID: "pub_1", Email: "r@e.com"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/contacts")

	if _, err := client.Contacts.List(ctx, Params{"publication_id": "pub_1", "status": "active"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/contacts")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Contacts.Get(ctx, "reader@example.com", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/contacts/reader@example.com")

	if _, err := client.Contacts.Update(ctx, "con_1", UpdateContactRequest{
		PublicationID: "pub_1",
		Status:        "unsubscribed",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// This endpoint reads the publication from the query AND keeps it in the body.
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/contacts/con_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	assertBodyField(t, mock.last(t), "publication_id", "pub_1")
	assertBodyField(t, mock.last(t), "status", "unsubscribed")

	if _, err := client.Contacts.Delete(ctx, "con_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/contacts/con_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
}

func TestSegments(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Segments.Create(ctx, Params{"publication_id": "pub_1", "name": "Engaged"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/segments")
	assertBodyField(t, mock.last(t), "name", "Engaged")

	if _, err := client.Segments.List(ctx, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/segments")

	if _, err := client.Segments.Get(ctx, "seg_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/segments/seg_1")

	if _, err := client.Segments.Update(ctx, "seg_1", Params{"publication_id": "pub_1", "name": "Renamed"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/segments/seg_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	assertBodyField(t, mock.last(t), "name", "Renamed")

	if _, err := client.Segments.Delete(ctx, "seg_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/segments/seg_1")
}

func TestTopics(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Topics.Create(ctx, CreateTopicRequest{
		PublicationID:       "pub_1",
		Name:                "Product updates",
		DefaultSubscription: "opt_in",
		Visibility:          "public",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/topics")
	assertBodyField(t, mock.last(t), "default_subscription", "opt_in")
	assertBodyField(t, mock.last(t), "visibility", "public")

	if _, err := client.Topics.List(ctx, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/topics")

	if _, err := client.Topics.Get(ctx, "top_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/topics/top_1")

	if _, err := client.Topics.Update(ctx, "top_1", Params{"publication_id": "pub_1", "name": "Renamed"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/topics/top_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Topics.Delete(ctx, "top_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/topics/top_1")
}

func TestPosts(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	created, err := client.Posts.Create(ctx, CreatePostRequest{
		PublicationID: "pub_1",
		Subject:       "Issue 1",
		HTML:          "<p>Hello</p>",
		Kind:          "newsletter",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/posts")
	assertBodyField(t, mock.last(t), "subject", "Issue 1")
	assertBodyField(t, mock.last(t), "kind", "newsletter")
	if created.String("id") != "post_1" {
		t.Errorf("id = %q, want post_1", created.String("id"))
	}

	if _, err := client.Posts.List(ctx, Params{"publication_id": "pub_1", "limit": 10}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/posts")
	assertQuery(t, mock.last(t), "limit", "10")

	if _, err := client.Posts.Get(ctx, "post_1", nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/posts/post_1")

	if _, err := client.Posts.Update(ctx, "post_1", Params{"subject": "Issue 1 (edited)"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/posts/post_1")
	assertBodyField(t, mock.last(t), "subject", "Issue 1 (edited)")

	// Send now: the endpoint reads "now" from the ABSENCE of scheduled_at, so
	// an empty object must not be sent in its place.
	if _, err := client.Posts.Send(ctx, "post_1", SendPostRequest{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/posts/post_1/send")
	if mock.last(t).HasBody() {
		t.Errorf("an immediate send carried a body: %q", mock.last(t).RawBody)
	}

	if _, err := client.Posts.Send(ctx, "post_1", SendPostRequest{ScheduledAt: "2026-09-01T09:00:00Z"}); err != nil {
		t.Fatalf("Send scheduled: %v", err)
	}
	assertBodyField(t, mock.last(t), "scheduled_at", "2026-09-01T09:00:00Z")

	result, err := client.Posts.SendTest(ctx, "post_1", SendTestPostRequest{
		Recipients: []string{"you@example.com"},
		From:       "Acme <hello@acme.com>",
	})
	if err != nil {
		t.Fatalf("SendTest: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/posts/post_1/test")
	if len(result.SentTo) != 1 || result.SentTo[0] != "you@example.com" {
		t.Errorf("sent_to = %v", result.SentTo)
	}

	if _, err := client.Posts.Delete(ctx, "post_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/posts/post_1")
}

func TestSenders(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Senders.Create(ctx, Params{"publication_id": "pub_1", "name": "Acme", "email": "hello@acme.com"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/senders")
	assertBodyField(t, mock.last(t), "email", "hello@acme.com")

	if _, err := client.Senders.List(ctx, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/senders")

	if _, err := client.Senders.Get(ctx, "snd_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/senders/snd_1")

	if _, err := client.Senders.Update(ctx, "snd_1", Params{"publication_id": "pub_1", "name": "Acme Support"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Unlike its siblings, this update reads publication_id from the BODY only.
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/senders/snd_1")
	if got := mock.last(t).Query.Get("publication_id"); got != "" {
		t.Errorf("publication_id was sent in the query (%q); this endpoint reads the body", got)
	}
	assertBodyField(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Senders.Delete(ctx, "snd_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/senders/snd_1")
}

func TestAssets(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	params := Params{
		"publication_id": "pub_1",
		"content":        []byte("hi"),
		"content_type":   "image/png",
		"filename":       "hero.png",
	}
	if _, err := client.Assets.Upload(ctx, params); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/assets")
	// Raw bytes are base64-encoded for the caller.
	assertBodyField(t, mock.last(t), "content", "aGk=")
	// …without mutating the caller's map, which they may hold and re-use.
	if _, stillBytes := params["content"].([]byte); !stillBytes {
		t.Error("Upload rewrote the caller's params in place")
	}

	// An already-encoded string passes straight through.
	if _, err := client.Assets.Upload(ctx, Params{"publication_id": "pub_1", "content": "YWxyZWFkeQ=="}); err != nil {
		t.Fatalf("Upload string: %v", err)
	}
	assertBodyField(t, mock.last(t), "content", "YWxyZWFkeQ==")

	if _, err := client.Assets.List(ctx, Params{"publication_id": "pub_1", "limit": 100}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/assets")

	if _, err := client.Assets.Delete(ctx, "ast_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/assets/ast_1")
}

func TestSuppressions(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Suppressions.List(ctx, Params{"reason": "bounce", "limit": 10}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/suppressions")
	assertQuery(t, mock.last(t), "reason", "bounce")

	if _, err := client.Suppressions.Add(ctx, Params{"emails": []string{"a@b.com"}, "reason": "manual"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/suppressions")
	assertBodyField(t, mock.last(t), "reason", "manual")

	// Remove carries its payload on a DELETE, which is what this endpoint reads.
	if _, err := client.Suppressions.Remove(ctx, Params{"emails": []string{"a@b.com"}}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/suppressions")
	if !mock.last(t).HasBody() {
		t.Error("Remove sent no body; the endpoint reads `emails` from it")
	}

	csv, err := client.Suppressions.Export(ctx)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/suppressions/export")
	// The export is text/csv, handed back verbatim rather than parsed as JSON.
	if !strings.HasPrefix(csv, "email,reason,source,created_at") {
		t.Errorf("export = %q, want the raw CSV including its header row", csv)
	}
}

func TestTemplates(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Templates.Render(ctx, Params{"spec": Params{"blocks": []interface{}{}}}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/templates/render")

	if _, err := client.Templates.Create(ctx, Params{"publication_id": "pub_1", "name": "Receipt", "html": "<p>hi</p>"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/templates")

	if _, err := client.Templates.List(ctx, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/templates")

	if _, err := client.Templates.Get(ctx, "tpl_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/templates/tpl_1")

	if _, err := client.Templates.Update(ctx, "tpl_1", Params{"publication_id": "pub_1", "name": "Receipt v2"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/templates/tpl_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Templates.Publish(ctx, "tpl_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/templates/tpl_1/publish")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Templates.Unpublish(ctx, "tpl_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Unpublish: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/templates/tpl_1/unpublish")

	if _, err := client.Templates.Versions(ctx, "tpl_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Versions: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/templates/tpl_1/versions")

	// A version number is an int on the wire; the path takes it either way.
	if _, err := client.Templates.RestoreVersion(ctx, "tpl_1", 3, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("RestoreVersion: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/templates/tpl_1/versions/3/restore")

	if _, err := client.Templates.Duplicate(ctx, "tpl_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/templates/tpl_1/duplicate")

	if _, err := client.Templates.Delete(ctx, "tpl_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/templates/tpl_1")
}

func TestDomains(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Domains.Create(ctx, Params{"publication_id": "pub_1", "name": "acme.com", "purpose": "email"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/domains")

	if _, err := client.Domains.List(ctx, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/domains")

	if _, err := client.Domains.Get(ctx, "dom_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/domains/dom_1")

	if _, err := client.Domains.Verify(ctx, "dom_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/domains/dom_1/verify")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Domains.Update(ctx, "dom_1", Params{"publication_id": "pub_1", "open_tracking": false}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/domains/dom_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	assertBodyField(t, mock.last(t), "open_tracking", false)

	if _, err := client.Domains.Delete(ctx, "dom_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/domains/dom_1")

	// Tracking sub-domains: publication_id in the query, only `subdomain` in
	// the body.
	if _, err := client.Domains.Tracking.Create(ctx, "dom_1", Params{"publication_id": "pub_1", "subdomain": "links"}); err != nil {
		t.Fatalf("Tracking.Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/domains/dom_1/tracking-domains")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	assertBodyField(t, mock.last(t), "subdomain", "links")
	if _, leaked := mock.last(t).Body["publication_id"]; leaked {
		t.Error("publication_id leaked into the tracking-domain body")
	}

	if _, err := client.Domains.Tracking.List(ctx, "dom_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Tracking.List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/domains/dom_1/tracking-domains")

	if _, err := client.Domains.Tracking.Verify(ctx, "dom_1", "trk_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Tracking.Verify: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/domains/dom_1/tracking-domains/trk_1/verify")

	if _, err := client.Domains.Tracking.Delete(ctx, "dom_1", "trk_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Tracking.Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/domains/dom_1/tracking-domains/trk_1")
}

func TestWebhooks(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Webhooks.Create(ctx, Params{
		"publication_id": "pub_1",
		"url":            "https://example.com/hooks",
		"events":         []string{"email.delivered"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// The endpoints live under /v1/webhooks/, not /v1/webhook-endpoints.
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/webhooks/endpoints")

	if _, err := client.Webhooks.List(ctx, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/webhooks/endpoints")

	if _, err := client.Webhooks.Get(ctx, "whe_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/webhooks/endpoints/whe_1")

	if _, err := client.Webhooks.Update(ctx, "whe_1", Params{"publication_id": "pub_1", "enabled": false}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/webhooks/endpoints/whe_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Webhooks.Delete(ctx, "whe_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/webhooks/endpoints/whe_1")
}

func TestContactProperties(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.ContactProperties.Create(ctx, Params{"key": "plan", "type": "string"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/contact-properties")
	assertBodyField(t, mock.last(t), "key", "plan")

	if _, err := client.ContactProperties.List(ctx, nil); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/contact-properties")

	if _, err := client.ContactProperties.Update(ctx, "cp_1", Params{"key": "tier"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/contact-properties/cp_1")

	if _, err := client.ContactProperties.Delete(ctx, "cp_1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/contact-properties/cp_1")
}

func TestAPIKeys(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.APIKeys.Create(ctx, Params{"name": "CI", "permission": "sending_access"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/api-keys")
	assertBodyField(t, mock.last(t), "permission", "sending_access")

	if _, err := client.APIKeys.List(ctx); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/api-keys")

	if _, err := client.APIKeys.Revoke(ctx, "tok_1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/api-keys/tok_1")
}

func TestAutomations(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	steps := []interface{}{Params{"key": "trigger", "type": "trigger"}}

	if _, err := client.Automations.Validate(ctx, Params{"publication_id": "pub_1", "steps": steps}); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/automations/validate")

	if _, err := client.Automations.Create(ctx, Params{"publication_id": "pub_1", "name": "Welcome", "steps": steps}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/automations")

	if _, err := client.Automations.List(ctx, Params{"publication_id": "pub_1", "status": "active"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/automations")

	if _, err := client.Automations.Get(ctx, "aut_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/automations/aut_1")

	if _, err := client.Automations.Update(ctx, "aut_1", Params{"publication_id": "pub_1", "name": "Welcome v2"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// This endpoint takes publication_id in the query and REJECTS it in the body.
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/automations/aut_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	if _, leaked := mock.last(t).Body["publication_id"]; leaked {
		t.Error("publication_id leaked into the automation update body")
	}
	assertBodyField(t, mock.last(t), "name", "Welcome v2")

	if _, err := client.Automations.Activate(ctx, "aut_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/automations/aut_1/activate")

	// Pause with no cancel_runs sends NO body, so the per-verb default applies.
	if _, err := client.Automations.Pause(ctx, "aut_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/automations/aut_1/pause")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	if mock.last(t).HasBody() {
		t.Errorf("pause sent a body (%q); the endpoint's default only applies when the field is absent", mock.last(t).RawBody)
	}

	if _, err := client.Automations.Archive(ctx, "aut_1", Params{"publication_id": "pub_1", "cancel_runs": false}); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/automations/aut_1/archive")
	assertBodyField(t, mock.last(t), "cancel_runs", false)
	if _, leaked := mock.last(t).Body["publication_id"]; leaked {
		t.Error("publication_id leaked into the archive body")
	}

	if _, err := client.Automations.Versions(ctx, "aut_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Versions: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/automations/aut_1/versions")

	if _, err := client.Automations.Version(ctx, "aut_1", 2, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Version: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/automations/aut_1/versions/2")

	if _, err := client.Automations.Metrics(ctx, "aut_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Metrics: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/automations/aut_1/metrics")

	if _, err := client.Automations.Test(ctx, "aut_1", Params{"publication_id": "pub_1", "email": "you@example.com"}); err != nil {
		t.Fatalf("Test: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/automations/aut_1/test")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	assertBodyField(t, mock.last(t), "email", "you@example.com")
	if _, leaked := mock.last(t).Body["publication_id"]; leaked {
		t.Error("publication_id leaked into the automation test body")
	}

	if _, err := client.Automations.Delete(ctx, "aut_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/automations/aut_1")
}

func TestAutomationRuns(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.AutomationRuns.List(ctx, "aut_1", Params{
		"publication_id": "pub_1",
		"status":         []string{"running", "waiting"},
		"is_test":        true,
	}); err != nil {
		t.Fatalf("List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/automations/aut_1/runs")
	// A list filter is comma-joined, and a bool renders as the literal the
	// server matches on — it 400s on anything else.
	assertQuery(t, mock.last(t), "status", "running,waiting")
	assertQuery(t, mock.last(t), "is_test", "true")

	if _, err := client.AutomationRuns.Get(ctx, "aut_1", "run_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/automations/aut_1/runs/run_1")

	if _, err := client.AutomationRuns.Cancel(ctx, "aut_1", "run_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/automations/aut_1/runs/run_1/cancel")
}

func TestEventsAndDefinitions(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Events.Send(ctx, Params{
		"publication_id": "pub_1",
		"name":           "order.placed",
		"email":          "buyer@example.com",
	}); err != nil {
		t.Fatalf("Events.Send: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/events")
	assertBodyField(t, mock.last(t), "name", "order.placed")

	if _, err := client.Events.List(ctx, Params{"publication_id": "pub_1", "name": "order.placed"}); err != nil {
		t.Fatalf("Events.List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/events")

	if _, err := client.EventDefinitions.Create(ctx, Params{"publication_id": "pub_1", "name": "order.placed"}); err != nil {
		t.Fatalf("Definitions.Create: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/event-definitions")

	if _, err := client.EventDefinitions.List(ctx, Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Definitions.List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/event-definitions")

	if _, err := client.EventDefinitions.Get(ctx, "evd_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Definitions.Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/event-definitions/evd_1")

	if _, err := client.EventDefinitions.Update(ctx, "evd_1", Params{
		"publication_id": "pub_1",
		"description":    "Someone bought something",
	}); err != nil {
		t.Fatalf("Definitions.Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/event-definitions/evd_1")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")
	if _, leaked := mock.last(t).Body["publication_id"]; leaked {
		t.Error("publication_id leaked into the event-definition update body")
	}

	if _, err := client.EventDefinitions.Delete(ctx, "evd_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Definitions.Delete: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodDelete, "/v1/event-definitions/evd_1")
}

// The two updates that strip publication_id out of the body can end up with
// nothing left to send. They must still send `{}`: those endpoints read the body
// with `json().catch(() => null)` and hand the result to a schema, so no body at
// all arrives as `null` and comes back 400 "Validation failed" — a failure with
// nothing to do with what the caller asked for.
func TestUpdatesThatStripThePublicationStillSendAnObject(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Automations.Update(ctx, "aut_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("Automations.Update: %v", err)
	}
	if got := strings.TrimSpace(mock.last(t).RawBody); got != "{}" {
		t.Errorf("automations body = %q, want {}", got)
	}

	if _, err := client.EventDefinitions.Update(ctx, "evd_1", Params{"publication_id": "pub_1"}); err != nil {
		t.Fatalf("EventDefinitions.Update: %v", err)
	}
	if got := strings.TrimSpace(mock.last(t).RawBody); got != "{}" {
		t.Errorf("event-definitions body = %q, want {}", got)
	}

	// And a nil payload must not become the literal `null`.
	if _, err := client.Automations.Update(ctx, "aut_1", nil); err != nil {
		t.Fatalf("Automations.Update(nil): %v", err)
	}
	if got := strings.TrimSpace(mock.last(t).RawBody); got != "{}" {
		t.Errorf("nil params body = %q, want {}", got)
	}
}
