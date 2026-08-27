# Changelog

All notable changes to the `github.com/mailtea-app/mailtea-go` module are
documented here. The `v<version>` tag on this repository is the release —
pkg.go.dev indexes the tag — and these sections become its release notes.

## 0.1.0 (2026-08-27)

First release. The official Go SDK for Mailtea, ported from the Python SDK
resource for resource and path for path, with no dependencies outside the
standard library.

- `mailtea.New(apiKey, opts...)` — the key comes from the argument or
  `MAILTEA_API_KEY`; the base URL from `WithBaseURL`, `MAILTEA_API_BASE_URL`, or
  `https://api.mailtea.app`. `WithHTTPClient` injects a transport, so tests need
  no network.
- Every REST resource the API exposes: `Emails` (send, batch, get, list,
  analytics, update, reschedule, cancel, plus the `Inbound` sub-resource and its
  attachments), `Contacts`, `Segments`, `Topics`, `Posts`, `Senders`, `Assets`,
  `Suppressions` (including the CSV `Export`), `Templates`, `Domains` (and
  `Domains.Tracking`), `Webhooks`, `ContactProperties`, `APIKeys`,
  `Automations`, `AutomationRuns`, `Events` and `EventDefinitions`.
- Typed request structs for `Emails.Send`/`Batch`/`Update`,
  `Contacts.Create`/`Update`, `Posts.Create`/`Send`/`SendTest` and
  `Topics.Create`; every one has an `Extra` field so a wire field this release
  does not name yet is still sendable. Everything else takes `mailtea.Params`.
- `*mailtea.Error` on every failure, carrying the HTTP status, the API's own
  message, its `code` and `details`, and the `x-request-id`. Status `0` means
  the fault was on this side — a missing key, an unreachable host.
- `SignWebhook` and `VerifyWebhookSignature` — Standard Webhooks signing in
  exact parity with the platform's signer, with constant-time comparison and
  replay protection.
