package mailtea

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// A tiny stand-in for the Mailtea API, so this SDK's tests run with no
// credentials and no network. It records every request it receives, which is
// what the assertions read, and it checks the bearer token first — the same way
// the real API does, so a client that forgets the key fails its test rather
// than silently "sending".
//
// Its behaviour mirrors examples/.shared/node/mock-mailtea.mjs in the Mailtea
// monorepo, extended to every resource this SDK reaches.

const (
	mockEmailID   = "txemail_00000000000000000000000000000000"
	mockAPIKey    = "mt_pat_test_key"
	mockRequestID = "req_00000000000000000000000000000000"
)

type recordedRequest struct {
	Method string
	// Path is the decoded request path, which is what routing reads.
	Path string
	// EscapedPath is the path exactly as it travelled on the wire. The two
	// differ when an id had to be escaped, and only this one proves it was.
	EscapedPath   string
	Query         url.Values
	Authorization string
	UserAgent     string
	ContentType   string
	RawBody       string
	// Body is the decoded JSON object body, nil when the request sent none or
	// sent an array instead.
	Body map[string]interface{}
	// Array is the decoded JSON array body (the batch send sends one).
	Array []interface{}
}

// HasBody reports whether the request carried a body at all. "No body" and "{}"
// are different to this API, so the distinction has to be testable.
func (r recordedRequest) HasBody() bool { return r.RawBody != "" }

type mockResponse struct {
	status      int
	body        string
	contentType string
}

type mockMailtea struct {
	*httptest.Server

	mu       sync.Mutex
	requests []recordedRequest
	// override answers the next request instead of the default route table.
	override func(recordedRequest) *mockResponse
}

func startMockMailtea(t *testing.T) *mockMailtea {
	t.Helper()

	mock := &mockMailtea{}
	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)

		recorded := recordedRequest{
			Method:        r.Method,
			Path:          r.URL.Path,
			EscapedPath:   r.URL.EscapedPath(),
			Query:         r.URL.Query(),
			Authorization: r.Header.Get("Authorization"),
			UserAgent:     r.Header.Get("User-Agent"),
			ContentType:   r.Header.Get("Content-Type"),
			RawBody:       string(raw),
		}
		if len(raw) > 0 {
			var decoded interface{}
			if err := json.Unmarshal(raw, &decoded); err == nil {
				switch typed := decoded.(type) {
				case map[string]interface{}:
					recorded.Body = typed
				case []interface{}:
					recorded.Array = typed
				}
			}
		}

		mock.mu.Lock()
		mock.requests = append(mock.requests, recorded)
		override := mock.override
		mock.mu.Unlock()

		w.Header().Set("x-request-id", mockRequestID)

		// Auth first, exactly as the real API orders it.
		if !strings.HasPrefix(recorded.Authorization, "Bearer ") ||
			strings.TrimPrefix(recorded.Authorization, "Bearer ") == "" {
			writeMock(w, &mockResponse{status: http.StatusUnauthorized, body: `{"error":"Unauthorized"}`})
			return
		}

		if override != nil {
			writeMock(w, override(recorded))
			return
		}
		writeMock(w, routeMock(recorded))
	}))
	t.Cleanup(mock.Close)

	return mock
}

func writeMock(w http.ResponseWriter, response *mockResponse) {
	if response == nil {
		response = &mockResponse{status: http.StatusOK, body: "{}"}
	}
	contentType := response.contentType
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(response.status)
	_, _ = io.WriteString(w, response.body)
}

// collectionPaths are the routes that answer with a list envelope rather than a
// single object. Spelling them out beats guessing from segment counts:
// /v1/webhooks/endpoints is a collection and /v1/topics/top_1 is not, and both
// have the same shape.
var collectionPaths = map[string]bool{
	"/v1/emails":             true,
	"/v1/emails/inbound":     true,
	"/v1/contacts":           true,
	"/v1/segments":           true,
	"/v1/topics":             true,
	"/v1/posts":              true,
	"/v1/senders":            true,
	"/v1/assets":             true,
	"/v1/suppressions":       true,
	"/v1/templates":          true,
	"/v1/domains":            true,
	"/v1/webhooks/endpoints": true,
	"/v1/contact-properties": true,
	"/v1/api-keys":           true,
	"/v1/automations":        true,
	"/v1/events":             true,
	"/v1/event-definitions":  true,
}

var collectionSuffixes = []string{"/runs", "/versions", "/tracking-domains", "/attachments"}

func isCollection(path string) bool {
	if collectionPaths[path] {
		return true
	}
	for _, suffix := range collectionSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func lastSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	return parts[len(parts)-1]
}

const mockListBody = `{"object":"list","data":[{"id":"obj_1"}],"total":1,"limit":20,"offset":0,"has_more":false,"next_cursor":"cur_2"}`

