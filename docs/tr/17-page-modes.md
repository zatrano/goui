# 17. Sayfa modları (Live / SEO / Static)

GoUI **aynı bileşeni** üç şekilde sunabilir. Modu kayıt anında seçersiniz;
route tanımı tek satır kalır.

| Mod | İlk HTTP yanıtı | WebSocket | Tipik kullanım |
|-----|-----------------|-----------|----------------|
| `ModeLive` (varsayılan) | Boş `#app` kabuğu + istemci script | Evet | Admin, ERP, panel |
| `ModeSEO` | Dolu HTML + `<head>` meta + istemci script | Evet (hydrate) | Landing, blog, ürün |
| `ModeStatic` | Dolu HTML + meta, **istemci yok** | Hayır | Yasal, hakkında, düz içerik |

## Kayıt

```go
registry.Register("orders", NewOrders) // ModeLive

registry.RegisterPage("landing", NewLanding, core.ModeSEO)
registry.RegisterPage("privacy", NewPrivacy, core.ModeStatic)
```

`Register` davranışı değişmedi; her zaman `ModeLive`.

## İsteğe bağlı Head meta

SEO/Static bileşenlerde `core.HeadProvider` uygulayın:

```go
func (l *Landing) Head() core.Head {
    return core.Head{
        Title:       "Acme — Ana sayfa",
        Description: "…",
        Canonical:   "https://example.com/",
        OGTitle:     "Acme",
        OGImage:     "https://example.com/og.png",
    }
}
```

## Page renderer bağlama

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

Tek path:

```go
app.Get("/product", gouifiber.Page(renderer, "product"))
mux.Handle("/product", renderer.Handler("product"))
```

## Mount içinde request

SEO/Static handler `*http.Request`'i context'e koyar:

```go
func (p *Product) Mount(ctx context.Context) error {
    req := core.RequestFromContext(ctx)
    // req.URL.Query(), path, header…
    return nil
}
```

## ModeSEO hydrate

1. GET, `data-goui-component="ssr"` ve `data-goui-ssr="1"` ile HTML döner.
2. `goui.js` WebSocket açar, ilk full render gelir.
3. İstemci mevcut DOM'u **devralır** (flash yok), id'yi günceller; sonraki
   patch'ler normal akar.

`ModeLive` için elle `index.html` yazmaya devam edebilirsiniz; page renderer
zorunlu değildir.

## Örnek

[`examples/seo-pages`](../../examples/seo-pages): `/` (SEO), `/about` (Static),
`/admin` (Live).
