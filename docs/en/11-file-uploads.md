# 11. File Uploads

Binary file data never travels over the WebSocket in GoUI. Uploads are a
regular multipart HTTP `POST`, handled by a small `upload` package; only the
resulting *metadata* (an ID, name, URL, size, content type) flows back into
a component over the WS, via a normal event. This keeps the WS protocol
JSON-only and lets you swap the storage backend without touching any
component code.

Module path used throughout this document: `github.com/zatrano/goui`.

## 1. `upload.Storage`

```go
// upload/storage.go
type Storage interface {
	Save(originalName, contentType string, r io.Reader, size int64) (Meta, error)
	Open(id string) (io.ReadCloser, Meta, error)
	Delete(id string) error
}

type Meta struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
	StoredAt    time.Time `json:"-"`
}
```

Everything in GoUI that deals with uploaded files — the HTTP routes, the
`forms.DragDropUpload` component, the client `upload.js` module — talks to
files exclusively through this interface and through `Meta`. Swap the
implementation passed to your adapter's `Store` option (or to
`upload.NewHandler` / `upload.Mount`) and every other layer is
unaffected.

## 2. `LocalStore` — the built-in implementation

```go
// upload/storage.go
type LocalStore struct {
	Dir      string
	BaseURL  string // e.g. /goui/files
	MaxBytes int64

	mu    sync.RWMutex
	index map[string]Meta
}

func NewLocalStore(dir, baseURL string, maxBytes int64) (*LocalStore, error)
```

Defaults applied by `NewLocalStore` when you pass zero values:

| Field | Default when empty/zero |
|---|---|
| `BaseURL` | `/goui/files` |
| `MaxBytes` | `8 << 20` (8 MiB) |

`Dir` has no default — you must always supply a real directory; `NewLocalStore`
calls `os.MkdirAll(dir, 0o755)` so it doesn't need to exist beforehand.

```go
store, err := upload.NewLocalStore("./data/uploads", "/goui/files", 8<<20)
if err != nil {
	log.Fatal(err)
}
```

### 2.1 `Save`

- Generates a random 16-byte hex ID (`newID`), sanitizes the original
  filename (`sanitizeName` — strips directory components and `..`
  sequences via `filepath.Base` + a literal `".."` replace, so path
  traversal via the filename is not possible), and writes the file to
  `<Dir>/<id><ext>` where `<ext>` is taken from the *sanitized* name.
- Enforces `MaxBytes` **twice**: a fast-path rejection when the caller
  already knows `size` up front, and a hard stop while copying
  (`io.LimitReader(r, MaxBytes+1)` — if the copy reads more than `MaxBytes`,
  the partial file is removed and `"file too large"` is returned). This
  means `MaxBytes` is enforced even if the caller lies about (or doesn't
  know) `size` ahead of time.
- Falls back to `mime.TypeByExtension` when no `Content-Type` was supplied,
  and finally to `application/octet-stream`.
- Records the resulting `Meta` (with `URL: BaseURL + "/" + id`) in an
  in-memory index (`map[string]Meta` behind a `sync.RWMutex`). This index is
  **not persisted** — restarting the process loses the ability to serve
  previously uploaded files by ID via `Open`/`Delete`, even though the bytes
  are still on disk. Plan around this for anything beyond local development
  (see §5 for the production-shaped alternative).

### 2.2 `Open` / `Delete`

`Open(id)` looks the ID up in the in-memory index, then globs
`<Dir>/<id>.*` to find the file on disk (falling back to an extension-less
path). `Delete(id)` removes the index entry and globs+removes the matching
file(s). Both are safe to call concurrently with `Save` thanks to the
`sync.RWMutex`.

## 3. HTTP routes

The core module exposes a framework-agnostic `net/http` handler:

```go
// upload/handler.go
const (
    UploadPath  = "/goui/upload"
    FilesPrefix = "/goui/files"
)

func NewHandler(store Storage) *Handler   // POST upload + GET download
func Mount(mux *http.ServeMux, store Storage)
```

Wire uploads through your adapter's `Store` option (recommended), or mount the
handler directly on `net/http`:

