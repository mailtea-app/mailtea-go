package mailtea

// Version is this SDK's release. It is sent on every request as
// `User-Agent: mailtea-go/<Version>`, which is how a support request can be
// traced back to the client that made it.
//
// The mirror repo's `v<Version>` git tag IS the Go release — pkg.go.dev
// indexes the tag, there is no separate registry upload — so this constant and
// that tag must always agree.
const Version = "0.1.0"
