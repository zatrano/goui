# 02 — Components, BaseComponent, Registry, Templates

This page documents the `core` package (`github.com/zatrano/goui/core`) in
full: the `Component` contract, every `BaseComponent` field and method, the
`Registry`, and the HTML template cache.

```go
import "github.com/zatrano/goui/core"
```

## 1. The `Component` interface

```go
type Component interface {
    Mount(ctx context.Context) error
    Render() (string, error)
    HandleEvent(ctx context.Context, event string, payload map[string]any) error
    Unmount(ctx context.Context) error
}
```

Every GoUI view — from a two-line counter to a full form — implements these
four methods. `core.BaseComponent` (below) provides shared state and helpers
but deliberately does **not** implement `Component` itself: you always write
your own `Render`, `HandleEvent`, and lifecycle methods (or delegate to a
Tier 1/Tier 2 field's own implementation).

### `Mount(ctx context.Context) error`

- **When it is called:** exactly once, right after the component is
  instantiated by the `Registry` (or constructed manually) and before the
  first render. `ws.Session.MountComponent`, `Session.Prefetch`, and
  `Session.Activate` (fresh path) all call `Mount` as part of preparing a
  component for a session.
- **Do:** initialize any state that depends on runtime context — e.g.
  `TreeSelect.Mount` lazily allocates its `Expanded` map,
  `Cascader.Mount` seeds `Levels[0]` from `Items`,
  `MarkdownEditor.Mount` renders the initial preview HTML.
- **Don't:** perform the first render here — rendering is a separate step
  driven by the session (`sendFullRender`), and most Tier 1 fields simply
  `return nil`.
- Returning a non-nil error aborts session setup (the WebSocket handler
  writes an `error` frame and does not proceed).

### `Render() (string, error)`

- **When it is called:** after `Mount` for the very first render, and again
  after every `HandleEvent` call that does not return `core.ErrSkipRender`.
  The session diffs the previous rendered tree against the new one and sends
  only the patches that changed (see [`04-sessions-and-websocket.md`](04-sessions-and-websocket.md)).
- **Do:** return a single well-formed HTML fragment (one root element is
  strongly recommended — the session wraps multi-root output in an extra
  `<div>` so `data-goui-component` has somewhere to live). Call
  `ResetDirty()` once you have produced fresh output, if you track dirtiness.
  Use `core.RenderTemplate` (or hand-built strings via `forms.Attrs`) and
  route all translatable text through `T`.
- **Don't:** mutate component state inside `Render` — state changes belong in
  `HandleEvent`/`Mount`; `Render` should be a pure projection of the current
  fields.
- `Render` **must be safe to call more than once** with unchanged state —
  the session calls it once per event and once again on reconnect
  (`SendInitialRenders`).

### `HandleEvent(ctx context.Context, event string, payload map[string]any) error`

- **When it is called:** once per inbound `event` frame from the browser
  (`g-click`, `g-change`, `g-input` after debounce, `g-submit`). `event` is
  the string from the `z-*` attribute; `payload` is whatever
  `collectPayload`/`collectFormPayload` gathered client-side (typically
  `{"value": "..."}`, `{"checked": true, "value": "..."}`, or `{"fields": {...}}`
  for form submits).
- **Do:** mutate your component's fields based on `event`/`payload`, call
  `MarkDirty()` if you track dirtiness manually, and return
  `core.ErrSkipRender` for events whose UI is client-owned (rich text,
  code editors — see [`07-forms-tier2.md`](07-forms-tier2.md)).
- **Don't:** call `Render()` yourself — the session does that immediately
  after `HandleEvent` returns (unless you returned `ErrSkipRender`).
- Returning any other non-nil error causes the session to send an `error`
  frame back to the client instead of a `render` frame.

### `Unmount(ctx context.Context) error`

- **When it is called:** when the session closes (`Session.Close`, e.g. tab
  closed and the grace period elapsed, or the server shuts down the hub) —
  for **every** component still tracked, both active and prefetched-but-not-
  activated. It is also called immediately on a prefetched component that
  loses a race to a duplicate prefetch eviction.
- **Do:** release resources you opened in `Mount` (file handles, timers,
  subscriptions). Most Tier 1/Tier 2 fields have nothing to release and
  `return nil`.
- **Don't:** assume `Unmount` runs on every page navigation — it only runs
  when the *session* is torn down, not on every re-render.

## 2. `BaseComponent`

```go
type BaseComponent struct {
    ID       string
    Children map[string]Component
    // dirty bool // private

    Locale     string
    // translator, pusher are private
}
```

Embed it by value as the first field of your component struct:

```go
type MyComponent struct {
    core.BaseComponent
    // ... your fields
}
```

### Fields

| Field | Type | Purpose |
|---|---|---|
| `ID` | `string` | The session-assigned component instance ID. Set automatically by `ws.Session` via reflection when a component is mounted/activated (looks for a field named `BaseComponent` and sets its `ID`/`Locale` sub-fields) — you normally never set this yourself. |
| `Children` | `map[string]Component` | Optional slot for composing sub-components by name. The core package does not populate or consume this automatically; it's provided as a convention for components that manage nested components manually. |
| `Locale` | `string` | The active locale for this component instance (e.g. `"tr"`, `"en"`), set from the WebSocket `?locale=` query parameter when the session mounts/activates the component. Read by `T`/`ToastT`. |

### Methods

| Method | Signature | Behavior |
|---|---|---|
| `MarkDirty` | `func (b *BaseComponent) MarkDirty()` | Sets the internal dirty flag to `true`. Call this from `HandleEvent` whenever a change should trigger a re-render. |
| `IsDirty` | `func (b *BaseComponent) IsDirty() bool` | Reports the current dirty flag. Useful if your own `Render`/dispatch logic wants to skip work when nothing changed — the `ws.Session` itself always calls `Render` after a successful `HandleEvent` regardless of this flag, so `IsDirty`/`ResetDirty` are a convention for your own code, not enforced by the session. |
| `ResetDirty` | `func (b *BaseComponent) ResetDirty()` | Clears the dirty flag. Call at the end of `Render` once you've produced up-to-date HTML (see the `Counter` example in [`01-getting-started.md`](01-getting-started.md)). |
| `SetTranslator` | `func (b *BaseComponent) SetTranslator(t *i18n.Translator)` | Injects the shared `*i18n.Translator`. Called automatically by `ws.Session` (via an `interface{ SetTranslator(*i18n.Translator) }` type assertion) when a component is mounted or activated. You also call it manually for sub-fields you construct yourself (see the `ContactForm` example in [`01-getting-started.md`](01-getting-started.md) and [`06-validation.md`](06-validation.md)) since the session only reflects into the top-level component's own `BaseComponent`. |
| `SetPusher` | `func (b *BaseComponent) SetPusher(fn func(kind, text string))` | Injects a callback used by `Toast`/`ToastT`. `ws.Session.injectPusher` wires this automatically to `Session.EnqueuePush`, so pushes end up as `push` frames on the wire. |
| `Toast` | `func (b *BaseComponent) Toast(kind, text string)` | Sends a push notification (`kind` + already-translated `text`) through the injected pusher. A no-op (does not panic) if no pusher was injected — safe to call in tests or standalone components. |
| `ToastT` | `func (b *BaseComponent) ToastT(kind, key string, args ...any)` | Translates `key` via `T` first, then calls `Toast` with the result. Use this for user-facing notifications so they respect `Locale`. |
| `T` | `func (b *BaseComponent) T(key string, args ...any) string` | Translates `key` for `b.Locale` using the injected translator. If `Locale` is empty, falls back to `i18n.BaseLocale` (`"tr"`). If no translator was injected, returns the raw key wrapped as `"[[" + key + "]]"` — this makes missing wiring obvious in the rendered HTML rather than silently blank. See [`03-i18n.md`](03-i18n.md) for the full lookup/fallback algorithm. |

### `T` and `ToastT` in practice

```go
func (c *MyComponent) HandleEvent(_ context.Context, event string, _ map[string]any) error {
    if event == "save" {
        c.ToastT("success", "contact.submit_success") // translated + pushed
    }
    return nil
}

func (c *MyComponent) Render() (string, error) {
    label := c.T("form.submit") // "Gönder" (tr) or "Submit" (en)
    return "<button>" + html.EscapeString(label) + "</button>", nil
}
```

## 3. `Registry`

```go
type Registry struct { /* ... */ }

func NewRegistry() *Registry
func (r *Registry) Register(name string, factory func() Component) error
func (r *Registry) Create(name string) (Component, error)
```

The registry maps a **string name** (the value passed in
`?component=name` on the WebSocket URL, or to `Session.Prefetch`/`Activate`)
to a **factory function** that returns a fresh `Component` instance. It is
safe for concurrent use (guarded by an internal `sync.RWMutex`).

### `Register`

```go
registry := core.NewRegistry()
err := registry.Register("counter", func() core.Component { return &Counter{} })
```

- Registers `name` → `factory`.
- **Error:** returns `core.ErrComponentAlreadyRegistered` if `name` is
  already registered. Registration is typically done once at startup, so
  most code simply does `if err := registry.Register(...); err != nil { log.Fatal(err) }`.

### `Create`

```go
c, err := registry.Create("counter") // c is a fresh *Counter
```

- Looks up `name` and calls its factory, returning a brand-new `Component`
  instance every time (the factory closure is responsible for allocating a
  new struct — registries never share instances between sessions).
- **Error:** returns `core.ErrComponentNotRegistered` if `name` was never
  registered. This is the error you see surfacing as an `error` frame when a
  client connects with `?component=typo`, or calls `Prefetch`/`Activate` with
  an unknown name.

Both sentinel errors are defined in `core/errors.go`:

```go
var (
    ErrComponentNotRegistered     = errors.New("component not registered")
    ErrComponentAlreadyRegistered = errors.New("component already registered")
)
```

## 4. Template cache: `core.RenderTemplate`

```go
func RenderTemplate(tmplStr string, data any) (string, error)
```

`RenderTemplate` wraps Go's `html/template`, but **caches the parsed
template** keyed by an FNV-64a hash of the template *string* itself (not the
component or file). The first call with a given template string pays the
parse cost; every subsequent call with the identical string — from any
component, any session — reuses the cached `*template.Template` and only
pays for `Execute`. This is why it's idiomatic to write templates as string
literals directly inside `Render()`: the string is identical across calls,
so the cache hits every time after the first.

```go
func (c *Counter) Render() (string, error) {
    html, err := core.RenderTemplate(`<span>{{.Count}}</span>`, c)
    if err != nil {
        return "", err
    }
    c.ResetDirty()
    return html, nil
}
```

### The `{{call .T "key"}}` rule

Because `RenderTemplate` uses `html/template`, you cannot call a method
directly on the piped value inside conditionals or when passing the receiver
explicitly — instead you pass the `T` **function value** in the data map (or
rely on it being a field/method reachable via `.`) and invoke it with
`{{call .T "key"}}`. Two supported forms:

```go
// 1. Pass the whole component (T is a method value on BaseComponent, so
//    {{.T "key"}} works too, but {{call .T "key" .}} is the safe general form
//    when you need to pass placeholder data alongside the key):
core.RenderTemplate(`<p>{{call .T "welcome_message" .}}</p>`, c)
// requires the template's top-level data to have both a .T method and the
// fields referenced by the translation string (e.g. .Name for "{{.Name}}").

// 2. Pass a map with T explicitly, when the component itself isn't the
//    convenient top-level data value:
data := map[string]any{"Name": "GoUI", "T": bc.T}
core.RenderTemplate(`<p>{{call .T "welcome_message" .}}</p>`, data)

// Without placeholders you can omit the trailing "." argument:
core.RenderTemplate(`<button>{{call .T "form.submit"}}</button>`, map[string]any{"T": bc.T})
```

Rules of thumb:

- `.T` must resolve to a function of shape `func(key string, args ...any) string`
  — `BaseComponent.T` matches this exactly, so embedding `core.BaseComponent`
  and passing `c` (or `map[string]any{"T": c.T, ...}`) as the template data
  is enough.
- Use `{{call .T "key" .}}` (passing the whole data value as the single
  `args` element) when the translation string contains Go template
  placeholders like `{{.Name}}` — the translator re-parses the translated
  string as a template and executes it against that one argument.
- Use `{{call .T "key"}}` (no trailing argument) for plain strings with no
  placeholders.
- `html/template` auto-escapes the translated string as HTML, so you do not
  need to call `html.EscapeString` yourself when going through
  `RenderTemplate` — but you **do** need to escape manually when
  hand-building HTML with string concatenation (as most Tier 1/Tier 2 fields
  in the `forms` package do, via the standard library's `html.EscapeString`).

See [`03-i18n.md`](03-i18n.md) for how `Translate`/`T` resolve keys, and
[`component_i18n_test.go`](../../core/component_i18n_test.go) for runnable
examples of both forms above.
