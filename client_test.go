package mailtea

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewReadsTheKeyFromTheEnvironment(t *testing.T) {
	t.Setenv("MAILTEA_API_KEY", "mt_pat_from_env")

	client, err := New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.apiKey != "mt_pat_from_env" {
		t.Errorf("apiKey = %q, want the value from MAILTEA_API_KEY", client.apiKey)
	}

	// An explicit key beats the environment.
	explicit, err := New("mt_pat_explicit")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if explicit.apiKey != "mt_pat_explicit" {
		t.Errorf("apiKey = %q, want the explicit key", explicit.apiKey)
	}
}

// The misconfiguration has to be reported where it happened. Without this the
// only symptom is a 401 on the first send, which reads as "bad key" rather than
// "no key".
func TestNewWithoutAKeyIsATypedClientSideError(t *testing.T) {
	t.Setenv("MAILTEA_API_KEY", "")

	_, err := New("")
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want a *mailtea.Error", err)
	}
	if apiErr.Status != 0 {
		t.Errorf("status = %d, want 0 for a client-side fault", apiErr.Status)
	}
	if apiErr.Code != "missing_api_key" {
		t.Errorf("code = %q, want missing_api_key", apiErr.Code)
	}
}

func TestBaseURLResolution(t *testing.T) {
	t.Setenv("MAILTEA_API_KEY", "mt_pat_test")
	t.Setenv("MAILTEA_API_BASE_URL", "")

	// The branch every production caller takes, and the one nothing else
	// exercises because the tests always point at a mock.
	client, err := New("mt_pat_test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.BaseURL() != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", client.BaseURL(), DefaultBaseURL)
	}
	if DefaultBaseURL != "https://api.mailtea.app" {
		t.Errorf("DefaultBaseURL = %q, want https://api.mailtea.app", DefaultBaseURL)
	}

	for name, given := range map[string]string{
		"empty":            "",
		"whitespace":       "   ",
		"trailing slash":   DefaultBaseURL + "/",
		"trailing slashes": DefaultBaseURL + "///",
	} {
		t.Run("option "+name, func(t *testing.T) {
			client, err := New("mt_pat_test", WithBaseURL(given))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if client.BaseURL() != DefaultBaseURL {
				t.Errorf("baseURL = %q, want %q", client.BaseURL(), DefaultBaseURL)
			}
		})
	}

	// A self-hosted URL keeps its path prefix; only the trailing slash goes, so
	// paths concatenate to one slash rather than two.
	client, err = New("mt_pat_test", WithBaseURL("https://mail.example.internal/api/"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.BaseURL() != "https://mail.example.internal/api" {
		t.Errorf("baseURL = %q, want the trailing slash trimmed and the path kept", client.BaseURL())
	}
}

func TestBaseURLFromEnvironmentAndOptionPrecedence(t *testing.T) {
	t.Setenv("MAILTEA_API_KEY", "mt_pat_test")
	t.Setenv("MAILTEA_API_BASE_URL", "http://127.0.0.1:7787/")

	client, err := New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.BaseURL() != "http://127.0.0.1:7787" {
		t.Errorf("baseURL = %q, want the environment value with its slash trimmed", client.BaseURL())
	}

	client, err = New("", WithBaseURL("http://localhost:9999"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.BaseURL() != "http://localhost:9999" {
		t.Errorf("baseURL = %q, want the explicit option to beat the environment", client.BaseURL())
	}
}

func TestEveryRequestIdentifiesTheSDK(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	if _, err := client.Emails.Get(context.Background(), mockEmailID); err != nil {
		t.Fatalf("Get: %v", err)
	}

	request := mock.last(t)
	if want := "mailtea-go/" + Version; request.UserAgent != want {
		t.Errorf("User-Agent = %q, want %q", request.UserAgent, want)
	}
	// A GET carries no body, so it must not claim to.
	if request.ContentType != "" {
		t.Errorf("Content-Type = %q on a GET, want none", request.ContentType)
	}
}

func TestAPIErrorCarriesStatusMessageCodeDetailsAndRequestID(t *testing.T) {
	mock := startMockMailtea(t)
	mock.respondWith(&mockResponse{
		status: http.StatusUnprocessableEntity,
		body:   `{"error":"Domain is not verified","code":"domain_not_verified","details":[{"path":["from"],"message":"unverified"}]}`,
	})
	client := newTestClient(t, mock)

	_, err := client.Emails.Send(context.Background(), SendEmailRequest{
		From:    "Acme <hello@unverified.example.org>",
		To:      []string{"reader@yourdomain.com"},
		Subject: "Nope",
		Text:    "Nope",
	})

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want a *mailtea.Error", err)
	}
	if apiErr.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", apiErr.Status)
	}
	if apiErr.Message != "Domain is not verified" {
		t.Errorf("message = %q, want the API's own message", apiErr.Message)
	}
	if apiErr.Code != "domain_not_verified" {
		t.Errorf("code = %q, want domain_not_verified", apiErr.Code)
	}
	if apiErr.RequestID != mockRequestID {
		t.Errorf("requestID = %q, want the x-request-id header", apiErr.RequestID)
	}
	details, ok := apiErr.Details.([]interface{})
	if !ok || len(details) != 1 {
		t.Errorf("details = %#v, want the API's issue list", apiErr.Details)
	}
	if got := apiErr.Error(); !strings.Contains(got, "Domain is not verified") ||
		!strings.Contains(got, "status 422") ||
		!strings.Contains(got, "code domain_not_verified") ||
		!strings.Contains(got, mockRequestID) {
		t.Errorf("Error() = %q, want it to name the message, the status and the request id", got)
	}
}

// A gateway in front of Mailtea answers with HTML or nothing at all. The caller
// prints Message on failure, so it has to say something either way.
func TestAPIErrorMessageSurvivesANonJSONBody(t *testing.T) {
	mock := startMockMailtea(t)
	mock.respondWith(&mockResponse{status: http.StatusBadGateway, body: ""})
	client := newTestClient(t, mock)

	_, err := client.Emails.List(context.Background(), nil)

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want a *mailtea.Error", err)
	}
	if apiErr.Status != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", apiErr.Status)
	}
	if apiErr.Message != http.StatusText(http.StatusBadGateway) {
		t.Errorf("message = %q, want %q", apiErr.Message, http.StatusText(http.StatusBadGateway))
	}
}

func TestMissingBearerIsRejectedByTheAPI(t *testing.T) {
	mock := startMockMailtea(t)
	// New refuses to build a keyless client, so the only way to prove the
	// server-side half of the contract — the API rejects an empty bearer — is to
	// blank the key afterwards.
	client, err := New("not-a-real-key", WithBaseURL(mock.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	client.apiKey = ""

	_, err = client.Emails.List(context.Background(), nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want a *mailtea.Error", err)
	}
	if apiErr.Status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", apiErr.Status)
	}
}

func TestTransportFailureIsAClientSideError(t *testing.T) {
	client, err := New("mt_pat_test", WithHTTPClient(failingDoer{}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Emails.List(context.Background(), nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want a *mailtea.Error", err)
	}
	if apiErr.Status != 0 || apiErr.Code != "transport_error" {
		t.Errorf("status/code = %d/%q, want 0/transport_error", apiErr.Status, apiErr.Code)
	}
}

type failingDoer struct{}

func (failingDoer) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp: connection refused")
}

func TestContextCancellationPropagates(t *testing.T) {
	mock := startMockMailtea(t)
	client := newTestClient(t, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.Emails.List(ctx, nil); err == nil {
		t.Fatal("a cancelled context must fail the request")
	}
}
