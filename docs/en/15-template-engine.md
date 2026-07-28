# 15 — Template Engine

GoUI ships a Blade-inspired template engine on top of Go’s native
`html/template`. You write `.goui.html` files with familiar directives
(`@if`, `@extends`, `@component`, …); at startup they are transpiled once
into native Go templates and kept in memory. There is **no** runtime string
eval and **no** reimplementation of Go’s expression language — `{{ .Field }}`
pipelines are copied through unchanged, so context-aware auto-escaping stays
intact.

See also: [migrating from `RenderTemplate`](16-migrating-to-template-engine.md),
example [`examples/counter-view`](../../examples/counter-view).

## 1. Philosophy

The engine is a **structural preprocessor**:

| Handled by GoUI | Left to `html/template` |
|-----------------|-------------------------|
| `@if` / `@foreach` / `@extends` / `@include` / `@component` | `{{ .Field }}`, pipelines, `eq`, functions |
| File layout → dot-path names | Auto-escaping |
| Compile-time dependency graph | Execution |

## 2. Setup

- Extension: `.goui.html`
- Dot-path: `views/pages/home.goui.html` → `"pages.home"` (relative to `Root`)

```go
import gouitemplate "github.com/zatrano/goui/template"

reg, err := gouitemplate.NewRegistry(gouitemplate.Config{
    Root:        "./views",
    StrictProps: true, // recommended in production
})
if err != nil {
    log.Fatal(err)
}
defer reg.Close()

html, err := reg.Render("pages.home", data)
```

## 3. Directive reference

### Conditionals

```html
@if(.User.IsAdmin)
  <span>Admin</span>
@elseif(.User.Moderator)
  <span>Mod</span>
@else
  <span>User</span>
@endif

@unless(.Hidden)
  visible
@endunless
```

### Loops

Always use `$`-prefixed names (never rely on `.` rebinding):

```html
@foreach(.Items as $item)
  <li>{{ $item.Name }}</li>
@empty
  <li>None</li>
@endforeach

@foreach(.Items as $key, $item)
  <li>{{ $key }}: {{ $item }}</li>
@endforeach
```

### Switch

```html
@switch(.Status)
@case("ok")
  OK
@break
@default
  Other
@endswitch
```

### Output

```html
{{ .Name }}           <!-- escaped -->
{!! .TrustedHTML !!}  <!-- raw; see Security -->
{{-- comment --}}
@@literal             <!-- prints a single @ -->
```

### Helpers (`BaseFuncMap`)

```html
{{ default "Guest" .User.Name }}
{{ dict "Type" "submit" "Label" "Save" }}
{{ list 1 2 3 }}
```

## 4. Layouts (`@extends` / `@section` / `@yield`)

`layouts/app.goui.html`:

```html
<html>
<head><title>@yield("title", "App")</title></head>
<body>
  @yield("content")
</body>
</html>
```

`pages/home.goui.html`:

```html
@extends("layouts.app")
@section("title", "Home")
@section("content")
  <h1>Welcome</h1>
@endsection
```

With `@extends`, only `@section` blocks (plus whitespace) are allowed at the
top level.

## 5. Includes

```html
@include("partials.nav")
@include("partials.user", .User)
@includeIf("partials.optional")  <!-- omitted at compile time if missing -->
```

Missing `@include` targets fail at `NewRegistry`. `@includeIf` never fails.

## 6. Components and slots

Component file `components/card.goui.html`:

```html
@props(Title string)
<div class="card">
  @if(.Slots.header)
    <header>{{ .Slots.header }}</header>
  @endif
  <div>{{ .DefaultSlot }}</div>
</div>
```

Caller:

```html
@component("components.card", dict "Title" "Hi")
  @slot("header")
    {{ .PageTitle }}
  @endslot
  Default body
@endcomponent
```

Inside the component template, `.` is a `Dot`: use `.Props.*`, `.Slots.name`,
and `.DefaultSlot`. Slot bodies render in the **caller’s** data context.

Nested components are supported (a component may call another).

## 7. `@props` and `StrictProps`

```html
@props(Name string, Count int = 0)
```

With `StrictProps: true`, `NewRegistry` checks that every `.Props.X` used in
the file was declared (typos fail fast with a “did you mean …?” hint). Unused
declarations become soft warnings via `reg.Warnings()`.

When `StrictProps` is false (default), these checks are skipped.

## 8. Hot reload (dev)

```go
hub := ws.NewHub()
reg, err := gouitemplate.NewRegistry(gouitemplate.Config{
    Root:            "./views",
    WatchForChanges: true, // false in production
    OnReload: func() {
        hub.Broadcast(ws.PushMessage{
            Kind: "reload",
            Text: "templates updated",
        })
    },
    OnReloadError: func(err error) {
        log.Printf("template reload: %v", err)
    },
})
defer reg.Close()
```

The template package does **not** import `ws`; you wire the callback yourself.
Failed reloads keep the last good compile in memory.

## 9. `ViewComponent` integration

```go
type Counter struct {
    core.BaseComponent
    Count int
}

func (c *Counter) View() string { return "counter" }
func (c *Counter) Render() (string, error) {
    return "", gouitemplate.ErrViewRenderDirect
}

tmplReg, _ := gouitemplate.NewRegistry(gouitemplate.Config{Root: "./views"})
coreReg.Register("counter", gouitemplate.Wrap(tmplReg, func() core.Component {
    return &Counter{}
}))
```

See `examples/counter-view`.

## 10. Security

- Prefer `{{ }}` (escaped).
- `{!! !!}` / `raw` disables auto-escape — only for trusted, pre-sanitized HTML.
- Never pass unsanitized user input through raw output.

## 11. Blade comparison

| Blade | GoUI |
|-------|------|
| `@if` / `@foreach` / `@extends` | Same idea → native `html/template` |
| `@component` / `@slot` | Supported (two-phase render) |
| `@includeIf` | Compile-time |
| `@props` | Opt-in name checks (`StrictProps`) |
| `@php` / arbitrary PHP | **Not supported (by design)** — no arbitrary code in templates |
| Per-request file mtime cache | Process-lifetime in-memory compile |

Omitting `@php`-style escape hatches is intentional: templates stay data +
structure, logic stays in Go.

## Performance

Order-of-magnitude on a typical laptop (see `go test ./template/ -bench=.`):

| Operation | Rough cost |
|-----------|------------|
| `Render` simple page | low µs / op |
| `Render` with extends | low–mid µs / op |
| `Render` ~20 components | tens of µs / op |
| `NewRegistry` ~100 files | tens–low hundreds of ms |

Compile once at startup (or on hot reload); `Render` does no disk I/O.