```go
store, _ := upload.NewLocalStore(filepath.Join(root, "data", "uploads"), "/goui/files", 8<<20)

// Fiber (WebSocket + upload in one call)
gouifiber.Register(app, gouifiber.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})

// net/http ServeMux
gouistdlib.Register(mux, gouistdlib.Options{Server: server, Store: store})

// Or handler-only on any http.Handler mux:
upload.Mount(mux, store)
```

- **`POST /goui/upload`** expects a multipart form with a `file` field
  (`c.FormFile("file")`). On success it responds `200` with the `Meta` JSON
  body (`{"id":..., "name":..., "contentType":..., "size":..., "url":...}`).
  On a missing file or a `Storage.Save` error (including "file too large")
  it responds `400` with `{"error": "..."}`.
- **`GET /goui/files/:id`** streams the file back with the stored
  `Content-Type` and a `Content-Disposition: inline; filename="..."` header,
  or `404`/`500` on failure. Note the current handler buffers the whole file
  into memory (`io.ReadAll`) before writing the response — fine for the
  8 MiB-class defaults this package ships with, but something to revisit if
  you raise `MaxBytes` substantially or serve large media through this path.

Both routes are plain HTTP handlers with no GoUI-specific auth — put your own
middleware in front of `/goui/upload` and `/goui/files` if uploads/downloads
need to be gated.

## 4. The client-side flow end to end

`forms.DragDropUpload` (`forms/upload.go`) renders a drop zone plus a
hidden "carrier" button:

```go
type DragDropUpload struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Files      []UploadedRef
	Accept     string
	Multiple   bool
	ShowThumbs bool
	UploadURL  string // defaults to /goui/upload
	EventName  string
	OnChange   func(files []UploadedRef)
}
```

The full round trip:

1. The rendered markup carries `data-goui-upload`, `data-upload-url`
   (defaults to `/goui/upload`), `data-upload-event`, and `data-accept`/
   `data-multiple` attributes read by the client.
2. `client/modules/upload.js` listens for `dragover`/`drop`/`change` on any
   `[data-goui-upload]` zone, filters by `accept`, and for each file calls:

   ```js
   export async function postFile(url, file) {
     const fd = new FormData();
     fd.append('file', file, file.name);
     const res = await fetch(url, { method: 'POST', body: fd });
     const data = await res.json();
     if (!res.ok) throw new Error(data.error || res.statusText);
     return data; // the Meta JSON from the server
   }
   ```

3. On success, `notifyUploaded(zone, meta)` stuffs the returned `Meta` into
   `data-goui-*` attributes on the zone's hidden `.goui-upload-carrier`
   button and **synthetically clicks it**:

   ```js
   carrier.setAttribute('data-goui-value', meta.id || '');
   carrier.setAttribute('data-goui-id', meta.id || '');
   carrier.setAttribute('data-goui-name', meta.name || '');
   carrier.setAttribute('data-goui-url', meta.url || '');
   carrier.setAttribute('data-goui-size', String(meta.size || 0));
   carrier.setAttribute('data-goui-content-type', meta.contentType || '');
   carrier.click();
   ```

4. That click is a normal `g-click` event (`data-upload-event`, which
   defaults to `<name>.uploaded`) delegated by `goui.js`, which reads those
   same `data-goui-*` attributes into the event payload
   (`collectPayload` recognizes `data-goui-id`, `-name`, `-url`, `-size`,
   `-content-type`, `-value`, among others) and sends it as a regular WS
   event.
