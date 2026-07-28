# 02 — Bileşenler, BaseComponent, Registry, Şablonlar

Bu sayfa, `core` paketini (`github.com/zatrano/goui/core`) baştan sona
belgeler: `Component` sözleşmesi, her `BaseComponent` alanı ve metodu,
`Registry`, ve HTML şablon önbelleği.

```go
import "github.com/zatrano/goui/core"
```

## 1. `Component` arayüzü

```go
type Component interface {
    Mount(ctx context.Context) error
    Render() (string, error)
    HandleEvent(ctx context.Context, event string, payload map[string]any) error
    Unmount(ctx context.Context) error
}
```

İki satırlık bir sayaçtan tam bir forma kadar her GoUI görünümü bu dört
metodu uygular. `core.BaseComponent` (aşağıda) paylaşılan durum ve
yardımcılar sağlar ama kasıtlı olarak `Component`'i kendisi **uygulamaz**:
her zaman kendi `Render`, `HandleEvent` ve yaşam döngüsü metotlarınızı
yazarsınız (veya bir Tier 1/Tier 2 alanının kendi uygulamasına delege
edersiniz).

### `Mount(ctx context.Context) error`

- **Ne zaman çağrılır:** tam olarak bir kez, bileşen `Registry` tarafından
  (veya elle inşa edilerek) örneklendirildikten hemen sonra ve ilk render'dan
  önce. `ws.Session.MountComponent`, `Session.Prefetch`, ve
  `Session.Activate` (fresh path) hepsi bir bileşeni oturum için hazırlamanın
  parçası olarak `Mount`'u çağırır.
- **Yapın:** çalışma zamanı bağlamına (runtime context) bağlı olan herhangi
  bir durumu başlatın — örn. `TreeSelect.Mount` `Expanded` map'ini tembel
  (lazily) tahsis eder, `Cascader.Mount` `Levels[0]`'ı `Items`'tan besler,
  `MarkdownEditor.Mount` ilk önizleme HTML'ini render eder.
- **Yapmayın:** ilk render'ı burada yapmayın — render, oturum tarafından
  yürütülen ayrı bir adımdır (`sendFullRender`) ve çoğu Tier 1 alanı basitçe
  `return nil` yapar.
