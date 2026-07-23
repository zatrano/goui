# 08. Toast Notifications

GoUI ships a small push-notification ("toast") system built directly on top of
the WebSocket transport described in earlier chapters. There is no separate
HTTP endpoint or polling involved — a toast is just another frame type on the
same connection a component already uses for rendering.

Module path used throughout this document: `github.com/zatrano/goui`.

## 1. The moving parts

| Piece | Location | Role |
|---|---|---|
| `core.BaseComponent.Toast` / `ToastT` | `core/component.go` | Component-side API to fire a toast for the *current* session. |
| `ws.PushMessage` | `ws/frame.go` | Wire payload: `{Kind, Text}`. |
| `ws.FrameTypePush` | `ws/frame.go` | Frame type (`"push"`) carrying a `PushMessage`. |
| `Session.EnqueuePush` | `ws/session.go` | Puts a push frame on the session's outbound channel. |
| `Hub.Push` | `ws/hub.go` | Sends a toast to *one* session, by session ID. |
| `Hub.Broadcast` | `ws/hub.go` | Sends a toast to *every* registered session. |
| `client/modules/toast.js` | client runtime | Renders the toast DOM and handles auto-dismiss. |

### 1.1 `core.BaseComponent`

Every component embeds `core.BaseComponent`, which holds an internal `pusher`
callback that the session wires up automatically when a component is
mounted or activated (see `Session.injectPusher` in `ws/session.go`). You never
set this yourself.

```go
// core/component.go (excerpt)

// Toast sends a push notification to the current session (no-op if no pusher).
func (b *BaseComponent) Toast(kind, text string) {
	if b.pusher != nil {
		b.pusher(kind, text)
	}
}

// ToastT translates key then sends a toast for the current session.
func (b *BaseComponent) ToastT(kind, key string, args ...any) {
	b.Toast(kind, b.T(key, args...))
}
```

Because `Toast` is a no-op when `pusher` is `nil`, it is always safe to call —
including from unit tests that construct a component directly without a
`Session`, or from a component's `Mount`/`HandleEvent` before it has been
attached to a live connection.

- **`Toast(kind, text string)`** — send a literal string.
- **`ToastT(kind, key string, args ...any)`** — resolve `key` through the
  component's injected `i18n.Translator` (same mechanism as `T()`), then send
  the translated string. Prefer `ToastT` in application code so toast copy
  lives in your locale files next to every other user-facing string.

### 1.2 Kinds

The `Kind` field is a free-form string, but the client and the default
stylesheet only give special treatment to four values:

```
success | error | warning | info
```

Anything else (including an empty string) is normalized to `info` by the
client (`normalizeKind` in `client/modules/toast.js`). The server does not
validate `Kind` — pick one of the four kinds above unless you also ship
matching CSS.

### 1.3 Timings

Toasts auto-dismiss on the client. The default lifetime is longer for errors
so users have more time to read them:

```js
// client/modules/toast.js
const DEFAULT_MS = 5000; // success / warning / info
const ERROR_MS = 8000;   // error
```

A close button (`×`) is always rendered so the user can dismiss a toast early
regardless of kind.

## 2. Sending a toast from a component

The common case: a component finishes handling an event and wants to confirm
success (or report a failure) to the user who triggered it.

```go
func (c *ContactForm) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch event {
	// ...
	case "save":
		if !forms.ValidateAll(&c.Name, &c.Email, &c.Country, &c.Message, &c.Subscribe) {
			c.MarkDirty()
			return nil
		}
		c.Submitted = true
		c.ToastT("success", "contact.submit_success")
		c.MarkDirty()
	}
	return nil
}
```

This is the exact pattern used by `examples/contact-form/main.go`. `ToastT`
looks up `"contact.submit_success"` in the component's locale (`c.Locale`,
set by the session from the WS `?locale=` query parameter) and pushes it as a
`success` toast to the same connection that sent the `save` event. The render
that follows (`session.sendRender`) is a completely separate frame — a toast
never replaces or blocks the normal diff/patch render cycle.

You can call `Toast`/`ToastT` from `Mount`, `HandleEvent`, or any helper
method the component calls into; it always targets **the session the
component belongs to**, whichever session that happens to be at the time.

## 3. Pushing from outside a component: `Hub`

Sometimes the notification has nothing to do with the component that is
currently rendering — a background job finished, another user triggered a
side effect, an admin wants to broadcast a maintenance notice. For that you
go through the `*ws.Hub` you already created when wiring up your adapter
(`Options.Server`).

### 3.1 Targeted: `Hub.Push`

```go
// Send a toast to one specific session by ID.
if err := wsHub.Push(sessionID, ws.PushMessage{
	Kind: "info",
	Text: "Your export is ready.",
}); err != nil {
	// ws.ErrSessionNotFound if the session is gone (disconnected past grace period).
	log.Printf("push failed: %v", err)
}
```

`Hub.Push` looks the session up by `sessionID` (the same ID the client
persists in `sessionStorage` and sends back on reconnect) and calls
`Session.EnqueuePush` on it. If the session isn't registered anymore, it
returns `ws.ErrSessionNotFound`.

### 3.2 Broadcast: `Hub.Broadcast`

```go
// Send a toast to every currently registered session.
wsHub.Broadcast(ws.PushMessage{
	Kind: "warning",
	Text: "Scheduled maintenance in 10 minutes.",
})
```

