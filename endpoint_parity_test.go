package mailtea

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The canonical Mailtea endpoint set, as reached by the reference client — the
// Python SDK, github.com/mailtea-app/mailtea-python. It was extracted from that
// SDK's source with:
//
//	grep -ohE '"/v1/[^"]*"' sdks/python/mailtea/*.py | sort -u
//
// The list is hardcoded rather than computed so this repo stays standalone: the
// mirror has no Python SDK to read. A trailing slash marks a path that
// continues with an id ("/v1/emails/" is GET /v1/emails/{id}), which is exactly
// the form the literal takes in both SDKs.
//
// This SDK must reach EVERY one of these. Missing one is a Go caller who has to
// drop to raw HTTP for something every other Mailtea client does for them.
var pythonEndpointPrefixes = []string{
	"/v1/api-keys",
	"/v1/api-keys/",
	"/v1/assets",
	"/v1/assets/",
	"/v1/automations",
	"/v1/automations/",
	"/v1/automations/validate",
	"/v1/contact-properties",
	"/v1/contact-properties/",
	"/v1/contacts",
	"/v1/contacts/",
	"/v1/domains",
	"/v1/domains/",
	"/v1/emails",
	"/v1/emails/",
	"/v1/emails/analytics",
	"/v1/emails/batch",
	"/v1/emails/inbound",
	"/v1/event-definitions",
	"/v1/event-definitions/",
	"/v1/events",
	"/v1/posts",
	"/v1/posts/",
	"/v1/segments",
	"/v1/segments/",
	"/v1/senders",
	"/v1/senders/",
	"/v1/suppressions",
	"/v1/suppressions/export",
	"/v1/templates",
	"/v1/templates/",
	"/v1/templates/render",
	"/v1/topics",
	"/v1/topics/",
	"/v1/webhooks/endpoints",
}

var endpointLiteral = regexp.MustCompile(`"(/v1/[^"]*)"`)

// TestEndpointParityWithThePythonSDK reads this package's own source the same
// way the list above was extracted from Python's, so the two sets are compared
// on identical terms.
func TestEndpointParityWithThePythonSDK(t *testing.T) {
	reached := endpointsInSource(t)

	var missing []string
	for _, endpoint := range pythonEndpointPrefixes {
		if !reached[endpoint] {
			missing = append(missing, endpoint)
		}
	}
	if len(missing) > 0 {
		t.Errorf("this SDK reaches no endpoint for:\n  %s", strings.Join(missing, "\n  "))
	}

	if len(pythonEndpointPrefixes) != 35 {
		t.Errorf("the reference list has %d entries, want the 35 the Python SDK reaches", len(pythonEndpointPrefixes))
	}
}

// A path built by concatenation must still appear as a literal, or the parity
// check above silently passes over an endpoint nobody can reach. This reports
// what the SDK has that Python does not — informational, not a failure, since
// this SDK is allowed to grow first.
func TestEndpointsBeyondThePythonSDKAreReported(t *testing.T) {
	known := map[string]bool{}
	for _, endpoint := range pythonEndpointPrefixes {
		known[endpoint] = true
	}

	var extra []string
	for endpoint := range endpointsInSource(t) {
		if !known[endpoint] {
			extra = append(extra, endpoint)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Logf("endpoints this SDK reaches beyond the Python set: %s", strings.Join(extra, " "))
	}
}

// endpointsInSource collects every "/v1/…" string literal in the package's
// non-test source.
func endpointsInSource(t *testing.T) map[string]bool {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	found := map[string]bool{}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		scanned++
		for _, match := range endpointLiteral.FindAllStringSubmatch(string(source), -1) {
			found[match[1]] = true
		}
	}

	if scanned == 0 {
		t.Fatal("no source files scanned; the parity check would pass vacuously")
	}
	return found
}
