# 04 — Sessions, Hub, and the WebSocket Protocol

This page documents the `ws` package (`github.com/zatrano/goui/ws`): the
`Session`/`Hub` lifecycle, the wire frame format, and how to wire it into any
HTTP stack through `ws.Server` and a framework adapter.

```go
import "github.com/zatrano/goui/ws"
```

## 1. Architecture at a glance

```
Browser (goui.js)
    │  event / prefetch / activate  (JSON frames over WebSocket)
    ▼
Session ──► Component.HandleEvent / Mount
    │
    ▼
Render HTML ──► Diff (old tree → patches) ──► Frame(render)
    │
    ▼
Hub (sessions map, grace-period reconnect, Broadcast)
```

1. The browser connects to `/goui/ws?component=<name>&locale=<locale>`.
2. The server looks up `<name>` in the `core.Registry`, creates a `Session`,
   mounts the component, and sends a `session` frame followed by the first
   `render` frame (a full `OpReplace` patch).
3. User interaction sends `event` frames; the server calls `HandleEvent`,
   re-renders, diffs old vs. new HTML, and streams back the minimal set of
   patches as a `render` frame.
4. Optionally, `prefetch` silently mounts a component ahead of navigation;
   `activate` promotes it into the visible set and sends its first render.
5. On disconnect, the session is **not** torn down immediately — it is kept
   alive for a grace period so a reconnect with the same session ID resumes
   exactly where it left off.

## 2. `Session` lifecycle

```go
func NewSession(conn *websocket.Conn, translator *i18n.Translator, locale string) *Session
```

A `Session` owns every `core.Component` instance created for one logical
browser tab, for its entire lifetime — including across reconnects. Key
fields (all otherwise private) exposed via methods:

| Method | Purpose |
|---|---|
| `SetRegistry(registry *core.Registry)` | Attaches the registry used to resolve names for `Prefetch`/`Activate` and (indirectly) the initial `?component=` connect. Called by `ws.Server.ServeConn`; you rarely call this yourself. |
| `MountComponent(id string, c core.Component) error` | Injects the translator/pusher/ID/Locale into `c` (via `SetTranslator`, `SetPusher`, and reflection into an embedded `BaseComponent`), then calls `c.Mount(ctx)` and stores it under `id`. Used for the component created from the initial `?component=` query parameter. |
| `Prefetch(name string) error` | Looks `name` up in the registry, mounts a fresh instance **without rendering it**, and stores it in a side map keyed by registry name (not yet a session-visible component ID). Duplicate prefetches of the same name are a no-op. When more than `ws.MaxPrefetch` (5) names are prefetched, the oldest is evicted and `Unmount`ed (simple LRU by insertion order). |
| `Activate(name string) (string, error)` | Promotes a prefetched component into the active set (or, if it was never prefetched, creates+mounts a fresh one), assigns it a new component ID, sends its first full render, and returns the new ID. |
| `Run(ctx context.Context)` | Starts the read loop and write loop as two goroutines and blocks until either the context is cancelled, the read loop exits (connection closed/error), or the write loop exits. On return, marks the session disconnected and closes the underlying connection — but does **not** unmount components. Called once per WebSocket upgrade, by `ws.Server.ServeConn` after your adapter hands it an upgraded connection. |
| `Reattach(conn *websocket.Conn) error` / `ReattachConn(conn wsConn) error` | Binds a new live connection to an existing (disconnected) session, clearing `disconnectedAt`. Returns `ErrSessionAlreadyActive` if the session already has a live connection (e.g. a stale duplicate reconnect attempt). |
| `SendSessionFrame()` | Enqueues a `session` frame carrying the session ID, so the browser can persist it (`sessionStorage`) for future reconnects. Sent once, right after a **fresh** connect (not on reconnect — the client already knows its ID then). |
| `SendInitialRenders()` | Enqueues a fresh full `render` frame for every currently-mounted **active** component. Called on every connect/reconnect so the browser's DOM is (re)synchronized with server state — this is what makes reconnect "just work" without any client-side state. |
| `Close() error` | Unmounts every active *and* prefetched component (`Unmount(ctx)` on each), clears all internal maps, closes the connection, and closes the outbound channel. Called by the `Hub`'s cleanup loop once a disconnected session's grace period has elapsed. |
| `IsDisconnected() bool` / `DisconnectedAt() time.Time` / `IsExpired(grace time.Duration) bool` | Introspection used by the `Hub`'s cleanup loop; `IsExpired` is `false` while connected (`disconnectedAt` is zero) and becomes `true` once `time.Since(disconnectedAt) > grace`. |
| `EnqueuePush(msg PushMessage)` | Enqueues a `push` frame. This is what `BaseComponent.Toast`/`ToastT` end up calling, via the pusher callback `ws.Session.injectPusher` wires into every mounted component. |

