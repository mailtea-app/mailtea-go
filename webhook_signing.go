package mailtea

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"math"
	"strconv"
	"strings"
	"time"
)

// Standard Webhooks (standardwebhooks.com) signature verification.
//
// A stdlib-only mirror of the Mailtea signer, kept in exact parity so a
// signature produced by the platform verifies here byte for byte.
//
// The stored signing secret is `whsec_<base64>`; the HMAC key is the base64
// remainder decoded to bytes. The signed content is
// `{msg_id}.{timestamp}.{payload}` where timestamp is Unix SECONDS, matching
// the `webhook-timestamp` header. The `webhook-signature` header is
// `v1,<base64 HMAC-SHA256>`; during key rotation it may carry several
// space-delimited `v1,<sig>` tokens and a match against any one of them passes.

const (
	secretPrefix     = "whsec_"
	signatureVersion = "v1"

	// DefaultWebhookTolerance is how far the delivery's timestamp may sit from
	// now, each way, before it is treated as a replay. Five minutes, the
	// Standard Webhooks default.
	DefaultWebhookTolerance = 5 * time.Minute
)

// decodeSigningKey decodes the HMAC key from a whsec_-prefixed secret.
//
// It matches Node's lenient base64 decoder: the base64url alphabet is accepted
// and missing padding tolerated, so a secret minted with either alphabet
// decodes to the same bytes.
func decodeSigningKey(secret string) []byte {
	raw := secret
	if strings.HasPrefix(raw, secretPrefix) {
		raw = raw[len(secretPrefix):]
	}
	raw = strings.TrimSpace(raw)
	raw = strings.NewReplacer("-", "+", "_", "/").Replace(raw)
	if pad := len(raw) % 4; pad != 0 {
		raw += strings.Repeat("=", 4-pad)
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		// A secret that is not base64 cannot produce a matching signature, and
		// returning an error here would push a "can this even be parsed?"
		// branch into every caller. An unusable key simply never verifies.
		return nil
	}
	return key
}

func computeSignature(secret, msgID string, timestamp int64, payload string) string {
	signedContent := msgID + "." + strconv.FormatInt(timestamp, 10) + "." + payload
	mac := hmac.New(sha256.New, decodeSigningKey(secret))
	mac.Write([]byte(signedContent))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// SignWebhook signs a payload and returns the `webhook-signature` header value
// in Standard Webhooks form, `v1,<base64 HMAC-SHA256>`. Useful for faking
// Mailtea deliveries in tests.
//
// timestamp is Unix seconds — the same value sent in `webhook-timestamp`.
func SignWebhook(secret, msgID string, timestamp int64, payload string) string {
	return signatureVersion + "," + computeSignature(secret, msgID, timestamp, payload)
}

// VerifyWebhookSignature checks a `webhook-signature` header against the
// expected HMAC.
//
//	ok := mailtea.VerifyWebhookSignature(
//	    signingSecret,                       // whsec_… from Webhooks.Create
//	    r.Header.Get("webhook-id"),
//	    r.Header.Get("webhook-timestamp"),
//	    string(rawBody),                     // exact bytes received, not re-serialized
//	    r.Header.Get("webhook-signature"),
//	)
//
// The header may carry several space-delimited `v1,<sig>` tokens — Standard
// Webhooks allows key rotation, and the platform may sign one delivery with
// both the old and the new secret — so a match against any `v1` token passes.
//
// It returns false when the timestamp is outside DefaultWebhookTolerance of
// now, which is the replay protection. The comparison is constant-time, and a
// bad signature is a false rather than an error.
func VerifyWebhookSignature(secret, msgID, timestamp, payload, signatureHeader string) bool {
	return VerifyWebhookSignatureAt(secret, msgID, timestamp, payload, signatureHeader, DefaultWebhookTolerance, time.Now())
}

// VerifyWebhookSignatureAt is VerifyWebhookSignature with the tolerance and the
// current time supplied — the injectable form, so a test can prove that an
// expired timestamp is rejected without sleeping for five minutes.
func VerifyWebhookSignatureAt(
	secret, msgID, timestamp, payload, signatureHeader string,
	tolerance time.Duration,
	now time.Time,
) bool {
	timestampSeconds, err := parseUnixSeconds(timestamp)
	if err != nil {
		return false
	}

	skew := now.Unix() - timestampSeconds
	if skew < 0 {
		skew = -skew
	}
	if skew > int64(tolerance/time.Second) {
		return false
	}

	expected := computeSignature(secret, msgID, timestampSeconds, payload)

	for _, token := range strings.Split(signatureHeader, " ") {
		if token == "" {
			continue
		}
		version, signature, found := strings.Cut(token, ",")
		if !found || version != signatureVersion {
			continue
		}
		// Constant-time: a byte-by-byte early return leaks how much of the
		// signature was right, which is enough to forge one a byte at a time.
		if hmac.Equal([]byte(signature), []byte(expected)) {
			return true
		}
	}
	return false
}

// parseUnixSeconds reads the `webhook-timestamp` header. It is Unix seconds,
// but a float is accepted and floored — the Node and Python signers both floor
// before signing, so a fractional value must land on the same integer here.
func parseUnixSeconds(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return seconds, nil
	}
	asFloat, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(asFloat) || math.IsInf(asFloat, 0) {
		return 0, strconv.ErrRange
	}
	// Floor, not truncate: the Node and Python signers both floor before
	// signing, so a fractional value has to land on the same integer here.
	return int64(math.Floor(asFloat)), nil
}
