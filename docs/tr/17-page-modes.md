# 17. Sayfa modları (Live / SEO / Static)

GoUI **aynı bileşeni** üç şekilde sunabilir. Modu kayıt anında seçersiniz;
route tanımı tek satır kalır.

| Mod | İlk HTTP yanıtı | WebSocket | Tipik kullanım |
|-----|-----------------|-----------|----------------|
| `ModeLive` (varsayılan) | **SSR HTML** + istemci (`DeferFirstRender` ile boş kabuk) | Evet (hydrate/adopt) | Admin, ERP, panel |
| `ModeSEO` | SSR HTML + `<head>` meta + istemci | Evet (hydrate/adopt) | Landing, blog, ürün |
| `ModeStatic` | Dolu HTML + meta, **istemci yok** | Hayır | Yasal, hakkında, düz içerik |

## ModeLive “boş kabuk + WS” değildir

Sunucu sahibi UI (Vue SSR, LiveView dead render, GoUI) demek **ilk boyama
GET üzerinde senkron** demektir. WebSocket sonraki event/patch içindir; ilk
HTML için değil.

Page renderer kullanıldığında ModeLive ve ModeSEO aynı first-paint yolunu
paylaşır. Asıl fark **Head / crawler meta** (`HeadProvider`); “async vs sync
ilk render” değil.

Boş `#app` + WS beklemek **opt-in** (`DeferFirstRender`) — başlangıç
varsayılanı değil.

## Kayıt

```go
registry.Register("orders", NewOrders) // ModeLive

registry.RegisterPage("landing", NewLanding, core.ModeSEO)
registry.RegisterPage("privacy", NewPrivacy, core.ModeStatic)
```

## İsteğe bağlı Head meta

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

## Page renderer

```go
gouifiber.Register(app, gouifiber.Options{
    Server: server,
    Page:   renderer,
    Routes: []page.Route{
        {Path: "/", Component: "landing"},
        {Path: "/admin", Component: "orders"},
    },
})
```

Adapter'lar `server.Pending` paylaşır → HTTP `Mount` WS'te **adopt** edilir
(ikinci Mount yok).

## İlk boyamayı ayarlama

```go
Routes: []page.Route{
    {
        Path:              "/admin",
        Component:         "orders",
        FastRenderTimeout: 80 * time.Millisecond,
        SkeletonHTML:      page.SkeletonRows("320px", 5),
    },
    {
        Path:             "/legacy",
        Component:        "legacy",
        DeferFirstRender: true, // nadiren
        SkeletonHTML:     page.SkeletonBlock("240px"),
    },
},
```

| Alan | Varsayılan | Rol |
|------|------------|-----|
| (yok) | sync SSR | Vue/LiveView tarzı ilk boyama |
| `FastRenderTimeout` | kapalı | Mount bekleme tavanı; kaçarsa iskelet + park |
| `SkeletonHTML` | boş | Kaçış / defer için yer tutucu |
| `DeferFirstRender` | false | Eski boş kabuk (opt-in) |

### `Refresh`

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

## Trade-off'lar

- Varsayılan etkileşimli sayfalar ilk HTML için ekstra round-trip eklemez.
- Yavaş Mount ya GET'i bekler ya timeout + iskelet.
- WS gürültü değil; event kanalıdır.
- Nested streaming yok; `Refresh` kullanın.

## Örnek

[`examples/seo-pages`](../../examples/seo-pages).
