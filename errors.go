package mailtea

import (
	"fmt"
	"strings"
)

// Error is returned whenever the Mailtea API answers with a non-2xx status, or
// the client is misconfigured before a request is even attempted.
//
// Reach it with errors.As:
//
//	var apiErr *mailtea.Error
//	if errors.As(err, &apiErr) && apiErr.Status == 422 {
//	    // too late to cancel
//	}
type Error struct {
	// Status is the HTTP status code. Zero means the failure happened on this
	// side of the wire — a missing API key, an unreachable host, a response
	// body that was not the JSON it claimed to be — so there is no status to
	// report. Branch on it before assuming the API said anything at all.
	Status int

	// Message is the API's own `error` field ("Domain is not verified",
	// "Validation failed"), or the client-side reason when Status is 0.
	Message string

	// Code is the API's machine-readable code, when it sends one (for example
	// `marketing_plan_required` on a 402, or `template_version_not_found`).
	// Branching on Code survives a copy change to Message. Empty when absent.
	Code string

	// Details is the validation issue list the API returns alongside a 400,
	// naming the fields that failed. Nil when absent. It is decoded as
	// free-form JSON because its shape varies per endpoint; use DetailsJSON to
	// print it or json.Unmarshal it into your own type.
	Details interface{}

	// RequestID is the response's `x-request-id` header — quote it in a support
	// request and the exact call can be found.
	RequestID string

	// Body is the raw response body, kept verbatim. The parsed fields above are
	// what you branch on; this is what you log when they were not enough.
	Body string
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("mailtea: ")
	if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		b.WriteString("request failed")
	}

	parts := make([]string, 0, 3)
	if e.Status > 0 {
		parts = append(parts, fmt.Sprintf("status %d", e.Status))
	}
	if e.Code != "" {
		parts = append(parts, "code "+e.Code)
	}
	if e.RequestID != "" {
		parts = append(parts, "request id "+e.RequestID)
	}
	if len(parts) > 0 {
		b.WriteString(" (")
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString(")")
	}
	return b.String()
}