5. `DragDropUpload.HandleEvent` receives the `"uploaded"` action, builds an
   `UploadedRef` from the payload, appends/replaces `d.Files`, calls
   `OnChange` if set, and `MarkDirty()`s — a normal render follows, showing
   the new file (and thumbnail, if `ShowThumbs` and it's an image).

In other words: **the binary upload is plain HTTP, completely outside the
WS render loop; only the resulting metadata re-enters GoUI**, as an ordinary
event payload, indistinguishable in kind from a checkbox toggle or a text
change. `forms.NewImageUpload(name, event)` is a convenience preset
(`Accept: "image/*"`, `ShowThumbs: true`) for the common avatar/gallery case.

Removal follows the same shape: the "×" button on each listed file is a
plain `g-click` (`data-goui-value="<id>"`) wired to the `"remove"` action —
no HTTP call is made to actually delete the backing file on remove; add that
yourself in `HandleEvent`'s `"remove"` branch (call `store.Delete(id)`) if
removing from the list should also delete the stored object.

## 5. Writing your own `Storage` (S3/MinIO-shaped skeleton)

Because every layer above only depends on the `Storage` interface, adding a
cloud backend is a matter of implementing three methods. Below is a
**signature-only skeleton** — fill in the SDK calls for whichever object
store you use (AWS S3, MinIO, Cloudflare R2, GCS via an S3-compatible
endpoint, etc.):

```go
package myupload

import (
	"context"
	"io"
	"time"

	"github.com/zatrano/goui/upload"
)

// S3Store implements upload.Storage against an S3-compatible bucket.
type S3Store struct {
	Client   S3API  // whatever minimal client interface you need (put/get/delete/head)
	Bucket   string
	BaseURL  string // e.g. "https://cdn.example.com" or a presigned-URL base
	MaxBytes int64
}

// S3API is the minimal surface S3Store needs — narrow this to your SDK.
type S3API interface {
	PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	HeadObject(ctx context.Context, bucket, key string) (contentType string, size int64, err error)
	DeleteObject(ctx context.Context, bucket, key string) error
}

func NewS3Store(client S3API, bucket, baseURL string, maxBytes int64) *S3Store {
	// apply the same defaulting convention as LocalStore: baseURL "" -> your CDN/base,
	// maxBytes <= 0 -> 8<<20, etc.
	return &S3Store{Client: client, Bucket: bucket, BaseURL: baseURL, MaxBytes: maxBytes}
}

func (s *S3Store) Save(originalName, contentType string, r io.Reader, size int64) (upload.Meta, error) {
	// 1. size/MaxBytes guard (mirror LocalStore's LimitReader trick if size is unreliable)
	// 2. generate a key, e.g. time-bucketed prefix + random id + sanitized extension
	// 3. s.Client.PutObject(ctx, s.Bucket, key, r, size, contentType)
	// 4. persist {key -> original name / content type / size} somewhere durable —
	//    unlike LocalStore's in-memory map, this MUST survive a restart in production;
	//    use your application's own database table, not a package-level map.
	// 5. return upload.Meta{ID: key, Name: ..., ContentType: ..., Size: ..., URL: s.BaseURL + "/" + key, StoredAt: time.Now().UTC()}
	panic("not implemented")
}

func (s *S3Store) Open(id string) (io.ReadCloser, upload.Meta, error) {
	// 1. look up stored metadata for id from your durable store
	// 2. s.Client.GetObject(ctx, s.Bucket, id) (or redirect callers to a presigned URL
	//    instead of proxying bytes through your app — often the better production choice)
	panic("not implemented")
}

func (s *S3Store) Delete(id string) error {
	// 1. remove durable metadata row
	// 2. s.Client.DeleteObject(ctx, s.Bucket, id)
	panic("not implemented")
}
```

Then wire it exactly like `LocalStore`:

```go
store := myupload.NewS3Store(s3Client, "my-bucket", "https://cdn.example.com", 8<<20)
gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})
// GET /goui/files/:id can also just 302-redirect to a presigned URL in a custom Storage
```

Nothing in `forms.DragDropUpload`, the WS layer, or the client JS needs to
change — they only ever see `Meta` (`id`, `name`, `contentType`, `size`,
`url`) and the two HTTP routes' JSON contract.

## 6. Practical notes

- **`MaxBytes` is a server-side hard limit, not just a UI hint.** Set
  `accept`/`Multiple` on `DragDropUpload` to shape the UX, but always also
  configure `MaxBytes` to the real limit you want enforced, since a
  handcrafted request can bypass any client-side check.
- **`/goui/upload` and `/goui/files/:id` are unauthenticated by default.**
  Apply your own auth middleware in front of them if uploads/downloads are
  not meant to be public.
- **The in-memory `LocalStore.index` does not survive a restart.** It's
  intended for local development and the bundled examples. For anything
  persistent, either write your own `Storage` backed by a real database +
  disk/object storage, or at minimum replace the in-memory index with one
  that's rebuilt from disk (or backed by a real table) on startup.
- Deleting a file from a `DragDropUpload` list is a UI-only operation unless
  you add a `store.Delete(id)` call yourself in the `"remove"` branch of
  `HandleEvent` — see §4.
