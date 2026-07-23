<p align="center">
  <img src="assets/goui-banner.png" alt="GoUI — Sunucu güdümlü Go UI — WebSocket ve SEO HTML">
</p>

**GoUI**, Go tabanlı sunucu-merkezli bir UI framework’üdür. Component’leri Go’da yazarsınız; durum sunucudadır, HTML sunucuda üretilir ve WebSocket üzerinden minimal DOM yamaları (patch) gönderilir — genel sayfalar için `ModeSEO` / `ModeStatic` ile ilk boyamada dolu HTML de sunulabilir. Tarayıcıda küçük bir vanilla JS runtime çalışır — React/Vue bundle’ı ve client tarafında component ağacı yoktur.

LiveView fikrinden ilham alır (kalıcı bağlantı üzerinden sunucu-otoriteli görünümler); framework-bağımsız çekirdek, HTTP adaptörleri (net/http, Chi, Fiber, Gin, Echo), keyed HTML diff motoru, kademeli form kütüphanesi ve Go `html/template` üzerine Blade benzeri `.goui.html` template motoru ile bağımsız bir implementasyondur.

GoUI’yi, hem domain hem UI için tek dil (Go) istediğinizde, form state’ini sunucuda tutmak istediğinizde (doğrulama hatasından sonra Laravel tarzı `old()` uğraşı olmadan) ve SPA toolchain’i olmadan etkileşimli sayfalar çıkarmak istediğinizde kullanın.

## Neden GoUI?

| Yaklaşım | Kazanç | Bedel |
|----------|--------|-------|
| React/Vue SPA | Zengin client UX, geniş ekosistem | Çift model, API yüzeyi, build zinciri |
| Klasik MPA | Basit HTML | Tam sayfa yenileme, form round-trip zorluğu |
| HTMX | Kademeli iyileştirme | Çoğunlukla istek/yanıt; karmaşık state sizde |
| **GoUI** | Go component’ler, canlı WS patch, isteğe bağlı SEO/static HTML | Etkileşimli oturum başına kalıcı WS |

**GoUI tercih edin:** iç araçlar, admin/ERP panelleri, çok adımlı formlar, domain’i zaten Go’da olan tenant uygulamaları — ve `ModeSEO` / `ModeStatic` ile **ilk boyaması gerçek HTML** olan genel sayfalar.

**Başka bir şey tercih edin:** offline-first mobil; uzun ömürlü WebSocket’in pahalı olduğu çok yüksek eşzamanlı ucuz sayfa trafiği; büyük client-component pazarına ihtiyaç duyan ekipler.

## Mimari

```
Tarayıcı (goui.js)
    │  event / prefetch / activate
    ▼
Session ──► Component.HandleEvent / Mount
    │
    ▼
Render HTML ──► Diff (eski ağaç → patch) ──► Frame(render)
    │
    ▼
Hub (oturumlar, grace reconnect, Broadcast)
```

1. Client `/goui/ws?component=…` adresine bağlanır
2. Sunucu `Session` oluşturur, component’i mount eder, `session` + ilk `render` (`OpReplace`) gönderir
3. Kullanıcı olayları `event` frame olur → `HandleEvent` → yeniden render → minimal patch
4. İsteğe bağlı: `prefetch` sessiz mount; `activate` canlıya alır ve render eder
5. Kopmalarda oturum **grace period** (varsayılan 60 sn) boyunca tutulur; reconnect state’i korur

## Özellikler

### Çekirdek
- `Component` yaşam döngüsü: `Mount` / `Render` / `HandleEvent` / `Unmount`
- `BaseComponent`: dirty tracking, i18n yardımcıları, toast yardımcıları
- `Registry` factory’leri, HTML template cache (`RenderTemplate`)

### i18n
- JSON locale’ler, nested-flat key’ler (`form.required_field`)
- Base locale `tr`’ye düşme, sonra `[[key]]` yer tutucu

### WebSocket / Session / Hub
- Framework-bağımsız `ws.Server` + iç içe HTTP adaptörleri
- Session id ile reconnect; grace period temizliği
- Frame’ler: `event`, `render`, `push`, `error`, `session`, `prefetch`, `activate`

### Diff motoru
- HTML parse → ağaç → patch (`replace`, `update_text`, `set_attr`, `remove_attr`, `insert`, `remove`, `move`)
- `data-key` ile keyed list diff (basit key-map; LCS değil)

### Client runtime
- Vanilla JS: patch uygulama, event delegation (`g-click`, `g-change`, `g-input`, `g-submit`)
- Modüller: toast, prefetch, selectable, calendar, otp, richtext, codeeditor, upload, avatar, signature

### Forms
TextInput, NumericInput, DateTimeInput, ChoiceInput (checkbox/radio), FileInput, ColorInput, HiddenInput, Textarea, Select/Option/Optgroup, Button, Form/Fieldset/Legend/Label, Datalist, Output, Meter, Progress, Searchable Select, Multi Select, Combobox, Autocomplete, Tag/Chips, Tree Select, Cascader, Dual Listbox, Currency, Percentage, Rating, Date Range, Time Range, Calendar Picker, OTP/PIN, Phone, Country/Language/Timezone/Currency picker’lar, Rich Text (Quill), Markdown (goldmark), Code Editor (CodeMirror), Drag&Drop / Image / Avatar upload + cropper overlay, Emoji/Icon/Font picker’lar, Color swatch / Gradient, Signature, Mention, Karakter sayacı, Şifre gücü

