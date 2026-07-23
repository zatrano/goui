<p align="center">
  <img src="assets/goui-banner.png" alt="GoUI — Server-driven Go UI — WebSocket & SEO HTML">
</p>

**GoUI** is a Go server-driven UI framework. You write components in Go; the server owns state, renders HTML, and pushes minimal DOM patches over WebSocket — or serves full HTML on first paint with `ModeSEO` / `ModeStatic` for public pages. The browser runs a small vanilla JS runtime—no React/Vue bundle, no client-side component tree.

Inspired by the LiveView idea (server-authoritative views over a persistent connection), GoUI is an independent implementation with a framework-agnostic core, HTTP adapters (net/http, Chi, Fiber, Gin, Echo), a keyed HTML diff engine, a progressive forms library, and a Blade-inspired `.goui.html` template engine on top of Go’s `html/template`.

Use GoUI when you want one language (Go) for domain logic and UI, keep form state on the server (no Laravel-style `old()` gymnastics after validation failure), and ship interactive pages without a SPA toolchain.

## Why GoUI?

| Approach | What you get | Cost |
|----------|--------------|------|
| React/Vue SPA | Rich client UX, huge ecosystem | Duplicate models, API surface, build pipeline |
| Classic MPA | Simple HTML | Full page reloads, awkward form round-trips |
| HTMX | Progressive enhancement | Still mostly request/response; complex state is DIY |
| **GoUI** | Go components, live WS patches, optional SEO/static HTML | Persistent WS per interactive session |

**Prefer GoUI when:** internal tools, admin/ERP panels, multi-step forms, tenant apps where Go already owns the domain — and **public pages** when you opt into `ModeSEO` / `ModeStatic` (first paint is real HTML).

**Prefer something else when:** offline-first mobile apps; millions of concurrent cheap page views where long-lived WebSockets are too expensive; teams that need a large client-component marketplace.

## Architecture

```
Browser (goui.js)
    │  event / prefetch / activate
    ▼
Session ──► Component.HandleEvent / Mount
    │
    ▼
Render HTML ──► Diff (old tree → patches) ──► Frame(render)
    │
    ▼
Hub (sessions, grace reconnect, Broadcast)
```

1. Client connects to `/goui/ws?component=…`
2. Server creates a `Session`, mounts the component, sends `session` + initial `render` (`OpReplace`)
3. User events become `event` frames → `HandleEvent` → re-render → minimal patches
4. Optional: `prefetch` mounts silently; `activate` promotes and renders
5. Disconnects keep the session for a **grace period** (default 60s) so reconnect restores state

## Features

### Core
- `Component` lifecycle: `Mount` / `Render` / `HandleEvent` / `Unmount`
- `BaseComponent`: dirty tracking, i18n helpers, toast helpers
- `Registry` factories, HTML template cache (`RenderTemplate`)

### i18n
- JSON locales, nested-flat keys (`form.required_field`)
- Fallback to base locale `tr`, then `[[key]]` placeholder

### WebSocket / Session / Hub
- Framework-agnostic `ws.Server` + nested HTTP adapters
- Reconnect with session id; grace period cleanup
- Frames: `event`, `render`, `push`, `error`, `session`, `prefetch`, `activate`

### Diff engine
- HTML parse → tree → patches (`replace`, `update_text`, `set_attr`, `remove_attr`, `insert`, `remove`, `move`)
- Keyed list diff via `data-key` (simple key-map, not LCS)

### Client runtime
- Vanilla JS: patch apply, event delegation (`g-click`, `g-change`, `g-input`, `g-submit`)
- Modules: toast, prefetch, selectable, calendar, otp, richtext, codeeditor, upload, avatar, signature

### Forms
TextInput, NumericInput, DateTimeInput, ChoiceInput (checkbox/radio), FileInput, ColorInput, HiddenInput, Textarea, Select/Option/Optgroup, Button, Form/Fieldset/Legend/Label, Datalist, Output, Meter, Progress, Searchable Select, Multi Select, Combobox, Autocomplete, Tag/Chips, Tree Select, Cascader, Dual Listbox, Currency, Percentage, Rating, Date Range, Time Range, Calendar Picker, OTP/PIN, Phone, Country/Language/Timezone/Currency pickers, Rich Text (Quill), Markdown (goldmark), Code Editor (CodeMirror), Drag&Drop / Image / Avatar upload + cropper overlay, Emoji/Icon/Font pickers, Color swatch / Gradient, Signature, Mention, Character counter, Password strength

### Validation
`Required`, `MinLength`, `MaxLength`, `Pattern`, `Email`, `NumericRange`, `Custom` — server-side; state stays in the component after failed validation

### Push / Toast
`Toast` / `ToastT`, `Hub.Broadcast`, kinds: success / error / warning / info

### Prefetch
`data-goui-prefetch` + `data-goui-activate`; silent Mount; LRU cap 5; no pre-render