- Nil olmayan bir hata döndürmek oturum kurulumunu iptal eder (WebSocket
  handler bir `error` frame'i yazar ve ilerlemez).

### `Render() (string, error)`

- **Ne zaman çağrılır:** `Mount`'tan sonra ilk render için, ve
  `core.ErrSkipRender` döndürmeyen her `HandleEvent` çağrısından sonra
  tekrar. Oturum, önceki render edilmiş ağacı yenisiyle diff'ler ve sadece
  değişen yamaları gönderir (bkz.
  [`04-sessions-and-websocket.md`](04-sessions-and-websocket.md)).
- **Yapın:** iyi biçimlendirilmiş tek bir HTML parçası döndürün (tek bir kök
  eleman şiddetle önerilir — oturum, çoklu-kök çıktıyı ek bir `<div>` içine
  sarar, böylece `data-goui-component`'in yaşayacağı bir yer olur). Taze çıktı
  ürettikten sonra, dirtiness takip ediyorsanız `ResetDirty()`'yi çağırın.
  `core.RenderTemplate`'i (veya `forms.Attrs` aracılığıyla elle inşa edilmiş
  string'leri) kullanın ve tüm çevrilebilir metni `T` üzerinden yönlendirin.
- **Yapmayın:** `Render` içinde bileşen durumunu mutasyona uğratmayın —
  durum değişiklikleri `HandleEvent`/`Mount`'a aittir; `Render`, mevcut
  alanların saf (pure) bir yansıması olmalıdır.
- `Render`, değişmemiş durumla **birden fazla çağrılmaya karşı güvenli
  olmalıdır** — oturum onu olay başına bir kez ve yeniden bağlanmada bir kez
  daha çağırır (`SendInitialRenders`).

### `HandleEvent(ctx context.Context, event string, payload map[string]any) error`

- **Ne zaman çağrılır:** tarayıcıdan gelen her `event` frame'i için bir kez
  (`g-click`, `g-change`, debounce'tan sonra `g-input`, `g-submit`).
  `event`, `z-*` özniteliğinden gelen string'tir; `payload`,
  `collectPayload`/`collectFormPayload`'ın istemci tarafında topladığı her
  neyse odur (tipik olarak `{"value": "..."}`, `{"checked": true, "value": "..."}`,
  veya form submit'leri için `{"fields": {...}}`).
- **Yapın:** `event`/`payload`'a göre bileşeninizin alanlarını mutasyona
  uğratın, dirtiness'i elle takip ediyorsanız `MarkDirty()`'yi çağırın, ve
  UI'si istemci tarafına ait olan olaylar için (rich text, kod editörleri —
  bkz. [`07-forms-tier2.md`](07-forms-tier2.md)) `core.ErrSkipRender`
  döndürün.
- **Yapmayın:** `Render()`'ı kendiniz çağırmayın — oturum bunu
  `HandleEvent` döndükten hemen sonra yapar (eğer `ErrSkipRender`
  döndürmediyseniz).
- Başka herhangi bir nil olmayan hata döndürmek, oturumun istemciye bir
  `render` frame'i yerine bir `error` frame'i göndermesine sebep olur.

### `Unmount(ctx context.Context) error`

- **Ne zaman çağrılır:** oturum kapandığında (`Session.Close`, örn. sekme
  kapatıldığında ve grace period sona erdiğinde, veya sunucu hub'ı
  kapattığında) — hâlâ takip edilen **her** bileşen için, hem aktif hem de
  prefetch edilmiş-ama-aktive-edilmemiş olanlar dahil. Ayrıca, yinelenen bir
  prefetch eviction'ına yarışı kaybeden prefetch edilmiş bir bileşende de
  anında çağrılır.
- **Yapın:** `Mount`'ta açtığınız kaynakları serbest bırakın (dosya
  handle'ları, zamanlayıcılar, abonelikler). Çoğu Tier 1/Tier 2 alanının
  serbest bırakacak hiçbir şeyi yoktur ve `return nil` yapar.
- **Yapmayın:** `Unmount`'un her sayfa navigasyonunda çalıştığını
  varsaymayın — sadece *oturum* söküldüğünde çalışır, her yeniden
  render'da değil.

## 2. `BaseComponent`

```go
type BaseComponent struct {
    ID       string
    Children map[string]Component
    // dirty bool // private

    Locale     string
    // translator, pusher are private
}
```

İlk alan olarak değer (by value) türünde gömün:

```go
type MyComponent struct {
    core.BaseComponent
    // ... kendi alanlarınız
}
```

### Alanlar

| Alan | Tür | Amaç |
|---|---|---|
| `ID` | `string` | Oturum tarafından atanan bileşen örneği ID'si. Bileşen mount/activate edildiğinde `ws.Session` tarafından reflection ile otomatik ayarlanır (`BaseComponent` adlı bir alan arar ve onun `ID`/`Locale` alt alanlarını ayarlar) — normalde bunu kendiniz asla ayarlamazsınız. |
| `Children` | `map[string]Component` | Alt bileşenleri isimle kompoze etmek için isteğe bağlı bir slot. `core` paketi bunu otomatik olarak doldurmaz veya kullanmaz; iç içe bileşenleri elle yöneten bileşenler için bir konvansiyon olarak sağlanır. |
| `Locale` | `string` | Bu bileşen örneği için aktif locale (örn. `"tr"`, `"en"`); oturum, bileşeni mount/activate ettiğinde WebSocket `?locale=` sorgu parametresinden ayarlanır. `T`/`ToastT` tarafından okunur. |

### Metotlar

| Metot | İmza | Davranış |
|---|---|---|
| `MarkDirty` | `func (b *BaseComponent) MarkDirty()` | Dahili dirty bayrağını `true`'ya ayarlar. Bir değişikliğin yeniden render'ı tetiklemesi gerektiğinde `HandleEvent`'ten çağırın. |
| `IsDirty` | `func (b *BaseComponent) IsDirty() bool` | Mevcut dirty bayrağını raporlar. Kendi `Render`/dispatch mantığınız hiçbir şey değişmediğinde işi atlamak isterse yararlıdır — `ws.Session`'ın kendisi, bu bayraktan bağımsız olarak başarılı bir `HandleEvent`'ten sonra her zaman `Render`'ı çağırır, dolayısıyla `IsDirty`/`ResetDirty` kendi kodunuz için bir konvansiyondur, oturum tarafından zorunlu kılınmaz. |
| `ResetDirty` | `func (b *BaseComponent) ResetDirty()` | Dirty bayrağını temizler. Güncel HTML ürettiğinizde `Render`'ın sonunda çağırın (bkz. [`01-getting-started.md`](01-getting-started.md)'deki `Counter` örneği). |
| `SetTranslator` | `func (b *BaseComponent) SetTranslator(t *i18n.Translator)` | Paylaşılan `*i18n.Translator`'ı enjekte eder. Bir bileşen mount veya activate edildiğinde `ws.Session` tarafından otomatik olarak çağrılır (bir `interface{ SetTranslator(*i18n.Translator) }` type assertion aracılığıyla). Kendiniz inşa ettiğiniz alt alanlar için de bunu elle çağırırsınız (bkz. [`01-getting-started.md`](01-getting-started.md) ve [`06-validation.md`](06-validation.md)'deki `ContactForm` örneği), çünkü oturum yalnızca üst seviye bileşenin kendi `BaseComponent`'ine reflection yapar. |
| `SetPusher` | `func (b *BaseComponent) SetPusher(fn func(kind, text string))` | `Toast`/`ToastT` tarafından kullanılan bir callback enjekte eder. `ws.Session.injectPusher` bunu otomatik olarak `Session.EnqueuePush`'a bağlar, böylece push'lar tel üzerinde `push` frame'leri olarak sonuçlanır. |
| `Toast` | `func (b *BaseComponent) Toast(kind, text string)` | Enjekte edilmiş pusher aracılığıyla bir push bildirimi (`kind` + zaten çevrilmiş `text`) gönderir. Enjekte edilmiş bir pusher yoksa hiçbir şey yapmaz (panic atmaz) — testlerde veya bağımsız bileşenlerde çağırmak güvenlidir. |
| `ToastT` | `func (b *BaseComponent) ToastT(kind, key string, args ...any)` | Önce `key`'i `T` ile çevirir, ardından sonuçla `Toast`'ı çağırır. Kullanıcıya yönelik bildirimler için bunu kullanın, böylece `Locale`'e uyarlar. |
| `T` | `func (b *BaseComponent) T(key string, args ...any) string` | Enjekte edilmiş çevirmeni kullanarak `key`'i `b.Locale` için çevirir. `Locale` boşsa, `i18n.BaseLocale`'e (`"tr"`) geri döner. Hiçbir çevirmen enjekte edilmemişse, ham key'i `"[[" + key + "]]"` şeklinde sarılmış olarak döndürür — bu, eksik bağlantıyı sessizce boş bir string olarak render etmek yerine render edilmiş HTML'de açıkça görünür kılar. Tam arama/fallback algoritması için [`03-i18n.md`](03-i18n.md)'ye bakın. |

### Pratikte `T` ve `ToastT`

```go
func (c *MyComponent) HandleEvent(_ context.Context, event string, _ map[string]any) error {
    if event == "save" {
        c.ToastT("success", "contact.submit_success") // çevrilmiş + gönderilmiş
    }
    return nil
}

func (c *MyComponent) Render() (string, error) {
    label := c.T("form.submit") // "Gönder" (tr) veya "Submit" (en)
    return "<button>" + html.EscapeString(label) + "</button>", nil
}
```

## 3. `Registry`

```go
type Registry struct { /* ... */ }

func NewRegistry() *Registry
func (r *Registry) Register(name string, factory func() Component) error
func (r *Registry) Create(name string) (Component, error)
```

Registry, bir **string ismi** (WebSocket URL'sindeki `?component=name`
değeri, veya `Session.Prefetch`/`Activate`'e verilen değer) taze bir
`Component` örneği döndüren bir **factory (üretici) fonksiyona** eşler. Eşzamanlı
kullanım için güvenlidir (dahili bir `sync.RWMutex` ile korunur).

### `Register`

```go
registry := core.NewRegistry()
err := registry.Register("counter", func() core.Component { return &Counter{} })
```

- `name` → `factory` kaydeder.
- **Hata:** `name` zaten kayıtlıysa `core.ErrComponentAlreadyRegistered`
  döndürür. Kayıt genellikle başlangıçta bir kez yapılır, dolayısıyla çoğu
  kod basitçe `if err := registry.Register(...); err != nil { log.Fatal(err) }`
  yapar.

### `Create`

```go
c, err := registry.Create("counter") // c taze bir *Counter'dır
```

- `name`'i arar ve her seferinde tamamen yeni bir `Component` örneği
  döndürerek factory'sini çağırır (factory closure'ı yeni bir struct tahsis
  etmekten sorumludur — registry'ler örnekleri oturumlar arasında asla
  paylaşmaz).
- **Hata:** `name` hiç kaydedilmediyse `core.ErrComponentNotRegistered`
  döndürür. Bir istemci `?component=typo` ile bağlandığında, veya bilinmeyen
  bir isimle `Prefetch`/`Activate`'i çağırdığında bir `error` frame'i olarak
  ortaya çıkan hata budur.

Her iki sentinel hata da `core/errors.go`'da tanımlanır:

```go
var (
    ErrComponentNotRegistered     = errors.New("component not registered")
    ErrComponentAlreadyRegistered = errors.New("component already registered")
)
```

## 4. Şablon önbelleği: `core.RenderTemplate`

```go
func RenderTemplate(tmplStr string, data any) (string, error)
```

`RenderTemplate`, Go'nun `html/template`'ini sarar, ama **ayrıştırılmış
şablonu**, şablon *string*'inin kendisinin (bileşen veya dosya değil) bir
FNV-64a hash'i ile önbelleğe alır. Belirli bir şablon string'iyle yapılan
ilk çağrı, ayrıştırma (parse) maliyetini öder; herhangi bir bileşenden,
herhangi bir oturumdan gelen aynı string ile yapılan sonraki her çağrı,
önbelleğe alınmış `*template.Template`'i yeniden kullanır ve sadece
`Execute` maliyetini öder. Bu nedenle şablonları doğrudan `Render()` içinde
string literal'ları olarak yazmak idiomatiktir: string, çağrılar arasında
aynıdır, dolayısıyla önbellek ilk çağrıdan sonra her seferinde isabet eder.

```go
func (c *Counter) Render() (string, error) {
    html, err := core.RenderTemplate(`<span>{{.Count}}</span>`, c)
    if err != nil {
        return "", err
    }
    c.ResetDirty()
    return html, nil
}
```

### `{{call .T "key"}}` kuralı

`RenderTemplate`, `html/template`'i kullandığından, koşullar içinde veya
alıcıyı (receiver) açıkça geçirirken piped değer üzerinde doğrudan bir metot
çağıramazsınız — bunun yerine `T` **fonksiyon değerini** veri map'inde
geçirirsiniz (veya `.` ile erişilebilir bir alan/metot olmasına güvenirsiniz)
ve onu `{{call .T "key"}}` ile çağırırsınız. İki desteklenen form:

```go
// 1. Tüm bileşeni geçirin (T, BaseComponent üzerinde bir metot değeridir,
//    bu yüzden {{.T "key"}} de çalışır, ama {{call .T "key" .}}, anahtarın
//    yanında yer tutucu (placeholder) veri geçirmeniz gerektiğinde güvenli
//    genel formdur):
core.RenderTemplate(`<p>{{call .T "welcome_message" .}}</p>`, c)
// şablonun üst seviye verisinin hem bir .T metoduna hem de çeviri
// string'i tarafından referans verilen alanlara (örn. "{{.Name}}" için .Name)
// sahip olmasını gerektirir.

// 2. Bileşenin kendisi uygun üst seviye veri değeri olmadığında, T'yi
//    açıkça içeren bir map geçirin:
data := map[string]any{"Name": "GoUI", "T": bc.T}
core.RenderTemplate(`<p>{{call .T "welcome_message" .}}</p>`, data)

// Yer tutucu olmadan, sondaki "." argümanını atlayabilirsiniz:
core.RenderTemplate(`<button>{{call .T "form.submit"}}</button>`, map[string]any{"T": bc.T})
```

Genel kurallar:

- `.T`, `func(key string, args ...any) string` şeklinde bir fonksiyona
  çözülmelidir — `BaseComponent.T` bunu tam olarak eşleştirir, dolayısıyla
  `core.BaseComponent`'i gömmek ve `c`'yi (veya
  `map[string]any{"T": c.T, ...}`'yi) şablon verisi olarak geçirmek yeterlidir.
- Çeviri string'i `{{.Name}}` gibi Go şablon yer tutucuları içerdiğinde
  `{{call .T "key" .}}`'yi (tüm veri değerini tek `args` elemanı olarak
  geçirerek) kullanın — çevirmen, çevrilmiş string'i yeniden bir şablon
  olarak ayrıştırır ve o tek argümana karşı yürütür.
- Yer tutucusu olmayan düz string'ler için `{{call .T "key"}}`'yi (sondaki
  argüman olmadan) kullanın.
- `html/template`, çevrilmiş string'i HTML olarak otomatik olarak
  escape'ler, dolayısıyla `RenderTemplate` üzerinden giderken
  `html.EscapeString`'i kendiniz çağırmanıza gerek yoktur — ama string
  birleştirme ile elle HTML inşa ederken (çoğu `forms` paketindeki Tier
  1/Tier 2 alanının standart kütüphanenin `html.EscapeString`'i aracılığıyla
  yaptığı gibi) elle escape etmeniz **gerekir**.

`Translate`/`T`'nin key'leri nasıl çözdüğü için [`03-i18n.md`](03-i18n.md)'ye,
ve yukarıdaki her iki formun çalıştırılabilir örnekleri için
[`component_i18n_test.go`](../../core/component_i18n_test.go)'ya bakın.
</contents>
