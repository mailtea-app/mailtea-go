// Package mailtea is the official Go SDK for Mailtea — a thin, typed wrapper
// over the REST API at https://docs.mailtea.app/docs/api-reference.
//
// It has no dependencies outside the standard library.
//
//	client, err := mailtea.New(os.Getenv("MAILTEA_API_KEY"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	sent, err := client.Emails.Send(ctx, mailtea.SendEmailRequest{
//	    From:    "you@yourdomain.com",
//	    To:      []string{"recipient@example.com"},
//	    Subject: "Hello from Mailtea",
//	    HTML:    "<p>Your first email.</p>",
//	})
//
// Every method takes a context.Context and returns (result, error). Errors from
// the API are *mailtea.Error, reachable with errors.As.
package mailtea

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultBaseURL is the hosted Mailtea API.
const DefaultBaseURL = "https://api.mailtea.app"

// defaultTimeout applies to the client this SDK builds for itself. net/http's
// own default has no timeout at all, and a send that hangs forever is worse
// than one that fails.
const defaultTimeout = 30 * time.Second

// HTTPDoer is the slice of *http.Client this SDK uses. Supply your own to add a
// proxy, a timeout, retries, or — in a test — to answer without a network:
//
//	client, _ := mailtea.New("mt_pat_test", mailtea.WithHTTPClient(fake))
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Option configures a Client at construction.
type Option func(*Client)

// WithBaseURL points the client at a different Mailtea — a self-hosted
// instance, or http://127.0.0.1:7787 in local dev. An empty string is ignored,
// so passing os.Getenv("MAILTEA_API_BASE_URL") is safe when the variable is
// unset. A trailing slash is trimmed; a path prefix is kept.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/"); trimmed != "" {
			c.baseURL = trimmed
		}
	}
}

// WithHTTPClient replaces the underlying HTTP client. A nil value is ignored.
func WithHTTPClient(doer HTTPDoer) Option {
	return func(c *Client) {
		if doer != nil {
			c.httpClient = doer
		}
	}
}

// Client talks to one Mailtea instance with one API key. It is safe for
// concurrent use by multiple goroutines.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient HTTPDoer

	Emails            *EmailsService
	Contacts          *ContactsService
	Segments          *SegmentsService
	Topics            *TopicsService
	Posts             *PostsService
	Senders           *SendersService
	Assets            *AssetsService
	Suppressions      *SuppressionsService
	Templates         *TemplatesService
	Domains           *DomainsService
	Webhooks          *WebhooksService
	ContactProperties *ContactPropertiesService
	APIKeys           *APIKeysService
	Automations       *AutomationsService
	AutomationRuns    *AutomationRunsService
	Events            *EventsService
	EventDefinitions  *EventDefinitionsService
}

// New builds a client.
//
// The API key is an mt_pat_… or mt_svc_… token. Pass it explicitly, or pass ""
// to read MAILTEA_API_KEY from the environment. With neither, New returns a
// *Error with Status 0 and Code "missing_api_key" — the misconfiguration is
// reported where it happened rather than as a 401 on the first send.
//
// The base URL defaults to DefaultBaseURL, overridden by MAILTEA_API_BASE_URL
// and then by WithBaseURL.
func New(apiKey string, opts ...Option) (*Client, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("MAILTEA_API_KEY"))
	}
	if key == "" {
		return nil, &Error{
			Message: "Missing Mailtea API key. Pass it to mailtea.New(apiKey) or set the MAILTEA_API_KEY environment variable.",
			Code:    "missing_api_key",
		}
	}

	client := &Client{
		apiKey:     key,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
	WithBaseURL(os.Getenv("MAILTEA_API_BASE_URL"))(client)
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}

	client.Emails = &EmailsService{client: client, Inbound: &InboundService{
		client:      client,
		Attachments: &InboundAttachmentsService{client: client},
	}}
	client.Contacts = &ContactsService{client: client}
	client.Segments = &SegmentsService{client: client}
	client.Topics = &TopicsService{client: client}
	client.Posts = &PostsService{client: client}
	client.Senders = &SendersService{client: client}
	client.Assets = &AssetsService{client: client}
	client.Suppressions = &SuppressionsService{client: client}
	client.Templates = &TemplatesService{client: client}
	client.Domains = &DomainsService{client: client, Tracking: &TrackingDomainsService{client: client}}
	client.Webhooks = &WebhooksService{client: client}
	client.ContactProperties = &ContactPropertiesService{client: client}
	client.APIKeys = &APIKeysService{client: client}
	client.Automations = &AutomationsService{client: client}
	client.AutomationRuns = &AutomationRunsService{client: client}
	client.Events = &EventsService{client: client}
	client.EventDefinitions = &EventDefinitionsService{client: client}

	return client, nil
}

// BaseURL reports the API this client talks to, after the environment and
// options have been applied.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// do performs one request and returns the raw response body. A non-2xx status
// becomes a *Error carrying everything the caller needs to act on it.
func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var payload io.Reader
	hasBody := body != nil
	if hasBody {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, &Error{Message: "encoding request body: " + err.Error(), Code: "invalid_request"}
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return nil, &Error{Message: "building request: " + err.Error(), Code: "invalid_request"}
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "mailtea-go/"+Version)
	if hasBody {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, &Error{Message: method + " " + path + ": " + err.Error(), Code: "transport_error"}
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &Error{
			Message:   "reading " + method + " " + path + ": " + err.Error(),
			Code:      "transport_error",
			RequestID: response.Header.Get("x-request-id"),
		}
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, newAPIError(response.StatusCode, response.Header.Get("x-request-id"), raw)
	}
	return raw, nil
}

// call runs a request and decodes the response into out. A 204 or an empty body
// leaves out untouched, which is what the delete endpoints return.
func (c *Client) call(ctx context.Context, method, path string, body, out interface{}) error {
	raw, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return &Error{
			Message: "decoding " + method + " " + path + ": " + err.Error(),
			Code:    "invalid_response",
			Body:    string(raw),
		}
	}
	return nil
}

// object runs a request and returns the response as a free-form Object.
func (c *Client) object(ctx context.Context, method, path string, body interface{}) (Object, error) {
	var out Object
	if err := c.call(ctx, method, path, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// list runs a request and decodes the standard list envelope.
func (c *Client) list(ctx context.Context, method, path string, body interface{}) (*List, error) {
	var out List
	if err := c.call(ctx, method, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// text runs a request and returns the body verbatim, for the endpoints that
// answer with something other than JSON (the suppressions CSV export).
func (c *Client) text(ctx context.Context, method, path string, body interface{}) (string, error) {
	raw, err := c.do(ctx, method, path, body)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// newAPIError turns a non-2xx response into a *Error. A body that is not the
// JSON the API promised still has to say something useful, so the status line
// is the fallback: "the send failed" and "the domain isn't verified yet" are
// different messages, and only one of them is actionable.
func newAPIError(status int, requestID string, raw []byte) *Error {
	apiErr := &Error{Status: status, RequestID: requestID, Body: string(raw)}

	var parsed struct {
		Error   string      `json:"error"`
		Code    string      `json:"code"`
		Details interface{} `json:"details"`
	}
	if err := json.Unmarshal(raw, &parsed); err == nil {
		apiErr.Message = parsed.Error
		apiErr.Code = parsed.Code
		apiErr.Details = parsed.Details
	}
	if apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(raw))
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(status)
	}
	return apiErr
}