### Validation
`Required`, `MinLength`, `MaxLength`, `Pattern`, `Email`, `NumericRange`, `Custom` — sunucu tarafı; başarısız doğrulamadan sonra state component’te kalır

### Push / Toast
`Toast` / `ToastT`, `Hub.Broadcast`, kind: success / error / warning / info

### Prefetch
`data-goui-prefetch` + `data-goui-activate`; sessiz Mount; LRU üst sınırı 5; önceden render yok

### Sayfa modları (SEO)
- `ModeLive` (varsayılan), `ModeSEO` (SSR HTML + WS hydrate), `ModeStatic` (yalnız HTML)
- `Registry.RegisterPage`, `page.NewRenderer`, adapter `Routes` / `Page(...)`
- Title / description / Open Graph için `HeadProvider`
- Kılavuz: [docs/tr/17-page-modes.md](docs/tr/17-page-modes.md) · Örnek: [`examples/seo-pages`](examples/seo-pages)

## Template motoru (Blade benzeri)

Dosya tabanlı `.goui.html` view’lar: `@extends` / `@section` / `@yield`, `@include`,
`@component` / `@slot`, opt-in `@props` kontrolü ve isteğe bağlı hot reload —
bir kez native `html/template`’e derlenir (auto-escaping korunur).

- Kılavuz: [docs/tr/15-template-engine.md](docs/tr/15-template-engine.md)
- `RenderTemplate` migrasyonu: [docs/tr/16-migrating-to-template-engine.md](docs/tr/16-migrating-to-template-engine.md)
- Örnek: [`examples/counter-view`](examples/counter-view)

## Gereksinimler

- **Go** `1.25.0` (`go.mod`)
- Bir HTTP adaptörü: `adapters/stdlib` (net/http / Chi), `adapters/fiber`, `adapters/gin` veya `adapters/echo`
- Tarayıcı: WebSocket; mobilde prefetch için IntersectionObserver önerilir
- **Tailwind CLI** (opsiyonel) — yalnızca `forms/style.css` ötesinde utility CSS istiyorsanız

## Kurulum

```bash
go get github.com/zatrano/goui@latest
# bir adaptör seçin, örn.:
go get github.com/zatrano/goui/adapters/stdlib@latest
# veya: adapters/fiber | adapters/gin | adapters/echo
```

## HTTP adaptörleri

| Yığın | Modül | Mount yardımcısı |
|-------|--------|----------------|
| net/http | `adapters/stdlib` | `Register(mux, opts)` |
| Chi | `adapters/stdlib` | `Mount(router, opts)` |
| Fiber v3 | `adapters/fiber` | `Register(app, opts)` |
| Gin | `adapters/gin` | `Register(router, opts)` |
| Echo | `adapters/echo` | `Register(echo, opts)` |

## Hızlı başlangıç

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
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")) // kendi dizin yapınıza göre ayarlayın

	registry := core.NewRegistry()
	_ = registry.Register("counter", func() core.Component { return &Counter{} })

	mux := http.NewServeMux()
	mux.Handle("/client/", http.StripPrefix("/client/", http.FileServer(http.Dir(filepath.Join(root, "client")))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html") // /client/goui.js yükleyen HTML
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

Veya hazır demoyu çalıştırın:

```bash
go run ./examples/counter
# http://localhost:3000
```

## Klasör yapısı

| Yol | Rol |
|-----|-----|
| `core/` | Component sözleşmesi, registry, template cache |
| `i18n/` | Translator + locale JSON |
| `ws/` | Session, Hub, frame’ler, framework-bağımsız `Server` |
| `diff/` | HTML parse, keyed diff, patch’ler |
| `forms/` | Searchable select ve picker’lar dahil form kontrolleri |
| `validation/` | Rule yardımcıları |
| `upload/` | `Storage` + `LocalStore` + `net/http` handler |
| `adapters/` | İç içe modüller: stdlib, fiber, gin, echo |
| `client/` | Tarayıcı runtime + modüller |
| `examples/` | Çalıştırılabilir demolar (Fiber demoları + `examples/adapters/*`) |
| `docs/` | Tam kılavuzlar (`docs/en`, `docs/tr`) |

## Dökümantasyon

- İngilizce: [`docs/en/`](docs/en/)
- Türkçe: [`docs/tr/`](docs/tr/) · özet: [`README.tr.md`](README.tr.md)

Başlangıç: [Kurulum](docs/tr/01-getting-started.md), [Proje entegrasyonu](docs/tr/13-project-integration.md) ve [Template motoru](docs/tr/15-template-engine.md).

## Yol haritası / bilinen sınırlar

- Bazı gelişmiş form kontrolleri henüz yok (gelecek / isteğe bağlı).
- **Upload** şu an yalnızca `LocalStore`; `upload.Storage` S3/MinIO için hazır.
- Diff tipik admin UI’lar için; dinamik listelerde mutlaka `data-key` kullanın.
- Prefetch yalnızca mount eder (bilinçli olarak HTML önceden gönderilmez).

## Lisans

MIT taslağı — [`LICENSE`](LICENSE). Genel yayın öncesi onaylayın.

## Katkı

[`CONTRIBUTING.md`](CONTRIBUTING.md) ve [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).

## Değişiklik günlüğü

[`CHANGELOG.md`](CHANGELOG.md).

## İletişim

Proje: [github.com/zatrano/goui](https://github.com/zatrano/goui)
Depo yayınlandıktan sonra issue ve PR’lar memnuniyetle karşılanır.
