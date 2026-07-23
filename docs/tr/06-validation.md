# 06 — Doğrulama (Validation)

GoUI'nin doğrulaması, birlikte çalışan iki paketin içinde yaşar:

- **`github.com/zatrano/goui/validation`** — saf, durumsuz (stateless)
  `Rule` fonksiyonları ve `Validate` çalıştırıcısı.
- **`github.com/zatrano/goui/forms`** — `FieldValidation` (her Tier
  1/Tier 2 alanına gömülü) ve `ValidateAll`; bunlar kuralları render
  hattına bağlar (hata mesajları, `aria-invalid`, hata CSS sınıfı) ve
  başarısızlıkta gönderilen değerleri korur.

```go
import (
    "github.com/zatrano/goui/forms"
    "github.com/zatrano/goui/validation"
)
```

## 1. `validation.Rule`

```go
type Rule func(value string) (ok bool, messageKey string)
```

Bir kural, alanın mevcut **string** değerini alır ve geçip geçmediğini,
artı geçmediğinde kullanılacak bir i18n mesaj key'ini döndürür. Aşağıdaki
tüm kural yapıcıları (constructor) bir `Rule` closure'ı döndürür.

### `Required()`

Değer boş veya sadece boşluksa başarısız olur.

```go
validation.Required() // → başarısızlıkta "validation.required"
```

### `MinLength(n int)`

**Rune** uzunluğu (byte uzunluğu değil — Türkçe'deki `ğ, ü, ş, ö, ç, ı, İ`
gibi çok baytlı UTF-8 için güvenli) `n`'nin altındaysa başarısız olur.

```go
validation.MinLength(3) // → "validation.min_length"
```

### `MaxLength(n int)`

Rune uzunluğu `n`'nin üzerindeyse başarısız olur.

```go
validation.MaxLength(120) // → "validation.max_length"
```

### `Pattern(regex string)`

Değer verilen düzenli ifadeyle eşleşmiyorsa başarısız olur. `regex`'in
kendisi derlenemezse, döndürülen kural **her zaman başarısız olur**
(açık değil, kapalı başarısızlık) `"validation.pattern"` ile.

```go
validation.Pattern(`^[A-Z]{2}\d{4}$`) // → "validation.pattern"
```

### `Email()`

Değer basit bir `local@domain.tld` şekliyle eşleşmiyorsa başarısız olur
(`^[^@\s]+@[^@\s]+\.[^@\s]+$`) — kasıtlı olarak esnektir, tam bir RFC
5322 doğrulayıcısı değildir.

```go
validation.Email() // → "validation.email"
```

### `NumericRange(min, max float64)`

Değeri bir `float64` olarak ayrıştırır ve ayrıştırılamıyorsa, veya
`[min, max]` (dahil) dışında ayrıştırılıyorsa başarısız olur.

```go
validation.NumericRange(0, 100) // → "validation.numeric_range"
```

### `Custom(fn func(value string) bool, messageKey string)`

Rastgele bir yükleneni (predicate) kendi mesaj key'inizle sarar — yerleşik
kuralların kapsamadığı her şey için kaçış kapısı (uniqueness kontrolleri,
`Validate` üzerine katmanlanmış çapraz alan kuralları, işe özgü formatlar
vb.).

```go
validation.Custom(func(v string) bool {
    return strings.HasPrefix(v, "TR")
}, "validation.custom")
```

## 2. Kuralları çalıştırma: `Validate` ve `ValidateAll`

### `validation.Validate`

```go
func Validate(value string, rules ...Rule) []string
```

`value`'ya karşı **her** kuralı çalıştırır — ilk başarısızlıkta
**durmaz** — ve başarısız olan her kuralın mesaj key'lerini sırayla
döndürür. Dilimdeki `nil` kurallar güvenli bir şekilde atlanır.

```go
keys := validation.Validate("ab", validation.Required(), validation.MinLength(3), validation.Email())
// keys == ["validation.min_length", "validation.email"]
```

### `forms.ValidateAll`

```go
func ValidateAll(fields ...Validatable) bool
```

`Validatable` şudur:

```go
type Validatable interface {
    Validate() bool
}
```

Her Tier 1 ve Tier 2 alan türü, `Validate() bool`'u kendisi uygular
(dahili olarak gömülü `FieldValidation.Run`'a delege ederek), dolayısıyla
kontrol edilmesini istediğiniz alanların pointer'larını geçirirsiniz:

```go
ok := forms.ValidateAll(&c.Name, &c.Email, &c.Country, &c.Message, &c.Subscribe)
```

`ValidateAll` de — `validation.Validate` gibi — kısa devre yapmaz;
geçirilen **her** alanda `Validate()`'i çağırır (böylece sadece ilk
başarısız olanın değil, her alanın `Errors` diliminin doldurulur) ve
hepsi geçtiyse `true` döndürür. Listedeki `nil` girdiler atlanır.