### Page modes (SEO)
- `ModeLive` (default), `ModeSEO` (SSR HTML + WS hydrate), `ModeStatic` (HTML only)
- `Registry.RegisterPage`, `page.NewRenderer`, adapter `Routes` / `Page(...)`
- `HeadProvider` for title / description / Open Graph
- Guide: [docs/en/17-page-modes.md](docs/en/17-page-modes.md) · Example: [`examples/seo-pages`](examples/seo-pages)

## Template Engine (Blade-inspired)

File-based `.goui.html` views with `@extends` / `@section` / `@yield`, `@include`,
`@component` / `@slot`, opt-in `@props` checks, and optional hot reload — compiled
once onto native `html/template` (auto-escaping preserved).

- Guide: [docs/en/15-template-engine.md](docs/en/15-template-engine.md)
- Migration from `RenderTemplate`: [docs/en/16-migrating-to-template-engine.md](docs/en/16-migrating-to-template-engine.md)
- Example: [`examples/counter-view`](examples/counter-view)

## Requirements

- **Go** `1.25.0` (see `go.mod`)
- One HTTP adapter: `adapters/stdlib` (net/http / Chi), `adapters/fiber`, `adapters/gin`, or `adapters/echo`
- Browser: WebSocket; IntersectionObserver recommended for prefetch on mobile
- **Tailwind CLI** (optional) — only if you want utility CSS beyond `forms/style.css`

## Install

```bash
go get github.com/zatrano/goui@latest
# pick an adapter, e.g.:
go get github.com/zatrano/goui/adapters/stdlib@latest
# or: adapters/fiber | adapters/gin | adapters/echo
```

## HTTP adapters

| Stack | Module | Mount helper |
|-------|--------|----------------|
| net/http | `adapters/stdlib` | `Register(mux, opts)` |
| Chi | `adapters/stdlib` | `Mount(router, opts)` |
| Fiber v3 | `adapters/fiber` | `Register(app, opts)` |
| Gin | `adapters/gin` | `Register(router, opts)` |
| Echo | `adapters/echo` | `Register(echo, opts)` |

## Quick Start

```go
package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	gouistdlib "github.com/zatrano/goui/adapters/stdlib"
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
	case "decrement":
		c.Count--
	}
	return nil
}

func (c *Counter) Unmount(_ context.Context) error { return nil }

func main() {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")) // adjust for your layout

	registry := core.NewRegistry()
	_ = registry.Register("counter", func() core.Component { return &Counter{} })

	mux := http.NewServeMux()
	mux.Handle("/client/", http.StripPrefix("/client/", http.FileServer(http.Dir(filepath.Join(root, "client")))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html") // HTML that loads /client/goui.js
	})

	gouistdlib.Register(mux, gouistdlib.Options{
		Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator()),
	})
	log.Fatal(http.ListenAndServe(":3000", mux))
}
```

Minimal HTML:

```html
<div id="app"></div>
<script type="module">
  import { GoUIClient } from '/client/goui.js';
  new GoUIClient('/goui/ws', 'counter', { mount: '#app' }).connect();
</script>
```

Or run the shipped demo:

```bash
go run ./examples/counter
# http://localhost:3000
```

## Repository layout

| Path | Role |
|------|------|
| `core/` | Component contract, registry, template cache |
| `i18n/` | Translator + bundled locale JSON |
| `ws/` | Session, Hub, frames, framework-agnostic `Server` |
| `diff/` | HTML parse, keyed diff, patches |
| `forms/` | Form controls, including searchable selects and pickers |
| `validation/` | Rule helpers |
| `upload/` | `Storage` + `LocalStore` + `net/http` handler |
| `adapters/` | Nested modules: stdlib, fiber, gin, echo |
| `client/` | Browser runtime + modules |
| `examples/` | Runnable demos (Fiber demos + `examples/adapters/*`) |
| `docs/` | Full guides (`docs/en`, `docs/tr`) |

## Documentation

- English guides: [`docs/en/`](docs/en/)
- Turkish guides: [`docs/tr/`](docs/tr/) · overview: [`README.tr.md`](README.tr.md)

Start with [Getting started](docs/en/01-getting-started.md), [Project integration](docs/en/13-project-integration.md), and the [Template engine](docs/en/15-template-engine.md).

## Roadmap / known limits

- Some advanced form controls are not implemented yet (future / optional).
- **Upload** ships with `LocalStore` only; `upload.Storage` is ready for S3/MinIO implementations.
- Diff is optimized for typical admin UIs; always use `data-key` on dynamic lists.
- Prefetch is mount-only (intentionally does not pre-send HTML).

## License

MIT draft — see [`LICENSE`](LICENSE). Confirm before tagging a public release.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md).

## Contact

Project: [github.com/zatrano/goui](https://github.com/zatrano/goui)
Issues and PRs welcome once the repository is published.
