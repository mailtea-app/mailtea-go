# mailtea-go

The official Go SDK for [Mailtea](https://mailtea.app) — a thin, typed wrapper
over the [REST API](https://docs.mailtea.app/docs/api-reference).

No dependencies outside the standard library. Go 1.18 or newer.

## Install

```bash
go get github.com/mailtea-app/mailtea-go@v0.1.0
```

```go
import "github.com/mailtea-app/mailtea-go" // package mailtea
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mailtea-app/mailtea-go"
)

func main() {
	client, err := mailtea.New(os.Getenv("MAILTEA_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	sent, err := client.Emails.Send(ctx, mailtea.SendEmailRequest{
		From:    "you@yourdomain.com",
		To:      []string{"recipient@example.com"},
		Subject: "Hello from Mailtea",
		HTML:    "<p>Your first email, sent with <strong>Mailtea</strong>.</p>",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sent.ID)

	email, err := client.Emails.Get(ctx, sent.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(email.Status) // queued, sent, delivered, bounced, …
}
```

A single message is capped at **50 recipients combined** across `to` + `cc` +
`bcc`; the API rejects a larger one rather than accepting a send that fails
downstream.

## Configuration

| What | How |
| --- | --- |
| API key | `mailtea.New("mt_pat_…")`, or `mailtea.New("")` to read `MAILTEA_API_KEY` |
| Base URL | `mailtea.New(key, mailtea.WithBaseURL("http://127.0.0.1:7787"))`, or `MAILTEA_API_BASE_URL` |
| Transport | `mailtea.New(key, mailtea.WithHTTPClient(myClient))` — anything with `Do(*http.Request) (*http.Response, error)` |

An explicit option beats the environment variable, which beats the default
(`https://api.mailtea.app`). `New` returns a `*mailtea.Error` with
`Code: "missing_api_key"` when there is no key at all, rather than letting the
misconfiguration surface as a 401 on your first send.

Pass a custom client to add a proxy, a timeout, retries — or, in a test, to
answer without a network:

```go
client, err := mailtea.New("mt_pat_test", mailtea.WithHTTPClient(fakeDoer{}))
```

## API

Every method takes a `context.Context` first and returns `(result, error)`.

Payloads follow the REST wire format (`reply_to`, `scheduled_at`, …). The
methods below marked **typed** take a request struct; every other method takes
`mailtea.Params`, a `map[string]interface{}` of wire-format keys. Each typed
struct also has an `Extra mailtea.Params` field whose keys are merged over the
named ones, so a field the API adds tomorrow is sendable today.

Responses that this SDK does not type are `mailtea.Object` — a map with typed
accessors (`String`, `Int`, `Bool`, `Object`, `List`) and `Decode(&yourStruct)`.
List endpoints return `*mailtea.List` (`Data []Object`, plus `Total`/`Limit`/
`Offset`/`HasMore` for offset pagination or `NextCursor` for cursor pagination).

| Method | Description |
| --- | --- |
| `Emails.Send(ctx, req)` **typed** | Send a transactional email → `{ID}` |
| `Emails.Batch(ctx, reqs)` **typed** | Send up to 100 emails → `{Data: [{ID}]}` |
| `Emails.Get(ctx, id)` | Retrieve an email and its delivery status (typed `*Email`) |
| `Emails.List(ctx, params)` | List emails → `*List`. Pass `"mode": "test"` for test-mode mail |
| `Emails.Update(ctx, id, req)` **typed** | Reschedule a scheduled email |
| `Emails.Reschedule(ctx, id, scheduledAt)` | Convenience wrapper over `Update` |
| `Emails.Cancel(ctx, id)` | Cancel a scheduled email (`POST …/cancel`; there is no `DELETE`) |
| `Emails.Analytics(ctx, params)` | Aggregate transactional metrics over an optional date window |
| `Emails.Inbound.List / Get / Reply` | Received emails; `Reply` threads by construction |
| `Emails.Inbound.Attachments.List / Get` | Attachments on a received email, with signed download URLs |
| `Contacts.Create / Upsert` **typed** | Create or update a contact (the endpoint upserts) |
| `Contacts.Update` **typed** | Change a contact's status |
| `Contacts.List / Get / Delete` | Manage audience contacts |
| `Posts.Create(ctx, req)` **typed** | Create a newsletter post (draft, or `Send: true`) |
| `Posts.Send(ctx, id, req)` **typed** | Send a draft post to the audience, now or scheduled |
| `Posts.SendTest(ctx, id, req)` **typed** | Send a `[TEST]` copy → `{SentTo, FailedTo}` |
| `Posts.List / Get / Update / Delete` | Manage posts |
| `Segments.Create / List / Get / Update / Delete` | Manage audience segments |
| `Topics.Create(ctx, req)` **typed** | Create a topic definition (`opt_in` / `opt_out`) |
| `Topics.List / Get / Update / Delete` | Manage topic definitions |
| `Senders.Create / List / Get / Update / Delete` | Manage named From identities (`email` immutable) |
| `Assets.Upload / List / Delete` | The publication's image library (`content` accepts raw `[]byte`) |
| `Suppressions.List / Add / Remove` | Manage the team-wide do-not-send list |
| `Suppressions.Export(ctx)` | Export the whole list as CSV (raw `string`, not JSON) |
| `Templates.Create / List / Get / Update / Delete` | Manage reusable email templates |
| `Templates.Render(ctx, params)` | Render a spec to HTML without saving → `{html, text}` |
| `Templates.Publish / Unpublish / Duplicate` | Template lifecycle |
| `Templates.Versions / RestoreVersion` | Design history; restoring returns the template to **draft** |
| `Domains.Create / List / Get / Verify / Update / Delete` | Manage sending domains |
| `Domains.Tracking.Create / List / Verify / Delete` | CNAME tracking sub-domains under a domain |
| `Webhooks.Create / List / Get / Update / Delete` | Manage outbound event subscriptions |
| `ContactProperties.Create / List / Update / Delete` | Custom contact fields (team-scoped) |
| `APIKeys.Create / List / Revoke` | Manage API keys (needs `settings:write`). `"mode": "test"` mints a test key |
| `Automations.Create / List / Get / Update / Delete` | Automation graphs (`steps` + optional `connections`) |
| `Automations.Validate(ctx, params)` | Dry-run a graph → `{valid, issues}` |
| `Automations.Activate / Pause / Archive` | Lifecycle (`cancel_runs` defaults **false** on pause, **true** on archive) |
| `Automations.Versions / Version / Metrics` | Stored versions and per-step funnel counts |
| `Automations.Test(ctx, id, params)` | One test run against a real contact — **sends real, billed email** |
| `AutomationRuns.List / Get / Cancel` | Inspect and cancel runs (a run pins the version it started on) |
| `Events.Send / List` | Record and list custom product events |
| `EventDefinitions.Create / List / Get / Update / Delete` | The event catalog (`name` immutable) |

`Emails.Send` also takes `Tags`, custom `Headers`, `Attachments`, and
`ScheduledAt`. Attachments carry base64 `Content`; set a `ContentID` (plus
`ContentType`) to embed an inline image referenced by `cid:` in the HTML:

```go
_, err := client.Emails.Send(ctx, mailtea.SendEmailRequest{
	From:    "you@yourdomain.com",
	To:      []string{"recipient@example.com"},
	Subject: "Your receipt",
	HTML:    `<p>Thanks!</p><img src="cid:logo" />`,
	Tags:    []mailtea.Tag{{Name: "category", Value: "receipt"}},
	Attachments: []mailtea.Attachment{
		{Filename: "receipt.pdf", Content: pdfBase64},
		{Filename: "logo.png", Content: logoBase64, ContentType: "image/png", ContentID: "logo"}, // inline
	},
})
```

## Test mode

A test key (`mt_test_…`) sends nothing. Every message it creates is validated,
recorded and emits webhooks, but is never handed to a provider — so CI can point
at production Mailtea with your real code and your real webhook handler.

```go
key, err := client.APIKeys.Create(ctx, mailtea.Params{"name": "CI", "mode": "test"})
// key["token"] starts with mt_test_

test, err := mailtea.New(key["token"].(string))
_, err = test.Emails.Send(ctx, mailtea.SendEmailRequest{
	From:    "you@yourdomain.com",
	To:      []string{"bounced@test.mailtea.email"},
	Subject: "Bounce handling",
	HTML:    "<p>Never delivered.</p>",
})

list, err := test.Emails.List(ctx, mailtea.Params{"mode": "test"})
```

Reserved recipients on `test.mailtea.email` force the outcome — `delivered@`,
`bounced@`, `complained@`, `delayed@`, `failed@` — and the first `To` recipient
decides. `Email.Mode` reports which mode a row was written in. A test key reads
only test mail and a live key only live mail; there is no mixed view.

A test key is **not** a data sandbox. It reads and writes your real contacts,
templates, senders and webhooks. Only delivery is simulated.

## Webhooks

Mailtea signs every outbound webhook with
[Standard Webhooks](https://www.standardwebhooks.com/).
`VerifyWebhookSignature` checks the signature and rejects replays. Pass the
**raw** request body — not re-serialized JSON — and the endpoint's `whsec_…`
signing secret:

```go
func handler(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ok := mailtea.VerifyWebhookSignature(
		signingSecret, // whsec_… returned once by Webhooks.Create
		r.Header.Get("webhook-id"),
		r.Header.Get("webhook-timestamp"),
		string(raw),
		r.Header.Get("webhook-signature"),
	)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// … handle the event
}
```

`SignWebhook(secret, msgID, timestamp, payload)` produces the same header, handy
for faking deliveries in tests. `VerifyWebhookSignatureAt` takes the tolerance
and the current time explicitly, so a test can prove an expired delivery is
rejected without waiting five minutes.

## Errors

Every failure is a `*mailtea.Error`, reachable with `errors.As`:

```go
var apiErr *mailtea.Error
if errors.As(err, &apiErr) {
	log.Printf("status=%d code=%s request_id=%s: %s",
		apiErr.Status, apiErr.Code, apiErr.RequestID, apiErr.Message)
}
```

| Field | What it carries |
| --- | --- |
| `Status` | HTTP status code. **`0` means the fault was on this side** — a missing key, an unreachable host, an undecodable body |
| `Message` | The API's own `error` field ("Domain is not verified"), or the client-side reason |
| `Code` | The API's machine-readable code, when it sends one. Branching on this survives a copy change to `Message` |
| `Details` | The validation issue list naming the fields that failed |
| `RequestID` | The response's `x-request-id` — quote it in a support request |
| `Body` | The raw response body, verbatim |

Surfacing `Message` is the difference between "the send failed" and "the domain
isn't verified yet".

## Local development

```bash
git clone https://github.com/mailtea-app/mailtea-go
cd mailtea-go
go test ./...
```

The tests answer from a bundled `httptest` mock of the Mailtea API, so they need
no API key and make no network calls. One of them checks endpoint parity with
the Python SDK: every `/v1/…` path the reference client reaches, this one must
reach too.

To run against a Mailtea on your own machine:

```bash
export MAILTEA_API_KEY="mt_pat_…"
export MAILTEA_API_BASE_URL="http://127.0.0.1:7787"
```

## License

MIT. See [LICENSE](LICENSE).
