package mailtea

import (
	"strconv"
	"testing"
	"time"
)

// Standard Webhooks signature verification — cross-implementation parity.
//
// The expected signature headers below were produced by the ACTUAL TypeScript
// signer (packages/contracts/src/webhook-signing.ts in the Mailtea monorepo,
// via its byte-for-byte copy sdks/webhook-ingester/src/webhook-signing.ts), so
// this Go implementation is checked against real platform output rather than
// against itself. They are the same vectors the Python SDK's tests use.
//
//	SECRET_A       = "whsec_dGVzdC1zaWduaW5nLWtleS0zMi1ieXRlcy1sb25n"
//	SECRET_B       = "whsec_cm90YXRpb24tc2VjcmV0LTMyLWJ5dGVzLWxvbmctdGVzdA"
//	MSG_ID         = "msg_2xyzABC123"
//	TIMESTAMP      = 1700000000
//	PAYLOAD        = `{"id":"evt_test","type":"email.delivered","data":{"email_id":"txemail_1"}}`

const (
	secretA        = "whsec_dGVzdC1zaWduaW5nLWtleS0zMi1ieXRlcy1sb25n"
	secretB        = "whsec_cm90YXRpb24tc2VjcmV0LTMyLWJ5dGVzLWxvbmctdGVzdA"
	msgID          = "msg_2xyzABC123"
	signedAt int64 = 1700000000
	payload        = `{"id":"evt_test","type":"email.delivered","data":{"email_id":"txemail_1"}}`

	headerA = "v1,txVPz2dd9nKoJKXT5SM0Ryfb0Tb5H+HRpz4nI/xpw+Q="
	headerB = "v1,c1hy0+MUcGBKxkM1Y96IQmkXAe2lGJY2AeZIQvKBJvg="
	// During rotation the platform signs with both secrets and sends both tokens.
	rotationHeader = headerA + " " + headerB
)

func at(offset int64) time.Time { return time.Unix(signedAt+offset, 0) }

func verifyAt(secret, timestamp, body, header string, now time.Time) bool {
	return VerifyWebhookSignatureAt(secret, msgID, timestamp, body, header, DefaultWebhookTolerance, now)
}

func TestSignMatchesTheTypeScriptVectors(t *testing.T) {
	if got := SignWebhook(secretA, msgID, signedAt, payload); got != headerA {
		t.Errorf("sign(A) = %q, want %q", got, headerA)
	}
	if got := SignWebhook(secretB, msgID, signedAt, payload); got != headerB {
		t.Errorf("sign(B) = %q, want %q", got, headerB)
	}
}

