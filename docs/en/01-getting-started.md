# 01 — Getting Started

GoUI is a Go server-driven UI framework: you write
components in Go, the server owns all state and renders HTML, and a small
vanilla-JS runtime in the browser applies minimal DOM patches over a
WebSocket connection. There is no client-side component tree, no build
pipeline, and no second copy of your domain model in JavaScript.

This guide gets a new machine from zero to a running GoUI app, explains the
project skeleton, walks through your first component, and shows how to run
every example that ships with the repository.

## 1. Requirements

- **Go 1.25 or newer** (the module itself targets `go 1.25.0`; check with `go version`)
- A browser with WebSocket support (all modern browsers)
- No Node.js, no npm, no bundler — the client runtime is plain ES modules served as static files

## 2. Install

GoUI is a normal Go module. Add it to your project with:

```bash
go get github.com/zatrano/goui@latest
```

The module path is **always** `github.com/zatrano/goui`. Every subpackage is
imported from underneath it, for example:

```go
import (
    "github.com/zatrano/goui/core"
    "github.com/zatrano/goui/forms"
    "github.com/zatrano/goui/i18n"
    "github.com/zatrano/goui/upload"
    "github.com/zatrano/goui/validation"
    "github.com/zatrano/goui/ws"
)
```

GoUI is **framework-agnostic**: the core module has no HTTP-router dependency.
WebSocket and upload routes are wired through small adapter modules. Add the
core module plus whichever adapter matches your stack:

```bash
go get github.com/zatrano/goui@latest
go get github.com/zatrano/goui/adapters/fiber@latest   # or gin, echo, stdlib
```

See [13-project-integration.md](13-project-integration.md) for a full adapter
comparison. The shipped demos use the Fiber adapter; proof examples live under
`examples/adapters/{nethttp,chi,gin,echo}`.

## 3. Project skeleton

A minimal GoUI application has this shape:

```
myapp/
├── go.mod
├── main.go              # HTTP app, registry, hub, adapter routes
├── index.html            # loads /client/goui.js and connects
└── (optional) i18n/
    └── locales/
        ├── tr.json
        └── en.json
```

`main.go` is responsible for four things:

1. Building a `core.Registry` and registering your component(s) by name.
2. Building an `i18n.Translator` (optionally loading locale files).
3. Building a `ws.Hub`, wrapping it in `ws.NewServer(hub, registry, translator)`,
   and registering `GET /goui/ws` through your chosen adapter (see §2).
4. Serving the GoUI client runtime (`client/`) and your page's `index.html` as static files.

Serve `client/` however your HTTP stack normally serves static files — for
example with Fiber:

```go
app.Use("/client", static.New("./client"))
```

## 4. Your first component

Every GoUI component implements the `core.Component` interface:

```go
type Component interface {
    Mount(ctx context.Context) error
    Render() (string, error)
    HandleEvent(ctx context.Context, event string, payload map[string]any) error
    Unmount(ctx context.Context) error
}
```

Embed `core.BaseComponent` to get dirty-tracking, i18n, and toast helpers for
free (see [`02-components.md`](02-components.md) for the full contract).

Here is the canonical "Counter" component:

```go
package main

import (
    "context"
    "log"
    "path/filepath"

    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/static"

    gouifiber "github.com/zatrano/goui/adapters/fiber"
    "github.com/zatrano/goui/core"
    "github.com/zatrano/goui/i18n"
    "github.com/zatrano/goui/ws"
)

type Counter struct {
    core.BaseComponent
    Count int
}

func (c *Counter) Mount(_ context.Context) error { return nil }

func (c *Counter) Render() (string, error) {
    html, err := core.RenderTemplate(`<div class="counter">
<span class="count">{{.Count}}</span>
<button type="button" g-click="increment">+</button>
<button type="button" g-click="decrement">-</button>
</div>`, c)
    if err != nil {
        return "", err
    }
    c.ResetDirty()
    return html, nil
}

func (c *Counter) HandleEvent(_ context.Context, event string, _ map[string]any) error {
    switch event {
    case "increment":
        c.Count++
        c.MarkDirty()
    case "decrement":
        c.Count--
        c.MarkDirty()
    }
    return nil
}

func (c *Counter) Unmount(_ context.Context) error { return nil }

func main() {
    registry := core.NewRegistry()
    if err := registry.Register("counter", func() core.Component { return &Counter{} }); err != nil {
        log.Fatal(err)
    }

    translator := i18n.NewTranslator()
    hub := ws.NewHub()

    app := fiber.New()
    app.Use("/client", static.New(filepath.Join(".", "client")))
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendFile("index.html")
    })

    gouifiber.Register(app, gouifiber.Options{
        Server: ws.NewServer(hub, registry, translator),
    })

    log.Println("listening on http://localhost:3000")
    log.Fatal(app.Listen(":3000"))
}
```

And the matching `index.html`:

