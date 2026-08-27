package mailtea

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSendPostsTheWirePayload(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	tracked := false
	sent, err := client.Emails.Send(context.Background(), SendEmailRequest{
		From:          "Acme <hello@acme.com>",
		To:            []string{"reader@yourdomain.com"},
		Subject:       "Hello from Go",
		HTML:          "<p>Hi</p>",
		Tags:          []Tag{{Name: "example", Value: "go"}},
		Headers:       map[string]string{"X-Entity-Ref-ID": "abc"},
		Attachments:   []Attachment{{Filename: "logo.png", Content: "aGk=", ContentType: "image/png", ContentID: "logo"}},
		TrackingClick: &tracked,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	request := mock.last(t)
	assertRoute(t, request, http.MethodPost, "/v1/emails")
	if request.ContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", request.ContentType)
	}
	assertBodyField(t, request, "from", "Acme <hello@acme.com>")
	assertBodyField(t, request, "subject", "Hello from Go")
	assertBodyField(t, request, "html", "<p>Hi</p>")
	assertBodyField(t, request, "tracking_click", false)

	recipients, _ := json.Marshal(request.Body["to"])
	if string(recipients) != `["reader@yourdomain.com"]` {
		t.Errorf(`body["to"] = %s, want ["reader@yourdomain.com"]`, recipients)
	}
	tags, _ := json.Marshal(request.Body["tags"])
	if string(tags) != `[{"name":"example","value":"go"}]` {
		t.Errorf(`body["tags"] = %s`, tags)
	}
	attachments, _ := json.Marshal(request.Body["attachments"])
	// Decoded and re-encoded by the mock, so the keys come back sorted.
	if string(attachments) != `[{"content":"aGk=","content_id":"logo","content_type":"image/png","filename":"logo.png"}]` {
		t.Errorf(`body["attachments"] = %s`, attachments)
	}
	// An unset field must not go out at all: the API reads "absent" and "false"
	// differently for tracking, and "" is not a valid scheduled_at.
	if _, present := request.Body["tracking_open"]; present {
		t.Error("tracking_open was sent when the caller left it unset")
	}
	if _, present := request.Body["scheduled_at"]; present {
		t.Error("scheduled_at was sent when the caller left it unset")
	}

	// The id is the whole point of the call: it is what you look the send up by
	// and what webhooks reference.
	if sent.ID != mockEmailID {
		t.Errorf("id = %q, want %q", sent.ID, mockEmailID)
	}
}

// The escape hatch matters as much as the named fields: a wire field this
// release does not know about must still be sendable.
func TestSendMergesExtraFields(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Emails.Send(context.Background(), SendEmailRequest{
		From:    "Acme <hello@acme.com>",
		To:      []string{"reader@yourdomain.com"},
		Subject: "Extras",
		Text:    "Extras",
		Extra:   Params{"idempotency_key": "key_1", "subject": "overridden"},
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	request := mock.last(t)
	assertBodyField(t, request, "idempotency_key", "key_1")
	assertBodyField(t, request, "subject", "overridden")
}

func TestSendSchedulesWithScheduledAt(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Emails.Send(context.Background(), SendEmailRequest{
		From:        "Acme <hello@acme.com>",
		To:          []string{"reader@yourdomain.com"},
		Subject:     "Later",
		Text:        "Later",
		ScheduledAt: "2026-09-01T09:00:00Z",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	assertBodyField(t, mock.last(t), "scheduled_at", "2026-09-01T09:00:00Z")
}

func TestBatchSendsABareArray(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	result, err := client.Emails.Batch(context.Background(), []SendEmailRequest{
		{From: "a@b.com", To: []string{"x@a.com"}, Subject: "1", HTML: "<p>1</p>"},
		{From: "a@b.com", To: []string{"y@a.com"}, Subject: "2", HTML: "<p>2</p>"},
	})
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}

	request := mock.last(t)
	assertRoute(t, request, http.MethodPost, "/v1/emails/batch")
	// The endpoint takes a bare array, not an object wrapping one.
	if request.Body != nil {
		t.Errorf("batch sent an object body: %s", request.RawBody)
	}
	if len(request.Array) != 2 {
		t.Fatalf("batch body had %d items, want 2: %s", len(request.Array), request.RawBody)
	}
	if len(result.Data) != 2 || result.Data[0].ID != mockEmailID {
		t.Errorf("result = %+v, want two ids", result.Data)
	}
}

func TestGetReturnsTheEmailAndAliasesStatus(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	email, err := client.Emails.Get(context.Background(), mockEmailID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	assertRoute(t, mock.last(t), http.MethodGet, "/v1/emails/"+mockEmailID)
	if email.LastEvent != "delivered" {
		t.Errorf("last_event = %q, want delivered", email.LastEvent)
	}
	// `status` is the friendly alias every other Mailtea client exposes; the
	// API only sends `last_event`.
	if email.Status != "delivered" {
		t.Errorf("status = %q, want it aliased from last_event", email.Status)
	}
	if email.OpenCount != 2 || email.ClickCount != 1 {
		t.Errorf("counters = %d/%d, want 2/1", email.OpenCount, email.ClickCount)
	}
	if len(email.DroppedRecipients) != 1 || email.DroppedRecipients[0].Reason != "suppressed" {
		t.Errorf("dropped_recipients = %+v, want the one suppressed address", email.DroppedRecipients)
	}
	if len(email.Attachments) != 1 || email.Attachments[0].Filename != "receipt.pdf" {
		t.Errorf("attachments = %+v", email.Attachments)
	}
}

// An id that came from somewhere less trustworthy must not walk out of the path
// segment it belongs in.
func TestGetEscapesTheIDIntoOnePathSegment(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Emails.Get(context.Background(), "../v1/api-keys"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := mock.last(t).EscapedPath; got != "/v1/emails/..%2Fv1%2Fapi-keys" {
		t.Errorf("path = %q, want the id escaped into one segment", got)
	}
}

func TestListSendsItsFiltersAsQueryParameters(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	list, err := client.Emails.List(context.Background(), Params{
		"status": "delivered",
		"limit":  50,
		"search": "receipt@example.com",
		"unset":  nil,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	request := mock.last(t)
	assertRoute(t, request, http.MethodGet, "/v1/emails")
	assertQuery(t, request, "status", "delivered")
	assertQuery(t, request, "limit", "50")
	assertQuery(t, request, "search", "receipt@example.com")
	if _, present := request.Query["unset"]; present {
		t.Error("a nil filter was sent; it should be dropped")
	}
	if list.Total != 1 || len(list.Data) != 1 || list.Data[0].String("id") != "obj_1" {
		t.Errorf("list = %+v, want the envelope decoded", list)
	}
	if list.NextCursor != "cur_2" {
		t.Errorf("next_cursor = %q, want cur_2", list.NextCursor)
	}
}

func TestAnalytics(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	analytics, err := client.Emails.Analytics(context.Background(), Params{"from_date": "2026-08-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("Analytics: %v", err)
	}

	request := mock.last(t)
	assertRoute(t, request, http.MethodGet, "/v1/emails/analytics")
	assertQuery(t, request, "from_date", "2026-08-01T00:00:00Z")
	if analytics.Int("delivered") != 2 {
		t.Errorf("delivered = %d, want 2", analytics.Int("delivered"))
	}
}

func TestUpdateAndRescheduleAndCancel(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Emails.Update(context.Background(), mockEmailID, UpdateEmailRequest{
		ScheduledAt: "2026-09-01T09:00:00Z",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/emails/"+mockEmailID)
	assertBodyField(t, mock.last(t), "scheduled_at", "2026-09-01T09:00:00Z")

	if _, err := client.Emails.Reschedule(context.Background(), mockEmailID, "2026-09-02T09:00:00Z"); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPatch, "/v1/emails/"+mockEmailID)
	assertBodyField(t, mock.last(t), "scheduled_at", "2026-09-02T09:00:00Z")

	canceled, err := client.Emails.Cancel(context.Background(), mockEmailID)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	// Cancel is a POST to /cancel. There is no DELETE on emails.
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/emails/"+mockEmailID+"/cancel")
	if mock.last(t).HasBody() {
		t.Errorf("cancel sent a body: %q", mock.last(t).RawBody)
	}
	// The reply is `{object, id}` — the API does NOT echo a `canceled` status
	// back, so the id is the whole confirmation. Asserting on a `last_event`
	// here would only be asserting on the mock.
	if canceled.String("id") != mockEmailID {
		t.Errorf("cancel returned %v, want the id it was given", canceled)
	}
	if canceled.String("last_event") != "" {
		t.Errorf("cancel returned a last_event (%q); the API sends none", canceled.String("last_event"))
	}
}

func TestInbound(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)
	ctx := context.Background()

	if _, err := client.Emails.Inbound.List(ctx, Params{"publication_id": "pub_1", "limit": 20}); err != nil {
		t.Fatalf("Inbound.List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/emails/inbound")
	assertQuery(t, mock.last(t), "publication_id", "pub_1")

	if _, err := client.Emails.Inbound.Get(ctx, "in_1"); err != nil {
		t.Fatalf("Inbound.Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/emails/inbound/in_1")

	if _, err := client.Emails.Inbound.Reply(ctx, "in_1", Params{"text": "Thanks!"}); err != nil {
		t.Fatalf("Inbound.Reply: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodPost, "/v1/emails/inbound/in_1/reply")
	assertBodyField(t, mock.last(t), "text", "Thanks!")

	if _, err := client.Emails.Inbound.Attachments.List(ctx, "in_1"); err != nil {
		t.Fatalf("Attachments.List: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/emails/inbound/in_1/attachments")

	if _, err := client.Emails.Inbound.Attachments.Get(ctx, "in_1", "att_1"); err != nil {
		t.Fatalf("Attachments.Get: %v", err)
	}
	assertRoute(t, mock.last(t), http.MethodGet, "/v1/emails/inbound/in_1/attachments/att_1")
}
