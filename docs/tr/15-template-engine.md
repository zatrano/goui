# 15 — Template motoru

GoUI, Go’nun native `html/template` katmanı üzerine Blade benzeri bir template
motoru sunar. `.goui.html` dosyalarına tanıdık direktifler yazarsınız (`@if`,
`@extends`, `@component`, …); uygulama açılışında bunlar bir kez native Go
template’e çevrilir ve bellekte tutulur. Runtime’da string eval **yoktur**;
Go ifade dili yeniden yazılmaz — `{{ .Field }}` pipeline’ları olduğu gibi
kopyalanır, böylece context-aware auto-escaping korunur.

Ayrıca: [`RenderTemplate` migrasyonu](16-migrating-to-template-engine.md),
örnek [`examples/counter-view`](../../examples/counter-view).

## 1. Felsefe

Motor bir **yapısal önişlemcidir**:

| GoUI’nin işi | `html/template`’e bırakılan |
|--------------|-------------------------------|
| `@if` / `@foreach` / `@extends` / `@include` / `@component` | `{{ .Field }}`, pipeline, `eq`, fonksiyonlar |
| Dizin → dot-path adları | Auto-escaping |
| Derleme zamanı bağımlılık grafiği | Çalıştırma |

## 2. Kurulum

- Uzantı: `.goui.html`
- Dot-path: `views/pages/home.goui.html` → `"pages.home"` (`Root`’a göre)

```go
import gouitemplate "github.com/zatrano/goui/template"

reg, err := gouitemplate.NewRegistry(gouitemplate.Config{
    Root:        "./views",
    StrictProps: true, // prod’da önerilir
})
if err != nil {
    log.Fatal(err)
}
defer reg.Close()

html, err := reg.Render("pages.home", data)
```

## 3. Direktif referansı

### Koşullar

```html
@if(.User.IsAdmin)
  <span>Admin</span>
@elseif(.User.Moderator)
  <span>Mod</span>
@else
  <span>User</span>
@endif

@unless(.Hidden)
  görünür
@endunless
```

### Döngüler

Her zaman `$` ile isimlendirin (`.` yeniden bağlamaya güvenmeyin):

```html
@foreach(.Items as $item)
  <li>{{ $item.Name }}</li>
@empty
  <li>Yok</li>
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
  Diğer
@endswitch
```

### Çıktı

```html
{{ .Name }}           <!-- escape -->
{!! .TrustedHTML !!}  <!-- ham; Güvenlik bölümüne bakın -->
{{-- yorum --}}
@@literal             <!-- tek @ yazar -->
```

### Yardımcılar (`BaseFuncMap`)

```html
{{ default "Guest" .User.Name }}
{{ dict "Type" "submit" "Label" "Save" }}
{{ list 1 2 3 }}
```

## 4. Layout (`@extends` / `@section` / `@yield`)

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
  <h1>Hoş geldiniz</h1>
@endsection
```

`@extends` varken kök seviyede yalnızca `@section` (ve boşluk) serbesttir.

## 5. Include

```html
@include("partials.nav")
@include("partials.user", .User)
@includeIf("partials.optional")  <!-- yoksa derleme zamanında atlanır -->
```

Eksik `@include` hedefi `NewRegistry`’yi düşürür. `@includeIf` düşürmez.

## 6. Component ve slot

`components/card.goui.html`:

```html
@props(Title string)
<div class="card">
  @if(.Slots.header)
    <header>{{ .Slots.header }}</header>
  @endif
  <div>{{ .DefaultSlot }}</div>
</div>
```

Çağıran:

```html
@component("components.card", dict "Title" "Hi")
  @slot("header")
    {{ .PageTitle }}
  @endslot
  Varsayılan gövde
@endcomponent
```

Component içinde `.` bir `Dot`’tur: `.Props.*`, `.Slots.name`, `.DefaultSlot`.
Slot gövdeleri **çağıranın** veri bağlamında render edilir.

İç içe component desteklenir.

## 7. `@props` ve `StrictProps`

```html
@props(Name string, Count int = 0)
```

`StrictProps: true` iken `NewRegistry`, dosyadaki her `.Props.X` kullanımının
beyan edildiğini kontrol eder (yazım hataları “did you mean …?” ile açılışta
yakalanır). Kullanılmayan beyanlar `reg.Warnings()` ile yumuşak uyarıdır.

`StrictProps` false (varsayılan) ise bu kontroller çalışmaz.

## 8. Hot reload (geliştirme)

```go
hub := ws.NewHub()
reg, err := gouitemplate.NewRegistry(gouitemplate.Config{
    Root:            "./views",
    WatchForChanges: true, // prod’da false
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

`template` paketi `ws` import **etmez**; callback’i siz bağlarsınız.
Başarısız reload son iyi derlemeyi korur.

## 9. `ViewComponent` entegrasyonu

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

Örnek: `examples/counter-view`.

## 10. Güvenlik

- Tercihen `{{ }}` (escape’li).
- `{!! !!}` / `raw` auto-escape’i kapatır — yalnızca güvenilir HTML.
- Kullanıcı girdisini sanitize etmeden raw’dan geçirmeyin.

## 11. Blade karşılaştırması

| Blade | GoUI |
|-------|------|
| `@if` / `@foreach` / `@extends` | Aynı fikir → native `html/template` |
| `@component` / `@slot` | Desteklenir (iki aşamalı render) |
| `@includeIf` | Derleme zamanı |
| `@props` | Opt-in isim kontrolü (`StrictProps`) |
| `@php` / keyfi PHP | **Yok (bilinçli tasarım)** — template’de keyfi kod yok |
| İstek başına mtime cache | Process ömrü boyunca bellek içi derleme |

`@php` tarzı kaçışların olmaması kasıtlıdır: template veri + yapı, mantık Go’da kalır.

## Performans

Tipik bir dizüstünde mertebe (`go test ./template/ -bench=.`):

| İşlem | Kabaca maliyet |
|-------|----------------|
| `Render` düz sayfa | düşük µs / op |
| `Render` + extends | düşük–orta µs / op |
| `Render` ~20 component | onlarca µs / op |
| `NewRegistry` ~100 dosya | onlarca–yüzlerce ms |

Açılışta bir kez derleyin (veya hot reload); `Render` disk I/O yapmaz.