func TestVerifyAcceptsATypeScriptSignedHeader(t *testing.T) {
	// The webhook-timestamp header arrives as a string, which is the only form
	// a real handler ever has.
	if !verifyAt(secretA, strconv.FormatInt(signedAt, 10), payload, headerA, at(0)) {
		t.Error("a header signed by the platform did not verify")
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	header := SignWebhook(secretA, msgID, signedAt, payload)
	if !verifyAt(secretA, strconv.FormatInt(signedAt, 10), payload, header, at(0)) {
		t.Error("a header this SDK just signed did not verify")
	}
}

func TestVerifyRejectsTamperedPayloadAndWrongSecret(t *testing.T) {
	// One trailing space is the whole tamper — the raw body must be passed
	// through byte for byte, never re-serialized.
	if verifyAt(secretA, strconv.FormatInt(signedAt, 10), payload+" ", headerA, at(0)) {
		t.Error("a tampered payload verified")
	}
	if verifyAt(secretB, strconv.FormatInt(signedAt, 10), payload, headerA, at(0)) {
		t.Error("a header signed with a different secret verified")
	}
}

func TestVerifyRejectsGarbageAndTheWrongVersion(t *testing.T) {
	for name, header := range map[string]string{
		"empty":         "",
		"no comma":      "not-a-signature",
		"wrong version": "v2," + headerA[3:],
		"empty tokens":  "  ",
	} {
		t.Run(name, func(t *testing.T) {
			if verifyAt(secretA, strconv.FormatInt(signedAt, 10), payload, header, at(0)) {
				t.Errorf("header %q verified", header)
			}
		})
	}
}

func TestVerifyRejectsAnUnparseableTimestamp(t *testing.T) {
	for _, timestamp := range []string{"", "not-a-number", "Inf", "NaN"} {
		if verifyAt(secretA, timestamp, payload, headerA, at(0)) {
			t.Errorf("timestamp %q verified", timestamp)
		}
	}
}

// Replay protection: the signature stays valid forever, so the timestamp is
// what stops a captured delivery being re-sent tomorrow.
func TestVerifyToleranceBoundary(t *testing.T) {
	stamp := strconv.FormatInt(signedAt, 10)
	if !verifyAt(secretA, stamp, payload, headerA, at(300)) {
		t.Error("exactly at the 300s tolerance must pass")
	}
	if verifyAt(secretA, stamp, payload, headerA, at(301)) {
		t.Error("one second past the tolerance must fail")
	}
	// Skew runs both ways: a delivery timestamped in the future is just as
	// suspect as one from last week.
	if verifyAt(secretA, stamp, payload, headerA, at(-301)) {
		t.Error("a timestamp 301s in the future must fail")
	}
	if verifyAt(secretA, stamp, payload, headerA, at(10000)) {
		t.Error("an expired timestamp must fail")
	}

	if !VerifyWebhookSignatureAt(secretA, msgID, stamp, payload, headerA, 3*time.Hour, at(5000)) {
		t.Error("a custom tolerance must be honoured")
	}
}

// The platform signs one delivery with the old AND the new secret while a key
// is rotating, so an endpoint holding either one has to accept it.
func TestVerifyAcceptsEitherTokenDuringRotation(t *testing.T) {
	stamp := strconv.FormatInt(signedAt, 10)
	if !verifyAt(secretA, stamp, payload, rotationHeader, at(0)) {
		t.Error("the old secret did not match the rotation header's first token")
	}
	if !verifyAt(secretB, stamp, payload, rotationHeader, at(0)) {
		t.Error("the new secret did not match the rotation header's second token")
	}
}

// A secret minted with the base64url alphabet, or without padding, must decode
// to the same key — Node's decoder accepts both and this one has to agree. A
// mismatch here is invisible until the one endpoint whose secret happened to
// contain a `+` starts rejecting every delivery.
func TestSigningKeyDecodesLenientlyLikeNode(t *testing.T) {
	// secretB is already an UNPADDED secret (46 base64 characters), and it
	// reproduces the TypeScript vector above — that is the padding half.
	// This is the alphabet half: the same bytes spelled with +/ and with -_.
	standard := SignWebhook("whsec_++//++//", msgID, signedAt, payload)
	urlSafe := SignWebhook("whsec_--__--__", msgID, signedAt, payload)
	if standard != urlSafe {
		t.Errorf("base64 and base64url spellings of one key signed differently: %q vs %q", standard, urlSafe)
	}

	// A secret without the whsec_ prefix is treated as the raw base64 key.
	if got := SignWebhook("dGVzdC1zaWduaW5nLWtleS0zMi1ieXRlcy1sb25n", msgID, signedAt, payload); got != headerA {
		t.Errorf("sign(unprefixed) = %q, want %q", got, headerA)
	}
}

// VerifyWebhookSignature is the form a handler actually calls; it must agree
// with the injectable one for a delivery signed right now.
func TestVerifyWebhookSignatureUsesTheRealClock(t *testing.T) {
	now := time.Now().Unix()
	header := SignWebhook(secretA, msgID, now, payload)
	if !VerifyWebhookSignature(secretA, msgID, strconv.FormatInt(now, 10), payload, header) {
		t.Error("a delivery signed now did not verify against the real clock")
	}
	if VerifyWebhookSignature(secretA, msgID, strconv.FormatInt(now-3600, 10), payload, header) {
		t.Error("an hour-old timestamp verified against the real clock")
	}
}
