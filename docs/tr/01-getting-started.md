# 01 — Başlarken

GoUI, sunucu güdümlü bir Go UI çatısıdır: bileşenleri
Go ile yazarsınız, tüm durumun (state) sahibi sunucudur ve HTML'i sunucu
render eder; tarayıcıdaki küçük, vanilla-JS bir çalışma zamanı (runtime) ise
bir WebSocket bağlantısı üzerinden minimum DOM yamalarını uygular. İstemci
tarafında bir bileşen ağacı yoktur, bir build (derleme) hattı yoktur ve
domain modelinizin JavaScript'te ikinci bir kopyası yoktur.

Bu kılavuz, sıfırdan çalışan bir GoUI uygulamasına giden yolu gösterir, proje
iskeletini açıklar, ilk bileşeninizi adım adım gösterir ve repoyla birlikte
gelen tüm örnekleri nasıl çalıştıracağınızı anlatır.

## 1. Gereksinimler

- **Go 1.25 veya daha yeni** (modülün kendisi `go 1.25.0`'ı hedefler; `go version` ile kontrol edin)
- WebSocket desteği olan bir tarayıcı (tüm modern tarayıcılar)
- Node.js yok, npm yok, bundler yok — istemci çalışma zamanı, statik dosya olarak sunulan düz ES modülleridir

## 2. Kurulum

GoUI normal bir Go modülüdür. Projenize şu şekilde ekleyin:

```bash
go get github.com/zatrano/goui@latest
```

Modül yolu **her zaman** `github.com/zatrano/goui`'dir. Her alt paket bunun
altından import edilir, örneğin:

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

GoUI **framework-agnostic**'tir: çekirdek modülün HTTP yönlendirici bağımlılığı
yoktur. WebSocket ve yükleme route'ları küçük adapter modülleriyle bağlanır.
Çekirdek modülü ve yığınınıza uyan adapter'ı ekleyin:

```bash
go get github.com/zatrano/goui@latest
go get github.com/zatrano/goui/adapters/fiber@latest   # veya gin, echo, stdlib
```

Tam adapter karşılaştırması için
[13-project-integration.md](13-project-integration.md)'ye bakın. Repodaki
demolar Fiber adapter'ını kullanır; kanıt örnekleri
`examples/adapters/{nethttp,chi,gin,echo}` altındadır.

## 3. Proje iskeleti

Minimal bir GoUI uygulaması şu şekle sahiptir:

```
myapp/
├── go.mod
├── main.go              # HTTP uygulaması, registry, hub, adapter route'ları
├── index.html            # /client/goui.js'i yükler ve bağlanır
└── (isteğe bağlı) i18n/
    └── locales/
        ├── tr.json
        └── en.json
```

`main.go` dört şeyden sorumludur:

1. Bir `core.Registry` inşa etmek ve bileşen(ler)inizi isimle kaydetmek.
2. Bir `i18n.Translator` inşa etmek (isteğe bağlı olarak locale dosyalarını yükleyerek).
3. Bir `ws.Hub` inşa etmek, `ws.NewServer(hub, registry, translator)` ile
   sarmalamak ve seçtiğiniz adapter üzerinden `GET /goui/ws`'i kaydetmek
   (bkz. §2).
4. GoUI istemci çalışma zamanını (`client/`) ve sayfanızın `index.html`'ini statik dosyalar olarak sunmak.

`client/` dizinini HTTP yığınınızın statik dosya sunma yöntemiyle servis edin —
Fiber için örnek:

```go
app.Use("/client", static.New("./client"))
```

## 4. İlk bileşeniniz

Her GoUI bileşeni `core.Component` arayüzünü uygular:

```go
type Component interface {
    Mount(ctx context.Context) error
    Render() (string, error)
    HandleEvent(ctx context.Context, event string, payload map[string]any) error
    Unmount(ctx context.Context) error
}
```

`core.BaseComponent`'i gömerek (embed) dirty-tracking, i18n ve toast
yardımcılarını bedava elde edersiniz (tam sözleşme için
[`02-components.md`](02-components.md)'ye bakın).

İşte kanonik "Counter" (Sayaç) bileşeni:

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

Ve buna eşlik eden `index.html`:

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

Bu sayfa yüklendiğinde neler olur:

1. `GoUIClient`, `/goui/ws?component=counter&locale=tr`'ye bir WebSocket açar.
2. Sunucu bir `ws.Session` oluşturur, `Registry`'den bir `Counter` inşa
   etmesini ister, `Mount`'u çağırır, ardından bir `session` frame'i ve
   onu takiben ilk `render` frame'ini gönderir.
3. `+`'a tıklamak bir `event` frame'i gönderir (`{"type":"event","event":"increment", ...}`);
   sunucu `HandleEvent`'i çağırır, yeniden render eder, eski/yeni HTML
   ağacını diff'ler ve minimal bir `render` yaması geri gönderir.
4. Sekme bağlantısı kesilirse (yeniden yükleme, ağ kesintisi), oturum bir
   grace period (varsayılan 60 sn, bkz.
   [`04-sessions-and-websocket.md`](04-sessions-and-websocket.md)) boyunca
   canlı tutulur; böylece yeniden bağlanma, baştan başlamak yerine durumu
   geri yükler.

## 5. Repoyla gelen örnekleri çalıştırma

Repo, `examples/` altında çalıştırılabilir on demo sunar; her biri repo
kökünden doğrudan `go run` edebileceğiniz bağımsız bir `main.go`'dur. Her
örnek kendi statik HTML sayfasını sunar, GoUI WebSocket route'unu bağlar ve
kendine özel bir portu dinler; böylece birkaçını yan yana çalıştırabilirsiniz:

| Port | Komut                                | Demo                          | Ne gösteriyor |
|------|-----------------------------------------|--------------------------------|---------------|
| 3000 | `go run ./examples/counter`             | **counter**                    | Minimal `Component` yaşam döngüsü, `g-click`, dirty tracking |
| 3001 | `go run ./examples/contact-form`        | **contact-form**               | Native form alanları, doğrulama, `Toast`/`ToastT`, prefetch → activate |
| 3002 | `go run ./examples/searchable-select`   | **searchable-select**          | Select ailesi: Searchable Select, Multi Select, Combobox, Autocomplete, Tag Input, Tree Select, Cascader, Dual Listbox |
| 3003 | `go run ./examples/numeric-controls`    | **numeric-controls**           | Currency Input, Percentage Input, Rating |
| 3004 | `go run ./examples/field-meta`          | **field-meta**                 | Karakter sayacı (`ShowCharCount`) ve parola gücü (`ShowStrength`) |
| 3005 | `go run ./examples/date-controls`       | **date-controls**               | Date Range Picker, Time Range Picker, Calendar Date Picker |
| 3006 | `go run ./examples/identity-inputs`     | **identity-inputs**            | OTP/PIN, Country/Language/Timezone/Currency Picker, Phone Input |
| 3007 | `go run ./examples/editors`             | **editors**                    | Markdown Editor (goldmark), Rich Text (Quill), Code Editor (CodeMirror) |
| 3008 | `go run ./examples/media-upload`        | **media-upload**               | Drag & Drop Upload, Image Upload, Avatar Upload + kırpma |
| 3009 | `go run ./examples/misc-controls`       | **misc-controls**              | Emoji/Icon/Font Picker, Swatch Color Picker, Gradient Picker, Mention, Signature Pad |
| 3010 | `go run ./examples/adapters/nethttp`    | **net/http adapter**           | Counter, düz `net/http` + `adapters/stdlib` |
| 3011 | `go run ./examples/adapters/chi`        | **Chi adapter**                | Counter, Chi üzerinde `adapters/stdlib` `Mount` |
| 3012 | `go run ./examples/adapters/gin`        | **Gin adapter**                | Counter, Gin üzerinde |
| 3013 | `go run ./examples/adapters/echo`       | **Echo adapter**               | Counter, Echo üzerinde |

Herhangi birini repo kökünden çalıştırın, ardından yazdırılan URL'yi açın:

```bash
go run ./examples/counter
# GoUI counter example at http://localhost:3000

go run ./examples/contact-form
# GoUI contact form at http://localhost:3001

go run ./examples/searchable-select
# GoUI searchable-select demo at http://localhost:3002
```

Her örnek kendi portunu dinlediğinden, birkaçını aynı anda ayrı terminallerde
başlatabilirsiniz — farklı form kontrollerini yan yana karşılaştırırken
faydalıdır. Her örnek, repo kökünü `runtime.Caller(0)` ile çözer; böylece
mevcut çalışma dizininiz ne olursa olsun çalışır ve şunları bağlar:

- `/client` → çatının JS çalışma zamanı (`client/`)
- `/forms` → `forms/style.css` ve ilgili statik varlıklar (kullanıldığı yerlerde)
- `/goui/ws` → WebSocket uç noktası (`ws.Path`, adapter tarafından bağlanır)
- `/goui/upload`, `/goui/files/:id` → dosya yükleme uç noktaları (media-upload, misc-controls; adapter `Store` seçeneği veya `upload.Mount` ile)

## 6. Sırada ne var

- [`02-components.md`](02-components.md) — tam `Component` / `BaseComponent` / `Registry` / şablon sözleşmesi
- [`03-i18n.md`](03-i18n.md) — çevirmen (translator) kurulumu ve locale dosyaları
- [`04-sessions-and-websocket.md`](04-sessions-and-websocket.md) — Session/Hub yaşam döngüsü ve tel protokolü
- [`05-forms-tier1.md`](05-forms-tier1.md) — her native form kontrolü
- [`06-validation.md`](06-validation.md) — sunucu tarafı doğrulama kuralları
- [`07-forms-tier2.md`](07-forms-tier2.md) — her zengin form kontrolü
- [`17-page-modes.md`](17-page-modes.md) — ModeLive / ModeSEO / ModeStatic (admin vs genel HTML)
</contents>
</invoke>
