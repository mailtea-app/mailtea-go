# Changelog

All notable changes to the `github.com/mailtea-app/mailtea-go` module are
documented here. The `v<version>` tag on this repository is the release —
pkg.go.dev indexes the tag — and these sections become its release notes.

## Unreleased

- Added: `Domains.Update` with `tracking_subdomain` set to nil removes a
  tracking subdomain. The domain's links go back to being served from the
  Mailtea host. Links in mail you have already sent point at the old hostname
  and stop resolving — there is no way to reinstate them. Params reaches the
  encoder as given, so the nil travels as a JSON null; leaving the key out and
  setting it to nil are different requests. An empty string is neither: it is
  refused with `tracking_subdomain_invalid`.
- Changed: the `MX` row in `records` now reports what the last verify found,
  instead of reading `pending` on every request but the verify itself. A domain
  nobody has verified reads `not_started`.

## 0.2.0 (2026-09-03)

- Added: the domain claims resource — `client.Domains.Claims.Create`, `.Get`,
  `.Verify` and `.Cancel`. When adding a domain is refused because the host is
  connected to another publication, publish one TXT record to prove you control
  its DNS and the domain moves to you.
- Documented: domains take `region` (fixed at creation), `tls` and
  `tracking_subdomain` on create, and the list filters on `region` and `status`.
  This SDK forwards whatever parameters you pass, so these worked already — this
  release is where they are stated and covered by tests.

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