Internally, each `Session` has a buffered outbound channel (32 frames) drained
by the write loop; if the channel is full, `enqueue` silently drops the frame
rather than blocking (a slow/dead client should not stall the read loop).

## 3. `Hub` lifecycle

```go
type Hub struct { /* ... */ }

func NewHub() *Hub
func NewHubWithGracePeriod(grace time.Duration) *Hub

const DefaultGracePeriod = 60 * time.Second
```

The `Hub` is the process-wide registry of live `Session`s, plus a background
cleanup goroutine.

- **`NewHub()`** starts a hub with `DefaultGracePeriod` (defined in
  `ws/hub.go` as **`60 * time.Second`**) and an internal cleanup tick of
  10 seconds (`defaultCleanupInterval`, not currently configurable).
- **`NewHubWithGracePeriod(grace time.Duration)`** lets you override the
  grace period — mainly intended for tests that want a short grace period
  to assert cleanup behavior deterministically, but nothing stops you from
  tuning it in production if 60s doesn't fit your reconnect UX (e.g. a
  flaky mobile network might want a longer grace period; a low-memory
  multi-tenant deployment might want a shorter one).

```go
hub := ws.NewHub()                              // 60s grace period
hub := ws.NewHubWithGracePeriod(10 * time.Second) // custom grace period
```

Hub methods:

