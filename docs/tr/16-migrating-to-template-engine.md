# 16 — Template motoruna migrasyon

Bu rehber, satır içi `core.RenderTemplate` kullanan component’leri `.goui.html`
dosyalarına ve `core.ViewComponent` arayüzüne taşır.

**Geriye dönük uyumluluk:** `core.RenderTemplate` **kaldırılmıyor**. Mevcut
component’ler çalışmaya devam eder. Migrasyon zorunlu değildir.

## Ne zaman değer?

| `RenderTemplate`’te kalın | `.goui.html`’e geçin |
|---------------------------|----------------------|
| Tek satırlık küçük markup | Çok bölümlü sayfa / layout |
| Component’ler arası paylaşım yok | Partial / component / slot paylaşımı |
| Prototip / geçici demo | Sık düzenlenecek üretim UI |

## Adım adım

### 1. Önce

```go
func (c *Counter) Render() (string, error) {
    return core.RenderTemplate(`<div class="counter">
<span>{{.Count}}</span>
<button g-click="increment">+</button>
</div>`, c)
}
```

### 2. View’ı ayırın

`views/counter.goui.html`:

```html
<div class="counter">
  <span>{{ .Count }}</span>
  <button type="button" g-click="increment">+</button>
</div>
```

### 3. `View` (+ stub `Render`)

```go
func (c *Counter) View() string { return "counter" }

func (c *Counter) Render() (string, error) {
    return "", gouitemplate.ErrViewRenderDirect
}
```

`core.Component` için `Render` hâlâ gerekir. `template.Wrap`, `ViewComponent`
için stub’ı çağırmaz.

### 4. Registry bağlantısı

```go
tmplReg, err := gouitemplate.NewRegistry(gouitemplate.Config{
    Root: filepath.Join(exampleDir, "views"),
})
if err != nil {
    log.Fatal(err)
}
defer tmplReg.Close()

coreReg := core.NewRegistry()
_ = coreReg.Register("counter", gouitemplate.Wrap(tmplReg, func() core.Component {
    return &Counter{}
}))
```

### 5. Doğrulama

- Uygulamayı `go build` edin
- Event’leri deneyin; dirty tracking `MarkDirty` / `ResetDirty` ile sürer
  (`Wrap` başarılı view render sonrası dirty’yi temizler)

Referans: [`examples/counter-view`](../../examples/counter-view)
(değiştirilmemiş [`examples/counter`](../../examples/counter) yanında).

## Kontrol listesi

- [ ] View dosyası `Config.Root` altında; `View()` dot-path ile eşleşiyor
- [ ] Factory `template.Wrap` ile kayıtlı
- [ ] `@props` kullanıyorsanız prod’da `StrictProps`
- [ ] `WatchForChanges` yalnızca geliştirmede
