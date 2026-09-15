package mailtea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// EmailsService is the `emails` resource. Reach it as client.Emails.
type EmailsService struct {
	client *Client

	// Inbound covers received mail: list, get, reply, and attachments.
	Inbound *InboundService
}

// Tag is a key/value label carried with a send, for filtering and analytics later.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Attachment is a file sent with an email. Content is base64. Set ContentType
// and a ContentID to embed an inline image referenced by `cid:` in the HTML;
// omit ContentID for an ordinary file attachment.
type Attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// TemplateRef seeds a send from a published server-side template instead of
// inline HTML. Variables fill the template's placeholders.
type TemplateRef struct {
	ID        string                 `json:"id"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// SendEmailRequest is the body of POST /v1/emails.
//
// Set the From with exactly one of From (a "Name <email>" string) or SenderID
// (a named, verified publication sender, which also supplies its default
// reply-to). Provide HTML/Text OR Template, never both.
//
// To, CC and BCC are capped at 50 recipients COMBINED — the provider refuses a
// larger message, so the API rejects it rather than accepting a send that dies
// downstream where you cannot see it.
type SendEmailRequest struct {
	// From is a verified sender, e.g. "Acme <hello@acme.com>".
	From string `json:"from,omitempty"`
	// SenderID selects a saved sender ("snd_…") instead of From.
	SenderID string `json:"sender_id,omitempty"`

	To      []string `json:"to"`
	Subject string   `json:"subject"`

	HTML string `json:"html,omitempty"`
	Text string `json:"text,omitempty"`
	// Template renders a published template server-side. Mutually exclusive
	// with HTML.
	Template *TemplateRef `json:"template,omitempty"`

	CC      []string `json:"cc,omitempty"`
	BCC     []string `json:"bcc,omitempty"`
	ReplyTo []string `json:"reply_to,omitempty"`

	// ScheduledAt queues the send for later, ISO 8601: "2026-09-01T09:00:00Z".
	ScheduledAt string `json:"scheduled_at,omitempty"`

	Tags    []Tag             `json:"tags,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	Attachments []Attachment `json:"attachments,omitempty"`

	// TrackingOpen and TrackingClick opt this message out of the open pixel or
	// out of rewritten links. They are pointers because "unset" and "false" are
	// different: unset means tracked, as it always has been. A sending domain
	// with tracking switched off cannot be overridden from here — policy
	// narrows, it never widens.
	TrackingOpen  *bool `json:"tracking_open,omitempty"`
	TrackingClick *bool `json:"tracking_click,omitempty"`

	// Extra carries wire fields this SDK version does not name yet. Its keys
	// are merged over the encoded struct, so a field the API adds tomorrow is
	// sendable today.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r SendEmailRequest) MarshalJSON() ([]byte, error) {
	type alias SendEmailRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// SendEmailResponse is what a send returns: the id you look the send up by, and
// the id webhooks reference.
type SendEmailResponse struct {
	Object string `json:"object"`
	ID     string `json:"id"`
}

// BatchResponse is what a batch send returns — one id per message, in order.
type BatchResponse struct {
	Data []SendEmailResponse `json:"data"`
}

// UpdateEmailRequest reschedules a scheduled email. ScheduledAt is currently
// the only field the API accepts.
type UpdateEmailRequest struct {
	ScheduledAt string `json:"scheduled_at,omitempty"`

	// Extra carries wire fields this SDK version does not name yet.
	Extra Params `json:"-"`
}

// MarshalJSON merges Extra over the named fields.
func (r UpdateEmailRequest) MarshalJSON() ([]byte, error) {
	type alias UpdateEmailRequest
	return marshalWithExtra(alias(r), r.Extra)
}

// DroppedRecipient is an address the message did not reach, and why. To/CC/BCC
// are what was ASKED for; a suppressed or unusable address is filtered out of
// the envelope but left in those fields, so without reading this a partially
// delivered send looks identical to a fully delivered one.
type DroppedRecipient struct {
	Address string `json:"address"`
	Field   string `json:"field"`
	Reason  string `json:"reason"`
}

// AttachmentMeta describes an attachment on a retrieved email. The bytes are
// not returned.
type AttachmentMeta struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// Email is one send, as returned by Get.
type Email struct {
	Object  string `json:"object"`
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	CC      string `json:"cc"`
	BCC     string `json:"bcc"`
	ReplyTo string `json:"reply_to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`

	// LastEvent is where the send got to: queued, scheduled, sent, delivered,
	// delivery_delayed, bounced, complained, failed, suppressed, canceled.
	// Reading it is how you check on a send without setting up a webhook.
	LastEvent string `json:"last_event"`
	// Status is a friendly alias of LastEvent, filled in by this SDK.
	Status string `json:"status"`

	// Mode is "live" for real mail, or "test" for a message sent with a test key
	// (mt_test_...): validated, recorded and webhook-emitting, but never
	// delivered. A string rather than a typed constant, so a mode added
	// server-side still decodes.
	Mode string `json:"mode"`

	// Error is why the send failed, in neutral words — the provider's own
	// wording is never returned. Empty on every email that has not failed.
	Error string `json:"error"`

	DroppedRecipients []DroppedRecipient `json:"dropped_recipients"`

	CreatedAt   string `json:"created_at"`
	ScheduledAt string `json:"scheduled_at"`
	FailedAt    string `json:"failed_at"`
	DelayedAt   string `json:"delayed_at"`
	OpenedAt    string `json:"opened_at"`
	OpenCount   int    `json:"open_count"`
	ClickedAt   string `json:"clicked_at"`
	ClickCount  int    `json:"click_count"`

	Tags        []Tag             `json:"tags"`
	Headers     map[string]string `json:"headers"`
	Attachments []AttachmentMeta  `json:"attachments"`
}

// Send sends one transactional email, or schedules it when ScheduledAt is set.
func (s *EmailsService) Send(ctx context.Context, request SendEmailRequest) (*SendEmailResponse, error) {
	var out SendEmailResponse
	if err := s.client.call(ctx, http.MethodPost, "/v1/emails", request, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Batch sends up to 100 emails in one request. The body is a bare array, which
// is what this endpoint takes — not an object wrapping one.
func (s *EmailsService) Batch(ctx context.Context, requests []SendEmailRequest) (*BatchResponse, error) {
	var out BatchResponse
	if err := s.client.call(ctx, http.MethodPost, "/v1/emails/batch", requests, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get retrieves an email with its delivery status and tracking counters.
//
// The id goes through url.PathEscape: a real "txemail_…" passes through
// untouched, and an id from somewhere less trustworthy cannot walk out of the
// path segment it belongs in.
func (s *EmailsService) Get(ctx context.Context, id string) (*Email, error) {
	var out Email
	if err := s.client.call(ctx, http.MethodGet, "/v1/emails/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	if out.Status == "" {
		out.Status = out.LastEvent
	}
	return &out, nil
}

// List lists emails, most recent first. Optional filters: status, tag_name,
// tag_value, search (substring match on recipient/sender/subject), from_date,
// to_date, limit, offset.
//
// from_date is clamped to the plan's analytics retention window — 30 days on
// most plans, 90 on Scale and Enterprise. A value reaching further back returns
// data from the start of that window rather than an error, and omitting it
// returns the window rather than all time.
func (s *EmailsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/emails"+query(params), nil)
}

// Analytics aggregates transactional metrics over an optional date window:
// totals, delivered/bounced/open/click counts, per-status counts, and rates.
// Optional filters: from_date, to_date (ISO 8601), clamped like List's.
func (s *EmailsService) Analytics(ctx context.Context, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodGet, "/v1/emails/analytics"+query(params), nil)
}

// Update changes a scheduled email (currently only scheduled_at).
func (s *EmailsService) Update(ctx context.Context, id string, request UpdateEmailRequest) (Object, error) {
	return s.client.object(ctx, http.MethodPatch, "/v1/emails/"+url.PathEscape(id), request)
}

// Reschedule is Update for the one case it exists for.
func (s *EmailsService) Reschedule(ctx context.Context, id, scheduledAt string) (Object, error) {
	return s.Update(ctx, id, UpdateEmailRequest{ScheduledAt: scheduledAt})
}

// Cancel stops a scheduled email before it sends.
//
// It works only while the email is still `scheduled`. Any other status —
// including the `queued` of an ordinary immediate send — answers 422, so treat
// that as "too late to stop it" rather than as a bug. There is no DELETE on
// emails; cancel is this POST.
//
// The reply is `{object, id}` and nothing more — a 2xx IS the confirmation.
// Call Get if you want to read the resulting `canceled` status back.
func (s *EmailsService) Cancel(ctx context.Context, id string) (Object, error) {
	return s.client.object(ctx, http.MethodPost, "/v1/emails/"+url.PathEscape(id)+"/cancel", nil)
}

// compile-time proof that the request types encode through their MarshalJSON.
var (
	_ json.Marshaler = SendEmailRequest{}
	_ json.Marshaler = UpdateEmailRequest{}
)