| Method | Purpose |
|---|---|
| `Register(s *Session)` | Adds a session to the hub's map, keyed by `s.ID`. Called once per fresh connect (not on reconnect — the session already exists). |
| `Unregister(sessionID string)` | Removes a session from the map without closing it. Rarely called directly; cleanup uses `delete` + `Close` together. |
| `Get(sessionID string) (*Session, bool)` | Looks up a session by ID — used by the WebSocket handler to find the session for a `?session=<id>` reconnect. |
| `Push(sessionID string, msg PushMessage) error` | Sends one push message to exactly one session by ID. Returns `ErrSessionNotFound` if the ID isn't registered. |
| `Broadcast(msg PushMessage)` | Sends the same push message to every currently registered session — this is what admin/notification endpoints use (see the `contact-form` example's `/admin/broadcast` route). |
| `Stop()` | Signals the cleanup goroutine to exit and blocks until it has. Call this on graceful application shutdown if you want the background goroutine to stop cleanly (not required for the process to exit, just tidy). |

Every 10 seconds, the cleanup loop scans all registered sessions; any
session whose `IsExpired(gracePeriod)` is `true` (disconnected longer than
the grace period) is removed from the map and has `Close()` called on it
(unmounting every component it still owned).

## 4. Reconnect semantics

- **Fresh connect:** `?component=<name>` (no `?session=`) → registry lookup,
  new `Session`, `MountComponent`, `hub.Register`, `SendSessionFrame`,
  `SendInitialRenders`, then `Run`.
- **Reconnect:** `?session=<id>` → `hub.Get(id)`; if found, `Reattach(conn)`
  rebinds the live connection (error `ErrSessionAlreadyActive` if another
  connection is already attached — e.g. two tabs racing on the same ID); no
  new `session` frame is sent (the client already has the ID), but
  `SendInitialRenders()` still runs so the DOM catches up to any server-side
  state changes that happened while disconnected.
- **Unknown session:** `?session=<id>` where `id` is not in the hub (already
  expired/cleaned up) → the server writes an `error` frame
  (`"session not found"`) and closes the connection. The client-side runtime
  (`goui.js`) specifically recognizes this message, clears its stored session
  ID from `sessionStorage`, and reconnects fresh with `?component=` instead.

## 5. Frame protocol reference

Every message on the WebSocket is one JSON object matching the `ws.Frame`
struct:

```go
type Frame struct {
    Type      string          `json:"type"`
    Component string          `json:"component,omitempty"`
    Event     string          `json:"event,omitempty"`
    Payload   json.RawMessage `json:"payload,omitempty"`
}
```

```go
const (
    FrameTypeEvent    = "event"
    FrameTypeRender   = "render"
    FrameTypePush     = "push"
    FrameTypeError    = "error"
    FrameTypeSession  = "session"
    FrameTypePrefetch = "prefetch"
    FrameTypeActivate = "activate"
)
```

| Type | Direction | Payload shape | When it is sent |
|---|---|---|---|
| `event` | client → server | `{"component": "<id>", "event": "<name>", "payload": {...}}` | Every `g-click`/`g-change`/`g-submit`/debounced-`g-input` interaction. Routed to `Session.handleEventFrame`, which looks up the component by `component` (its instance ID) and calls `component.HandleEvent(ctx, event, payload)`. |
| `render` | server → client | `[]diff.Patch` (JSON array of patch objects: `op`, `path`, plus op-specific fields like `html`, `text`, `attr`, `value`, `from_idx`, `to_idx`) | After `HandleEvent` succeeds without `core.ErrSkipRender` (incremental patch against the previous tree), on the very first render of a component (full `OpReplace`), and once per component from `SendInitialRenders` on every (re)connect. |
| `push` | server → client | `ws.PushMessage` — `{"kind": "success\|error\|warning\|info", "text": "..."}` | Whenever `BaseComponent.Toast`/`ToastT` is called (per-session) or `Hub.Broadcast`/`Hub.Push` is called (admin/system notifications). Not tied to any specific component — it's a session-wide notification. |
| `error` | server → client | `ws.ErrorPayload` — `{"message": "..."}` | Malformed inbound JSON, unknown component ID in an `event` frame, a `HandleEvent`/`Render`/diff error, a failed `Reattach`, or an unknown/expired `?session=` ID on connect. |
| `session` | server → client | `ws.SessionPayload` — `{"id": "<session-id>"}` | Exactly once, immediately after a **fresh** connect (never on reconnect). The client persists this ID (`sessionStorage`) and includes it as `?session=` on future reconnect attempts. |
| `prefetch` | client → server | `{"component": "<registry-name>"}` (note: `Component` here holds the *registry name*, not yet an instance ID) | Sent by the `prefetch.js` client module on hover (~100ms) or viewport entry over an element with `data-goui-prefetch="<name>"`. Triggers `Session.Prefetch(name)` — mounts silently, no render sent back. |
| `activate` | client → server | `{"component": "<registry-name>"}` | Sent by `prefetch.js` on click of an element with `data-goui-activate="<name>"`. Triggers `Session.Activate(name)`, which promotes the prefetched instance (or mounts fresh if it wasn't prefetched) and immediately sends its first `render` frame — with a freshly generated component instance ID. |

Notes:

- `prefetch`/`activate` frames carry the **registry name** in the
  `component` field (e.g. `"contact"`), whereas `event`/`render` frames
  carry a **runtime instance ID** (a random hex string generated per
  mounted component). Don't confuse the two — a registry name may be
  activated into a brand-new instance ID each time.
- `render` payloads are always an array of `diff.Patch`, even for the very
  first render of a component (a single-element array with
  `{"op": "replace", "path": [], "html": "...", "tag": "..."}`), so the
  client-side logic for "first render" and "incremental patch" is unified.
- Failures inside `HandleEvent` that are exactly `core.ErrSkipRender` (via
  `errors.Is`) produce **no** frame at all — not even an `error` frame. This
  is intentional for client-owned editors; see
  [`07-forms-tier2.md`](07-forms-tier2.md).

## 6. `ws.Server`, adapters, and a `main.go` skeleton

The core module exposes a framework-agnostic acceptor:

```go
type Server struct { /* Hub, Registry, Translator */ }

func NewServer(hub *Hub, registry *core.Registry, translator *i18n.Translator) *Server

// Path is the default WebSocket endpoint path.
const Path = "/goui/ws"

type ConnectParams struct {
    SessionID     string // ?session=
    ComponentName string // ?component=
    Locale        string // ?locale=
}

func (s *Server) ServeConn(ctx context.Context, conn Conn, p ConnectParams) error
```

Each adapter performs the HTTP WebSocket upgrade, reads the query parameters
above, and calls `ServeConn`. You never call `ServeConn` yourself unless you
are writing a custom adapter.

Register routes through the adapter for your stack (`Options.Server` is
required; `Options.Store` is optional — see [11-file-uploads.md](11-file-uploads.md)):

```go
server := ws.NewServer(hub, registry, tr)

// Fiber
import gouifiber "github.com/zatrano/goui/adapters/fiber"
gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})

// net/http ServeMux
import gouistdlib "github.com/zatrano/goui/adapters/stdlib"
gouistdlib.Register(mux, gouistdlib.Options{Server: server, Store: store})

// Chi (stdlib adapter Mount)
gouistdlib.Mount(chiRouter, gouistdlib.Options{Server: server})

// Gin
import gouigin "github.com/zatrano/goui/adapters/gin"
gouigin.Register(r, gouigin.Options{Server: server, Store: store})

// Echo
import gouiecho "github.com/zatrano/goui/adapters/echo"
gouiecho.Register(e, gouiecho.Options{Server: server, Store: store})
```

Full skeleton for a Fiber `main.go` that wires everything together:

```go
package main

import (
    "log"
    "path/filepath"

    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/static"

    gouifiber "github.com/zatrano/goui/adapters/fiber"
    "github.com/zatrano/goui/core"
    "github.com/zatrano/goui/i18n"
    "github.com/zatrano/goui/upload"
    "github.com/zatrano/goui/ws"
)

func main() {
    // 1. i18n
    tr := i18n.NewTranslator()
    _ = tr.LoadLocale("tr", filepath.Join("i18n", "locales", "tr.json"))
    _ = tr.LoadLocale("en", filepath.Join("i18n", "locales", "en.json"))

    // 2. Component registry
    registry := core.NewRegistry()
    if err := registry.Register("counter", func() core.Component { return &Counter{} }); err != nil {
        log.Fatal(err)
    }

    // 3. Hub — default 60s grace period, or tune it:
    hub := ws.NewHub()
    // hub := ws.NewHubWithGracePeriod(30 * time.Second)
    server := ws.NewServer(hub, registry, tr)

    // 4. Fiber app + static assets
    app := fiber.New()
    app.Use("/client", static.New("./client"))
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendFile("index.html")
    })

    // 5. (optional) file uploads — see 11-file-uploads.md
    store, err := upload.NewLocalStore("./.goui-uploads", "/goui/files", 8<<20)
    if err != nil {
        log.Fatal(err)
    }

    // 6. GoUI routes (WebSocket + optional upload)
    gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})

    // 7. (optional) server-initiated push, e.g. an admin broadcast endpoint
    app.Get("/admin/broadcast", func(c fiber.Ctx) error {
        hub.Broadcast(ws.PushMessage{Kind: "info", Text: c.Query("text", "Hello")})
        return c.JSON(fiber.Map{"ok": true})
    })

    log.Println("listening on http://localhost:3000")
    log.Fatal(app.Listen(":3000"))

    // On graceful shutdown elsewhere in your code: hub.Stop()
}
```

See [`01-getting-started.md`](01-getting-started.md) for the `Counter`
component itself, and [`05-forms-tier1.md`](05-forms-tier1.md) /
[`07-forms-tier2.md`](07-forms-tier2.md) for what to register in place of
`counter` in a real application.