```html
<!DOCTYPE html>
<html lang="tr">
<head>
  <meta charset="UTF-8">
  <title>GoUI Counter</title>
</head>
<body>
  <div id="app"></div>
  <script type="module">
    import { GoUIClient } from '/client/goui.js';
    const client = new GoUIClient('/goui/ws', 'counter', { mount: '#app', locale: 'tr' });
    client.connect();
  </script>
</body>
</html>
```

What happens when this page loads:

1. `GoUIClient` opens a WebSocket to `/goui/ws?component=counter&locale=tr`.
2. The server creates a `ws.Session`, asks the `Registry` to build a `Counter`,
   calls `Mount`, then sends a `session` frame followed by the first `render` frame.
3. Clicking `+` sends an `event` frame (`{"type":"event","event":"increment", ...}`);
   the server calls `HandleEvent`, re-renders, diffs the old/new HTML tree, and
   streams back a minimal `render` patch.
4. If the tab disconnects (reload, network blip), the session is kept alive
   for a grace period (default 60s, see [`04-sessions-and-websocket.md`](04-sessions-and-websocket.md))
   so reconnecting restores state instead of starting over.

## 5. Running the shipped examples

The repository ships ten runnable demos under `examples/`, each a standalone
`main.go` you can `go run` directly from the repo root. Every example serves
its own static HTML page, mounts the GoUI WebSocket route, and listens on a
dedicated port so you can run several side by side:

| Port | Command                                | Demo                          | What it shows |
|------|-----------------------------------------|--------------------------------|---------------|
| 3000 | `go run ./examples/counter`             | **counter**                    | Minimal `Component` lifecycle, `g-click`, dirty tracking |
| 3001 | `go run ./examples/contact-form`        | **contact-form**               | Native form fields, validation, `Toast`/`ToastT`, prefetch → activate |
| 3002 | `go run ./examples/searchable-select`   | **searchable-select**          | Select family: Searchable Select, Multi Select, Combobox, Autocomplete, Tag Input, Tree Select, Cascader, Dual Listbox |
| 3003 | `go run ./examples/numeric-controls`    | **numeric-controls**           | Currency Input, Percentage Input, Rating |
| 3004 | `go run ./examples/field-meta`          | **field-meta**                 | Character counter (`ShowCharCount`) and password strength (`ShowStrength`) |
| 3005 | `go run ./examples/date-controls`       | **date-controls**               | Date Range Picker, Time Range Picker, Calendar Date Picker |
| 3006 | `go run ./examples/identity-inputs`     | **identity-inputs**            | OTP/PIN, Country/Language/Timezone/Currency Picker, Phone Input |
| 3007 | `go run ./examples/editors`             | **editors**                    | Markdown Editor (goldmark), Rich Text (Quill), Code Editor (CodeMirror) |
| 3008 | `go run ./examples/media-upload`        | **media-upload**               | Drag & Drop Upload, Image Upload, Avatar Upload + crop |
| 3009 | `go run ./examples/misc-controls`       | **misc-controls**              | Emoji/Icon/Font Picker, Swatch Color Picker, Gradient Picker, Mention, Signature Pad |
| 3010 | `go run ./examples/adapters/nethttp`    | **net/http adapter**           | Counter on plain `net/http` + `adapters/stdlib` |
| 3011 | `go run ./examples/adapters/chi`        | **Chi adapter**                | Counter on Chi via `adapters/stdlib` `Mount` |
| 3012 | `go run ./examples/adapters/gin`        | **Gin adapter**                | Counter on Gin |
| 3013 | `go run ./examples/adapters/echo`       | **Echo adapter**               | Counter on Echo |

Run any of them from the repository root, then open the printed URL:

```bash
go run ./examples/counter
# GoUI counter example at http://localhost:3000

go run ./examples/contact-form
# GoUI contact form at http://localhost:3001

go run ./examples/searchable-select
# GoUI searchable-select demo at http://localhost:3002
```

Because each example listens on its own port, you can start several at once
in separate terminals — useful when comparing different form controls side
by side. Each example resolves the repository root via `runtime.Caller(0)` so
it works regardless of your current working directory, and mounts:

- `/client` → the framework's JS runtime (`client/`)
- `/forms` → `forms/style.css` and related static assets (where used)
- `/goui/ws` → the WebSocket endpoint (`ws.Path`, mounted by your adapter)
- `/goui/upload`, `/goui/files/:id` → file upload endpoints (media-upload, misc-controls; via adapter `Store` option or `upload.Mount`)

## 6. Where to go next

- [`02-components.md`](02-components.md) — the full `Component` / `BaseComponent` / `Registry` / template contract
- [`03-i18n.md`](03-i18n.md) — translator setup and locale files
- [`04-sessions-and-websocket.md`](04-sessions-and-websocket.md) — Session/Hub lifecycle and the wire protocol
- [`05-forms-tier1.md`](05-forms-tier1.md) — every native form control
- [`06-validation.md`](06-validation.md) — server-side validation rules
- [`07-forms-tier2.md`](07-forms-tier2.md) — every rich form control
- [`17-page-modes.md`](17-page-modes.md) — ModeLive / ModeSEO / ModeStatic for admin vs public HTML
