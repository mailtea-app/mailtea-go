package mailtea

import (
	"context"
	"net/http"
	"net/url"
)

// inboundBase is the prefix every inbound route hangs off.
const inboundBase = "/v1/emails/inbound"

// InboundService covers inbound (received) emails. Reach it as
// client.Emails.Inbound.
//
// List and retrieve mail delivered to your receiving domains, download
// attachments, and Reply — which threads correctly by construction and reuses
// the transactional send pipeline. Scoped to a publication: pass
// publication_id to List.
type InboundService struct {
	client *Client

	// Attachments covers the files on a received email.
	Attachments *InboundAttachmentsService
}

// List lists received emails in a publication, most recent first,
// cursor-paginated. Takes publication_id, optional limit (1-100, default 20)
// and cursor.
func (s *InboundService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, inboundBase+query(params), nil)
}

// Get retrieves a single received email, including its body, headers, and
// attachments.
func (s *InboundService) Get(ctx context.Context, id string) (Object, error) {
	return s.client.object(ctx, http.MethodGet, inboundBase+"/"+url.PathEscape(id), nil)
}

// Reply replies to a received email. The reply target (`to`), threading
// headers, and the "Re: " subject default are all server-derived — pass only
// the content (html/text, and optionally from, subject, cc, bcc,
// idempotency_key). Returns the resulting transactional email's id and status.
func (s *InboundService) Reply(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodPost, inboundBase+"/"+url.PathEscape(id)+"/reply", bodyOrNil(params))
}

// InboundAttachmentsService covers attachments on a received email. Reach it as
// client.Emails.Inbound.Attachments. Each returned object carries a short-lived
// signed download_url.
type InboundAttachmentsService struct {
	client *Client
}

// List lists an inbound email's attachments, each with a signed download URL.
func (s *InboundAttachmentsService) List(ctx context.Context, id string) (Object, error) {
	return s.client.object(ctx, http.MethodGet, inboundBase+"/"+url.PathEscape(id)+"/attachments", nil)
}

// Get retrieves a single inbound attachment with a signed download URL.
func (s *InboundAttachmentsService) Get(ctx context.Context, id, attachmentID string) (Object, error) {
	return s.client.object(
		ctx,
		http.MethodGet,
		inboundBase+"/"+url.PathEscape(id)+"/attachments/"+url.PathEscape(attachmentID),
		nil,
	)
}
