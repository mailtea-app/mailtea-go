package mailtea

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Version must not lie.
//
// It is written twice — the constant in version.go and the newest heading in
// CHANGELOG.md — and only one of them ends up on a release: the mirror's
// v<version> tag is cut from the constant, and the GitHub release notes are cut
// from the changelog section that matches it. A bump that touches one and not
// the other ships a release with empty notes, or a client that introduces
// itself as the wrong version in every bug report filed against it.
//
// This is the third Mailtea package to carry that trap: mailtea-cli had a test
// and it caught a stale VERSION during a bump; mailtea-mcp had none and had
// been introducing itself as "0.2.0" for five releases.
func TestVersionMatchesTheChangelog(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(Version) {
		t.Fatalf("Version = %q, want a bare semver like 0.1.0 (the git tag is v%s)", Version, Version)
	}

	raw, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatalf("reading CHANGELOG.md: %v", err)
	}

	heading := regexp.MustCompile(`(?m)^## (\d+\.\d+\.\d+)`).FindStringSubmatch(string(raw))
	if heading == nil {
		t.Fatal("CHANGELOG.md has no `## <version>` heading; the release notes come from one")
	}
	if heading[1] != Version {
		t.Errorf("version.go says %s but the newest CHANGELOG.md heading says %s", Version, heading[1])
	}
}

// The User-Agent is how a support request is traced back to the client that
// made it, so it has to carry the real version rather than a placeholder.
func TestUserAgentCarriesTheVersion(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.APIKeys.List(nil); err != nil { //nolint:staticcheck // a nil context is the documented "background" case
		t.Fatalf("List: %v", err)
	}
	if got := mock.last(t).UserAgent; !strings.HasSuffix(got, "/"+Version) {
		t.Errorf("User-Agent = %q, want it to end with /%s", got, Version)
	}
}