`Broadcast` snapshots the current session list under a read lock and enqueues
the push frame on each one. It never fails — sessions that are momentarily
disconnected (within their grace period) simply buffer the frame in their
outbound channel and receive it once a connection reattaches; sessions with a
full outbound buffer silently drop the frame (see the enqueue behavior below).

### 3.3 A real example: admin broadcast route

`examples/contact-form/main.go` exposes a plain HTTP endpoint that broadcasts
to every connected client — useful for a quick "announce to everyone online"
admin action without building any additional UI:

```go
wsHub := ws.NewHub()
server := ws.NewServer(wsHub, registry, tr)
gouifiber.Register(app, gouifiber.Options{Server: server})

app.Get("/admin/broadcast", func(c fiber.Ctx) error {
	text := c.Query("text")
	if text == "" {
		text = "Sunucudan duyuru"
	}
	kind := c.Query("kind")
	if kind == "" {
		kind = "info"
	}
	wsHub.Broadcast(ws.PushMessage{Kind: kind, Text: text})
	return c.JSON(fiber.Map{"ok": true, "kind": kind, "text": text})
})
```

Try it against the running example:

```
GET http://localhost:3001/admin/broadcast?text=Hello+everyone&kind=success
```

Every browser tab currently connected to the GoUI WebSocket endpoint pops a
toast immediately, with no page reload and no relation to any particular
component's render cycle. This is the pattern to copy for real admin tooling:
gate the route behind auth, then call `hub.Broadcast` (or `hub.Push` for a
single user) from wherever your business logic decides a notification is
warranted — a webhook handler, a cron job, another goroutine, etc.

## 4. Wiring the client

The client runtime (`client/goui.js`) already routes incoming `"push"` frames
to an `onPush` callback you provide when constructing `GoUIClient`. Wire that
callback to the toast module's `showToast` function:

```js
import { GoUIClient } from '/client/goui.js';
import { enhanceToast, showToast } from '/client/modules/toast.js';

enhanceToast(); // creates the .goui-toast-host container in <body>

const client = new GoUIClient('/goui/ws', 'contact', {
  locale: 'en',
  onPush: showToast,
  onError: (msg) => console.error('[goui]', msg),
});

client.connect();
```

`enhanceToast(root)` lazily creates a single `<div class="goui-toast-host">`
(with `aria-live="polite"`) the first time it's needed, so it's safe to call
once at startup even before any toast has fired. `showToast(payload)`:

1. Normalizes `payload.kind` to one of `success | error | warning | info`
   (default `info`).
2. Skips rendering entirely if `payload.text` is empty.
3. Creates a `<div class="goui-toast goui-toast-<kind>">` with a message span
   and a close button, and **prepends** it to the host (newest on top).
4. Sets `role="alert"` for error toasts and `role="status"` otherwise, so
   screen readers announce it appropriately.
5. Schedules removal after `ERROR_MS` (8000ms) for `error`, `DEFAULT_MS`
   (5000ms) for everything else — cancelable by clicking the close button.

## 5. Styling

Toast colors come from the same design tokens the rest of GoUI forms use
(see [12-theming-and-tailwind.md](12-theming-and-tailwind.md)). The relevant
rules live in `forms/style.css` under the `Toast / push notifications`
section:

```css
.goui-toast-host { position: fixed; top: 1rem; right: 1rem; z-index: 1000; /* ... */ }
.goui-toast { border: 1px solid var(--color-goui-border); background: var(--color-goui-surface); /* ... */ }
.goui-toast-success { border-color: ...var(--color-goui-success)...; background: ...var(--color-goui-success)...; }
.goui-toast-error   { border-color: ...var(--color-goui-error)...;   background: ...var(--color-goui-error)...;   }
.goui-toast-warning { border-color: ...var(--color-goui-warning)...; background: ...var(--color-goui-warning)...; }
.goui-toast-info    { border-color: ...var(--color-goui-info)...;    background: ...var(--color-goui-info)...;    }
```

Override the `--color-goui-*` custom properties in your own stylesheet (loaded
after `forms/style.css`) to rebrand toasts without touching any Go or JS code.

## 6. Delivery semantics you should know

- **Fire-and-forget on the server.** `EnqueuePush` does a non-blocking send
  on the session's buffered outbound channel (capacity 32). If the channel is
  full — an unusually backed-up client — the frame is silently dropped rather
  than blocking the caller. Toasts are for transient UX feedback, not a
  guaranteed delivery/audit log.
- **Survives brief disconnects.** If the target session is disconnected but
  still within its grace period (see
  [09-prefetch.md](09-prefetch.md) and
  [13-project-integration.md](13-project-integration.md) for grace period
  details), the push frame queues up and is delivered as soon as the
  WebSocket reattaches — the reader isn't "listening" while offline, but the
  channel buffer holds the frame.
- **`Hub.Push` targets a session, not a component.** A session can host
  several mounted/prefetched components at once; a toast is not scoped to
  any one of them. It always surfaces at the top-level `.goui-toast-host` for
  that browser tab.
- **No persistence.** Toasts are ephemeral by design. If you need users to
  see a notification they missed while offline, model that as application
  data (e.g. a notifications list rendered by a component on next Mount), not
  as a toast.
