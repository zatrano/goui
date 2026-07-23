# 17. Page modes (Live / SEO / Static)

GoUI can deliver the **same component** three ways. Pick the mode at
registration time — routes stay one-liners.

| Mode | First HTTP response | WebSocket | Typical use |
|------|---------------------|-----------|-------------|
| `ModeLive` (default) | Empty `#app` shell + client script | Yes | Admin, ERP, dashboards |
| `ModeSEO` | Full HTML body + `<head>` meta + client script | Yes (hydrates) | Marketing, blog, product pages |
| `ModeStatic` | Full HTML body + meta, **no** client script | No | Legal, about, pure content |

## Register

```go
registry.Register("orders", NewOrders) // ModeLive

registry.RegisterPage("landing", NewLanding, core.ModeSEO)
registry.RegisterPage("privacy", NewPrivacy, core.ModeStatic)
```

`Register` is unchanged and always `ModeLive`.

## Optional Head metadata

Implement `core.HeadProvider` on SEO/Static components:

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

// Fiber (same idea for Gin / Echo / stdlib)
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

Or mount a single path:

```go
app.Get("/product", gouifiber.Page(renderer, "product"))
// net/http:
mux.Handle("/product", renderer.Handler("product"))
```

## Request access in Mount

SEO/Static handlers put `*http.Request` on the context:

```go
func (p *Product) Mount(ctx context.Context) error {
    req := core.RequestFromContext(ctx)
    // req.URL.Query(), path, headers…
    return nil
}
```

## How ModeSEO hydrates

1. GET returns HTML with `data-goui-component="ssr"` and `data-goui-ssr="1"`.
2. `goui.js` opens the WebSocket and receives the first full render.
3. The client **adopts** the existing DOM (no flash), remaps the component id,
   then applies later patches as usual.

`ModeLive` still works with a hand-written `index.html` if you prefer; the
page renderer is optional.

## Example

See [`examples/seo-pages`](../../examples/seo-pages): `/` (SEO), `/about` (Static),
`/admin` (Live).