### `forms.FieldValidation`

```go
type FieldValidation struct {
    Rules  []validation.Rule
    Errors []string
}
```

Bunu kendi alan-benzeri türlerinize gömerek doğrulamayı bedava elde
edin:

```go
type MyField struct {
    core.BaseComponent
    forms.CommonAttrs
    forms.FieldValidation
    Value string
}

func (f *MyField) Validate() bool {
    return f.FieldValidation.run(f.Value, f.T) // FieldValidation üzerinde paket-özel yardımcı
}
```

(`forms`/`forms` paketlerinin içindeki Tier 1/Tier 2 alanları
private `run`/exported `Run` varyantını doğrudan çağırır; bu paketlerin
dışından exported `Run`'ı kullanın:)

```go
func (f *MyField) Validate() bool {
    return f.FieldValidation.Run(f.Value, f.T)
}
```

`Run`/`run` üç şey yapar:

1. Başarısız olan mesaj key'lerini almak için
   `validation.Validate(value, f.Rules...)`'u çağırır.
2. Geçirdiğiniz `translate` fonksiyonu aracılığıyla (tipik olarak `f.T`,
   böylece hata metni bileşenin `Locale`'ine uyar) her key'i
   `f.Errors []string`'e çevirir. `translate` `nil`'se, hatalar ham
   `"[[key]]"` biçimine geri döner.
3. Başarısız key sayısı sıfırsa yalnızca `true` döndürür.

Tier 1/Tier 2 `Render()` uygulamalarının doğrudan çağırdığı
`FieldValidation` üzerindeki iki dışa açık (exported) yardımcı daha
vardır:

```go
func (f *FieldValidation) ApplyErrorState(attrs Attrs, baseClass string) Attrs
func (f *FieldValidation) ErrorsHTML() string
```

- **`ApplyErrorState`** — `Errors` boşsa, `baseClass`'ın `class` olarak
  uygulandığından emin olur (çağıran özel bir tane ayarlamadıysa) ve
  aksi halde `attrs`'ı değiştirmeden döndürür. `Errors` boş değilse,
  `aria-invalid="true"`'yu ayarlar ve `border-goui-error` CSS sınıfını
  ekler, mevcut herhangi bir `class` değeriyle birleştirir.
- **`ErrorsHTML`** — `Errors`'daki her girdiyi
  `<p class="goui-field-error text-goui-error text-sm">...</p>` olarak
  (HTML-escape'lenmiş) render eder, birleştirilmiş halde. Her Tier
  1/Tier 2 alanının `Render()`'ı, kontrolün kendi markup'ından hemen
  sonra bu çağrının sonucunu ekler.

## 3. Başarısızlıkta korunan durum — `old()` cambazlığı yok

Klasik bir PHP/Laravel tarzı istek/yanıt (request/response) formunda,
başarısız bir doğrulama, tüm sayfanın yeniden yüklenmesi anlamına gelir
ve her alanı `old('field')`'dan elle yeniden doldurmanız ve alan adına
göre anahtarlanmış hata torbalarını (error bags) render etmeniz gerekir,
yoksa kullanıcı yazdığı her şeyi kaybeder.

GoUI'de bu problem **yapısal olarak** yoktur, çünkü bileşen *durumun
kendisidir* — kaybedilecek bir istek/yanıt döngüsü yoktur. Bir
`save`/`submit` olayı doğrulamada başarısız olduğunda:

1. `HandleEvent`, `forms.ValidateAll(...)`'u çağırır.
2. `ValidateAll` `false` döndürür. Her alanın kendi `Value`'su (veya
   `Checked`, `Values` vb.) **zaten** kullanıcının son yazdığı her
   neyse odur — bu, kullanıcı onunla etkileşime girdikçe her alanın
   kendi `HandleEvent`'i (`g-change`/`g-input`) tarafından artımlı
   olarak ayarlanmıştır, submit olayı hiç ateşlenmeden çok önce.
3. Her alanın `Errors []string`'i şimdi doldurulmuştur (adım 1'in
   `Validate()` çağrılarından).
4. `MarkDirty()`'yi çağırırsınız ve `nil` (veya kendi hatanızı)
   döndürürsünüz — **hiçbir** değerin herhangi bir yere geri
   kopyalanması gerekmez.
5. Sonraki `Render()`, kullanıcının yazdığı tam olarak aynı değerleri
   gösterir, şimdi hemen ardından render edilen kendi satır içi
   hata(ları)yla, ve `ApplyErrorState` tarafından otomatik olarak
   uygulanan `aria-invalid`/hata-sınırı stiliyle.

Ayrı bir "flash" deposu, `old()` yardımcısı, veya form girdisini
kaybeden bayat/süresi dolmuş bir oturum riski yoktur — değer, WebSocket
oturumunun tüm yaşam döngüsü boyunca aynı Go struct'ında yaşar (ve
grace period içinde yeniden bağlanmalara hayatta kalır, bkz.
[`04-sessions-and-websocket.md`](04-sessions-and-websocket.md)).