// routeMock answers the routes whose exact shape a test asserts on, and falls
// back to a generic object/list for the rest.
func routeMock(r recordedRequest) *mockResponse {
	id := lastSegment(r.Path)
	ok := func(body string) *mockResponse { return &mockResponse{status: http.StatusOK, body: body} }

	switch {
	case r.Method == http.MethodPost && r.Path == "/v1/emails":
		return ok(`{"id":"` + mockEmailID + `"}`)

	case r.Method == http.MethodPost && r.Path == "/v1/emails/batch":
		items := make([]string, 0, len(r.Array))
		for range r.Array {
			items = append(items, `{"id":"`+mockEmailID+`"}`)
		}
		return ok(`{"data":[` + strings.Join(items, ",") + `]}`)

	case r.Method == http.MethodGet && r.Path == "/v1/emails/analytics":
		return ok(`{"object":"analytics","total":3,"sent":3,"delivered":2,"bounced":1,"opened":1,"clicked":0,"status_counts":{"sent":3},"rates":{"delivery_rate":0.67},"from_date":"2026-08-01T00:00:00.000Z","to_date":null}`)

	case r.Method == http.MethodGet && strings.HasPrefix(r.Path, "/v1/emails/") && !strings.Contains(strings.TrimPrefix(r.Path, "/v1/emails/"), "/"):
		return ok(`{"object":"email","id":"` + id + `","subject":"Mock email","last_event":"delivered","to":"reader@yourdomain.com","open_count":2,"click_count":1,"created_at":"2026-01-01T00:00:00.000Z","dropped_recipients":[{"address":"blocked@example.com","field":"to","reason":"suppressed"}],"attachments":[{"filename":"receipt.pdf","content_type":"application/pdf","size":12}]}`)

	case r.Method == http.MethodPatch && strings.HasPrefix(r.Path, "/v1/emails/"):
		return ok(`{"object":"email","id":"` + id + `"}`)

	// Cancel is POST /v1/emails/:id/cancel. There is no DELETE on emails — the
	// real API has never defined one.
	//
	// The reply is `{object, id}` and nothing else — no `last_event`. That is
	// exactly what apps/api/src/email-rest.ts returns and what
	// examples/.shared/node/mock-mailtea.mjs answers, and a mock that threw in
	// a confirming status would teach a caller to read a field that never
	// arrives. Re-read the email with Get to see it turn `canceled`.
	case r.Method == http.MethodPost && strings.HasSuffix(r.Path, "/cancel") && strings.HasPrefix(r.Path, "/v1/emails/"):
		return ok(`{"object":"email","id":"` + strings.Split(r.Path, "/")[3] + `"}`)

	case r.Method == http.MethodGet && r.Path == "/v1/suppressions/export":
		return &mockResponse{
			status:      http.StatusOK,
			contentType: "text/csv",
			body:        "email,reason,source,created_at\nblocked@example.com,bounce,api,2026-01-01T00:00:00.000Z\n",
		}

	case r.Method == http.MethodPost && r.Path == "/v1/posts" && r.Body != nil:
		return ok(`{"object":"post","id":"post_1","status":"draft"}`)

	case r.Method == http.MethodPost && strings.HasSuffix(r.Path, "/test") && strings.HasPrefix(r.Path, "/v1/posts/"):
		return ok(`{"sent_to":["you@example.com"],"failed_to":[]}`)

	case r.Method == http.MethodGet && isCollection(r.Path):
		return ok(mockListBody)

	case r.Method == http.MethodGet:
		return ok(`{"object":"object","id":"` + id + `"}`)

	case r.Method == http.MethodPost:
		return ok(`{"object":"object","id":"obj_1"}`)

	case r.Method == http.MethodPatch:
		return ok(`{"object":"object","id":"` + id + `","updated":true}`)

	case r.Method == http.MethodDelete:
		return ok(`{"object":"object","id":"` + id + `","deleted":true}`)
	}

	return &mockResponse{status: http.StatusNotFound, body: `{"error":"Not Found","path":"` + r.Path + `"}`}
}

func (m *mockMailtea) all() []recordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]recordedRequest, len(m.requests))
	copy(out, m.requests)
	return out
}

// last is the most recent request, which is what most assertions want.
func (m *mockMailtea) last(t *testing.T) recordedRequest {
	t.Helper()
	all := m.all()
	if len(all) == 0 {
		t.Fatal("the mock received no requests")
	}
	return all[len(all)-1]
}

func (m *mockMailtea) respondWith(response *mockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.override = func(recordedRequest) *mockResponse { return response }
}

// newTestClient builds a client pointed at the mock.
func newTestClient(t *testing.T, mock *mockMailtea) *Client {
	t.Helper()
	client, err := New(mockAPIKey, WithBaseURL(mock.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

// assertRoute checks the method, path and bearer token of a recorded request —
// the three things every resource test has to prove.
func assertRoute(t *testing.T, request recordedRequest, method, path string) {
	t.Helper()
	if request.Method != method || request.Path != path {
		t.Errorf("got %s %s, want %s %s", request.Method, request.Path, method, path)
	}
	if request.Authorization != "Bearer "+mockAPIKey {
		t.Errorf("Authorization = %q, want %q", request.Authorization, "Bearer "+mockAPIKey)
	}
}

func assertQuery(t *testing.T, request recordedRequest, key, want string) {
	t.Helper()
	if got := request.Query.Get(key); got != want {
		t.Errorf("query %q = %q, want %q", key, got, want)
	}
}

func assertBodyField(t *testing.T, request recordedRequest, key string, want interface{}) {
	t.Helper()
	if request.Body == nil {
		t.Fatalf("request to %s carried no JSON object body (raw: %q)", request.Path, request.RawBody)
	}
	got, present := request.Body[key]
	if !present {
		t.Errorf("body has no %q (raw: %s)", key, request.RawBody)
		return
	}
	if got != want {
		t.Errorf("body[%q] = %#v, want %#v", key, got, want)
	}
}
