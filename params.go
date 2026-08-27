package mailtea

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Params is a free-form wire-format payload: snake_case keys exactly as the
// REST API names them ("reply_to", "publication_id", "scheduled_at").
//
// The methods this SDK types explicitly — emails.Send/Batch/Update,
// contacts.Create/Update, posts.Create/Send/SendTest, topics.Create — take a
// request struct instead. Everything else takes Params, so a field added to the
// API is reachable the day it ships rather than the day this SDK is re-released.
type Params map[string]interface{}

// Object is a decoded JSON response. It is a plain map, so an unfamiliar or
// brand-new field is still readable, with typed accessors for the common reads
// and Decode for pulling the whole thing into a struct of your own.
type Object map[string]interface{}

// List is the API's standard list envelope. Offset-paginated endpoints (emails,
// posts) fill Total/Limit/Offset/HasMore; cursor-paginated ones (contacts,
// senders, templates, automations, events, …) fill NextCursor and leave Total
// at zero. Data holds the rows either way.
type List struct {
	Object     string   `json:"object"`
	Data       []Object `json:"data"`
	Total      int      `json:"total"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
	HasMore    bool     `json:"has_more"`
	NextCursor string   `json:"next_cursor"`
}

// String returns a string field, or "" when the key is absent or not a string.
func (o Object) String(key string) string {
	value, _ := o[key].(string)
	return value
}

// Bool returns a boolean field, or false when the key is absent or not a bool.
func (o Object) Bool(key string) bool {
	value, _ := o[key].(bool)
	return value
}

// Float returns a numeric field. JSON numbers decode to float64, so this is the
// lossless read; Int rounds it for counters and ids.
func (o Object) Float(key string) float64 {
	switch value := o[key].(type) {
	case float64:
		return value
	case json.Number:
		parsed, err := value.Float64()
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

// Int returns a numeric field truncated to an int, or 0 when absent.
func (o Object) Int(key string) int {
	return int(o.Float(key))
}

// Object returns a nested object field, or nil when absent or not an object.
func (o Object) Object(key string) Object {
	switch value := o[key].(type) {
	case Object:
		return value
	case map[string]interface{}:
		return Object(value)
	default:
		return nil
	}
}

// List returns a nested array of objects, skipping any element that is not one.
func (o Object) List(key string) []Object {
	raw, ok := o[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]Object, 0, len(raw))
	for _, item := range raw {
		if nested, ok := item.(map[string]interface{}); ok {
			out = append(out, Object(nested))
		}
	}
	return out
}

// Decode re-encodes the object and unmarshals it into v, so a caller who wants
// a struct does not have to reach through the map:
//
//	var domain struct {
//	    ID     string `json:"id"`
//	    Status string `json:"status"`
//	}
//	err := created.Decode(&domain)
func (o Object) Decode(v interface{}) error {
	encoded, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, v)
}

// query renders Params as a query string, "?a=1&b=2", or "" when there is
// nothing to send. Nil values are dropped, which is how an optional filter is
// omitted rather than sent empty.
//
// url.Values.Encode sorts by key, which matters more here than it looks: Go map
// iteration is randomised, and a URL that changes shape between identical calls
// defeats HTTP caches, log grouping, and any test that asserts on the path.
func query(params Params) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range params {
		if value == nil {
			continue
		}
		values.Set(key, queryValue(value))
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}

// queryValue renders one filter value the way the API reads it.
//
// Booleans matter here: several endpoints (automation runs' `is_test`, for one)
// match the literal strings "true"/"false" and 400 on anything else. A slice is
// joined with commas, which is how the multi-value filters — `status` on
// automation runs — are spelled on the wire.
func queryValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case []string:
		return strings.Join(typed, ",")
	case []interface{}:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, queryValue(item))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprintf("%v", typed)
	}
}

// publicationQuery renders the ?publication_id= filter that several PATCH
// endpoints read from the query string. An empty id sends nothing rather than
// `publication_id=`, which the server would read as a publication named "".
func publicationQuery(id string) string {
	if id == "" {
		return ""
	}
	return query(Params{"publication_id": id})
}

// withPublicationID pulls `publication_id` out of a payload and returns it as a
// one-key query payload. Several PATCH endpoints read the publication from the
// query string while taking the rest of the update in the body.
func withPublicationID(params Params) Params {
	if params == nil {
		return nil
	}
	return Params{"publication_id": params["publication_id"]}
}

// withoutPublicationID copies params minus `publication_id`, for the endpoints
// that read it from the query and reject it in the body.
//
// The result is never nil, and it is sent even when empty. Those endpoints read
// the body with `json().catch(() => null)` and hand the result to a schema, so
// "no body" arrives as `null` and comes back 400 "Validation failed" — while
// `{}` is a valid no-op update. An empty update is a strange call to make, but
// it should not fail for a reason that has nothing to do with what was asked.
func withoutPublicationID(params Params) Params {
	out := make(Params, len(params))
	for key, value := range params {
		if key == "publication_id" {
			continue
		}
		out[key] = value
	}
	return out
}

// bodyOrNil sends no body at all for an empty payload. `{}` and "no body" are
// not the same to the API: some endpoints have a schema that rejects an empty
// object, and some read a per-verb default only when the field is absent.
func bodyOrNil(params Params) interface{} {
	if len(params) == 0 {
		return nil
	}
	return params
}

// marshalWithExtra encodes a typed request struct and merges the caller's Extra
// keys over the result, so a field this SDK version does not name yet can still
// be sent. Callers pass an alias type here, never the original, or MarshalJSON
// would call itself forever.
func marshalWithExtra(value interface{}, extra Params) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(extra) == 0 {
		return encoded, nil
	}

	merged := map[string]interface{}{}
	if err := json.Unmarshal(encoded, &merged); err != nil {
		return nil, err
	}
	for key, item := range extra {
		merged[key] = item
	}
	return json.Marshal(merged)
}
