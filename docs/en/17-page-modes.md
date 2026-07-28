# 17. Page modes (Live / SEO / Static)

GoUI can deliver the **same component** three ways. Pick the mode at
registration time — routes stay one-liners.

| Mode | First HTTP response | WebSocket | Typical use |
|------|---------------------|-----------|-------------|
| `ModeLive` (default) | **SSR HTML** + client (empty shell only if `DeferFirstRender`) | Yes (hydrate/adopt) | Admin, ERP, dashboards |
| `ModeSEO` | SSR HTML + `<head>` meta + client | Yes (hydrate/adopt) | Marketing, blog, product pages |
| `ModeStatic` | Full HTML + meta, **no** client script | No | Legal, about, pure content |

## Why ModeLive is not “empty shell + WS”

Server-owned UI (Vue SSR, LiveView dead render, GoUI) means the **first
paint is synchronous on the GET**. The WebSocket is for later events and
patches — not for the first HTML.

ModeLive and ModeSEO share that first-paint path when you use the page
renderer. The meaningful difference is **document Head / crawler metadata**
(`HeadProvider`), not “async vs sync first render”.

Empty `#app` + wait for WS is an **opt-in** escape hatch
(`DeferFirstRender`) for rare cases — not the beginner default.

## Register

```go
registry.Register("orders", NewOrders) // ModeLive

registry.RegisterPage("landing", NewLanding, core.ModeSEO)
registry.RegisterPage("privacy", NewPrivacy, core.ModeStatic)
```

`Register` is unchanged and always `ModeLive`.

## Optional Head metadata

Implement `core.HeadProvider` on SEO/Static components (also fine on Live):

```go
func (l *Landing) Head() core.Head {
    return core.Head{
        Title:       "Acme — Home",
        Description: "…",
        Canonical:   "https://example.com/",
        OGTitle:     "Acme",
        OGImage:     "https://example.com/og.png",
    }
}
```

## Mount the page renderer

```go
import "github.com/zatrano/goui/page"

renderer := page.NewRenderer(page.Options{
    Registry:   registry,
    Translator: translator,
})

gouifiber.Register(app, gouifiber.Options{
    Server: server,
    Page:   renderer,
    Routes: []page.Route{
        {Path: "/", Component: "landing"},
        {Path: "/about", Component: "privacy"},
        {Path: "/admin", Component: "orders"},
    },
})
```

Adapters share `server.Pending` with the renderer so HTTP `Mount` is
**adopted** by the WebSocket (no second `Mount`).

## Tuning first paint

```go
Routes: []page.Route{
    {
        Path:              "/admin",
        Component:         "orders",
        FastRenderTimeout: 80 * time.Millisecond, // optional cap
        SkeletonHTML:      page.SkeletonRows("320px", 5),
    },
    // Rare: skip SSR on purpose
    {
        Path:             "/legacy",
        Component:        "legacy",
        DeferFirstRender: true,
        SkeletonHTML:     page.SkeletonBlock("240px"),
    },
},
```

| Field | Default | Role |
|-------|---------|------|
| (none) | sync SSR | Vue/LiveView-style first paint |
| `FastRenderTimeout` | off | Cap Mount wait; miss → skeleton + park |
| `SkeletonHTML` | empty | Reserved layout on miss / defer |
| `DeferFirstRender` | false | Opt-in empty shell (old ModeLive) |

`FastRenderTimeout` requires `PendingMounts` (`ErrPendingRequired`).

### Async fill: `Refresh`

```go
func (o *Orders) Mount(ctx context.Context) error {
    go func() {
        rows, err := o.load(ctx)
        if err != nil { return }
        o.Rows = rows
        o.Refresh()
    }()
    return nil
}
```

## Honest trade-offs

- Default interactive pages **do not** add an extra round trip for first HTML.
- Slow `Mount` either blocks the GET (default) or you set a timeout + skeleton.
- WS is still required for interactivity; it is not “noise” — it is the event channel.
- Nested progressive sub-component streaming is not built-in; use `Refresh`.

## Request access in Mount

```go
func (p *Product) Mount(ctx context.Context) error {
    req := core.RequestFromContext(ctx)
    return nil
}
```

## Hydrate

1. GET returns HTML with `data-goui-ssr="1"` (and `pending` when parked).
2. Client adopts the DOM on first WS render (no flash).
3. Later events apply diff patches.

## Example

See [`examples/seo-pages`](../../examples/seo-pages).
