package mailtea

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
)

// AssetsService is the `assets` resource — a publication's image library. Reach
// it as client.Assets.
//
// An email or site image block needs an absolute URL, so this is how a picture
// that is not already in the library gets into one. Pointing an image at a host
// you do not control breaks the day that host moves the file.
//
// PNG, JPEG, GIF, WebP or SVG, 5 MB per image. The bytes are checked against
// the declared content_type, so a mislabelled file is rejected rather than
// stored.
type AssetsService struct {
	client *Client
}

// Upload puts an image in the library and returns it, including the `url` to
// use as an image block's src.
//
// Takes publication_id, content, content_type and filename. `content` may be
// raw []byte — base64-encoded for you — or a string that is already base64:
//
//	raw, _ := os.ReadFile("hero.png")
//	asset, err := client.Assets.Upload(ctx, mailtea.Params{
//	    "publication_id": "pub_123",
//	    "content":        raw,
//	    "content_type":   "image/png",
//	    "filename":       "hero.png",
//	})
//	asset.String("url")
func (s *AssetsService) Upload(ctx context.Context, params Params) (Object, error) {
	payload := params
	if raw, ok := params["content"].([]byte); ok {
		// Copy rather than mutate: the caller's map is theirs, and a surprise
		// in-place edit of a payload they may re-use is a bug that surfaces one
		// call later.
		payload = make(Params, len(params))
		for key, value := range params {
			payload[key] = value
		}
		payload["content"] = base64.StdEncoding.EncodeToString(raw)
	}
	return s.client.object(ctx, http.MethodPost, "/v1/assets", bodyOrNil(payload))
}

// List lists the library, newest first. Filters: publication_id (required),
// search (file name), limit (1-200, default 100).
func (s *AssetsService) List(ctx context.Context, params Params) (*List, error) {
	return s.client.list(ctx, http.MethodGet, "/v1/assets"+query(params), nil)
}

// Delete retires an asset.
//
// The stored file is KEPT and its URL keeps resolving, so images inside
// already-sent emails do not break. This hides the asset from the library — it
// does not remove it from any email, template or page referencing it.
func (s *AssetsService) Delete(ctx context.Context, id string, params Params) (Object, error) {
	return s.client.object(ctx, http.MethodDelete, "/v1/assets/"+url.PathEscape(id)+query(params), nil)
}