## 4. Tam form örneği

```go
package main

import (
    "context"
    "html"

    "github.com/zatrano/goui/core"
    "github.com/zatrano/goui/forms"
    "github.com/zatrano/goui/i18n"
    "github.com/zatrano/goui/validation"
)

type ContactForm struct {
    core.BaseComponent
    Name      forms.TextInput
    Email     forms.TextInput
    Message   forms.Textarea
    Submitted bool
    Summary   string
}

func NewContactForm(tr *i18n.Translator) *ContactForm {
    c := &ContactForm{
        Name: forms.TextInput{
            CommonAttrs:     forms.CommonAttrs{Name: "name", ID: "name", Required: true},
            Placeholder:     "Your name",
            FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required()}},
        },
        Email: forms.TextInput{
            CommonAttrs: forms.CommonAttrs{Name: "email", ID: "email", Required: true},
            Type:        "email",
            Placeholder: "you@example.com",
            FieldValidation: forms.FieldValidation{
                Rules: []validation.Rule{validation.Required(), validation.Email()},
            },
        },
        Message: forms.Textarea{
            CommonAttrs: forms.CommonAttrs{Name: "message", ID: "message"},
            Rows:        4,
            FieldValidation: forms.FieldValidation{
                Rules: []validation.Rule{validation.Required(), validation.MaxLength(500)},
            },
        },
    }
    // Alt alanlar, ws.Session'dan otomatik olarak SetTranslator almaz —
    // sadece üst seviye BaseComponent alır, dolayısıyla her alanı açıkça bağlayın.
    c.SetTranslator(tr)
    c.Name.SetTranslator(tr)
    c.Email.SetTranslator(tr)
    c.Message.SetTranslator(tr)
    return c
}

func (c *ContactForm) Mount(_ context.Context) error   { return nil }
func (c *ContactForm) Unmount(_ context.Context) error { return nil }

func (c *ContactForm) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
    switch event {
    case "name":
        return c.Name.HandleEvent(ctx, event, payload)
    case "email":
        return c.Email.HandleEvent(ctx, event, payload)
    case "message":
        return c.Message.HandleEvent(ctx, event, payload)
    case "save":
        if !forms.ValidateAll(&c.Name, &c.Email, &c.Message) {
            // Geri yüklenecek hiçbir şey yok: Name.Value, Email.Value,
            // Message.Value zaten kullanıcının yazdığı şeydir. Her alanın
            // Errors'u doldurulmuştur.
            c.Submitted = false
            c.MarkDirty()
            return nil
        }
        c.Submitted = true
        c.Summary = c.Name.Value + " <" + c.Email.Value + "> " + c.Message.Value
        c.ToastT("success", "contact.submit_success")
        c.MarkDirty()
    }
    return nil
}

func (c *ContactForm) Render() (string, error) {
    nameL, _ := (&forms.Label{For: "name", Text: "Name"}).Render()
    nameI, _ := c.Name.Render()
    emailL, _ := (&forms.Label{For: "email", Text: "Email"}).Render()
    emailI, _ := c.Email.Render()
    msgL, _ := (&forms.Label{For: "message", Text: "Message"}).Render()
    msgI, _ := c.Message.Render()
    btn, _ := (&forms.Button{Type: "button", Text: "Send", EventName: "save"}).Render()

    out := ""
    if c.Submitted {
        out = `<div class="result">` + html.EscapeString(c.Summary) + `</div>`
    }

    inner := forms.JoinHTML(
        `<div class="field">`, nameL, nameI, `</div>`,
        `<div class="field">`, emailL, emailI, `</div>`,
        `<div class="field">`, msgL, msgI, `</div>`,
        `<div class="actions">`, btn, `</div>`,
        out,
    )
    form := &forms.Form{Method: "post", OnSubmit: "save", InnerHTML: inner}
    return form.Render()
}
```

Ne olmadığına dikkat edin: başarısız bir `save`'den sonra gönderilen bir
değeri `Name.Value`'ya geri kopyalayan hiçbir kod yoktur — daha en baştan
hiç kaldırılmamıştı. Doğrulamanın mevcut duruma *eklediği* tek şey her
alandaki `Errors` dilimidir.

Bu tam kalıbın (registry, çevirmen, hub, HTML sayfası, broadcast uç
noktası ile) tam olarak bağlanmış, çalıştırılabilir versiyonu için
`examples/contact-form/main.go`'ya bakın ve şununla çalıştırın:

```bash
go run ./examples/contact-form
# http://localhost:3001
```
</contents>
