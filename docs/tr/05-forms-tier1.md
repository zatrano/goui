# 05 — Forms Tier 1 (Native Kontroller)

Tier 1 kontrolleri, native HTML form elemanlarına doğrudan eşlenir.
`github.com/zatrano/goui/forms` içinde yaşarlar ve iki gömülü yardımcı
türü paylaşırlar:

```go
import "github.com/zatrano/goui/forms"
```

- **`forms.CommonAttrs`** — neredeyse her kontrolün kabul ettiği
  öznitelikler: `ID`, `Class`, `Title`, `TabIndex *int`, `Spellcheck *bool`,
  `Draggable *bool`, `AriaLabel`, `AriaDescribedBy`, `Autocomplete`,
  `Disabled`, `ReadOnly`, `Required`, `Autofocus`, `Name`.
- **`forms.FieldValidation`** — `Rules []validation.Rule` ve
  `Errors []string`; tam doğrulama hikayesi için
  [`06-validation.md`](06-validation.md)'ye bakın. Bunu gömmek, alan
  üzerinde `Validate() bool` verir.

Her Tier 1 alanı, `forms.ValidateAll`'un ve genel form-bağlama kodunun
dayandığı, paylaşılan `forms.FieldValue` sözleşmesini de
(`Name() string`, `RawValue() string`, `SetRawValue(string)`) uygular.

Aşağıdaki tüm örnekler şunu varsayar:

```go
import (
    "github.com/zatrano/goui/core"
    "github.com/zatrano/goui/forms"
)
```

---

## `TextInput`

`type=text|password|email|search|tel|url`'i kapsar.

```go
type TextInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Type          string
    Value         string
    Placeholder   string
    MinLength     int
    MaxLength     int
    Pattern       string
    Size          int
    Multiple      bool // email
    List          string
    EventName     string // g-change / g-input olay adı (varsayılan Name)
    DebounceMS    int
    OnChange      func(newValue string)
    ShowCharCount bool
    ShowStrength  bool   // parola gücü göstergesi (sunucu tarafı)
    HelperText    string // alanın altında ipucu
}
```

```go
email := forms.TextInput{
    CommonAttrs: forms.CommonAttrs{Name: "email", ID: "email", Required: true},
    Type:        "email",
    Placeholder: "ornek@mail.com",
}
html, err := email.Render()
```

```html
<input id="email" name="email" required type="email"
       value="" placeholder="ornek@mail.com"
       g-change="email" g-input="email">
```

`ShowCharCount`/`ShowStrength`, girdinin altında ekstra `<p>`/`<div>`
meta verisi render eder (bkz.
[`forms.fieldMetaHTML`](../../forms/field_meta.go)); `ShowStrength` yalnızca
`Type == "password"` olduğunda etkili olur.

---

## `NumericInput`

`type=number|range`'i kapsar.

```go
type NumericInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Type      string
    Value     string
    Min       string
    Max       string
    Step      string
    EventName string
    OnChange  func(newValue string)
}
```

```go
qty := forms.NumericInput{
    CommonAttrs: forms.CommonAttrs{Name: "qty", ID: "qty"},
    Type:        "number",
    Min:         "0",
    Max:         "10",
    Step:        "1",
}
```

```html
<input id="qty" name="qty" type="number" value="" min="0" max="10" step="1"
       g-change="qty" g-input="qty">
```

---

## `DateTimeInput`

`type=date|time|datetime-local|month|week`'i kapsar.

```go
type DateTimeInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Type      string
    Value     string
    Min       string
    Max       string
    Step      string
    EventName string
    OnChange  func(newValue string)
}
```

```go
birthday := forms.DateTimeInput{
    CommonAttrs: forms.CommonAttrs{Name: "birthday", ID: "birthday"},
    Type:        "date",
}
```

```html
<input id="birthday" name="birthday" type="date" value="" g-change="birthday">
```

---

## `ChoiceInput` (checkbox / radio)

`type=checkbox|radio`'yu kapsar. `CheckboxInput` ve `RadioInput`, aynı
struct'ın tür takma adlarıdır (type alias) — çağrı noktasında sadece
okunabilirlik için.

```go
type ChoiceInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Type      string // checkbox | radio
    Value     string // işaretlendiğinde gönderilen değer
    Checked   bool
    EventName string
    LabelText string // isteğe bağlı bitişik etiket metni
    OnChange  func(checked bool, value string)
}

type CheckboxInput = ChoiceInput
type RadioInput = ChoiceInput
```

```go
subscribe := forms.ChoiceInput{
    CommonAttrs: forms.CommonAttrs{Name: "subscribe", ID: "subscribe"},
    Type:        "checkbox",
    Value:       "yes",
    LabelText:   "Subscribe to newsletter",
}
```

```html
<input id="subscribe" name="subscribe" type="checkbox" value="yes" g-change="subscribe">
<span>Subscribe to newsletter</span>
```

---

## `FileInput`

`type=file`'ı kapsar. Sadece son seçilen dosya adı/adlarını istemci
tarafında takip eder; gerçek ikili yükleme ayrı bir akıştır (bkz.
[`07-forms-tier2.md`](07-forms-tier2.md)'deki `forms.DragDropUpload` /
`forms.AvatarUpload` ve `upload` paketi).

```go
type FileInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Accept    string
    Capture   string
    Multiple  bool
    EventName string
    OnChange  func(fileNames string)
    Value     string // görüntüleme/durum için son seçilen dosya adı/adları
}
```

```go
avatar := forms.FileInput{
    CommonAttrs: forms.CommonAttrs{Name: "avatar", ID: "avatar"},
    Accept:      "image/*",
}
```

```html
<input id="avatar" name="avatar" type="file" accept="image/*" g-change="avatar">
```

---

## `ColorInput`

`type=color`'ı kapsar. Doğrulanmaz (`FieldValidation` gömülü değildir);
`Value` boş olduğunda `#000000`'a varsayılan olur.

```go
type ColorInput struct {
    core.BaseComponent
    CommonAttrs
    Value     string
    EventName string
    OnChange  func(newValue string)
}
```

```go
brand := forms.ColorInput{
    CommonAttrs: forms.CommonAttrs{Name: "brand", ID: "brand"},
    Value:       "#2563eb",
}
```

```html
<input id="brand" name="brand" type="color" value="#2563eb" g-change="brand">
```

---

## `HiddenInput`

`type=hidden`'ı kapsar. Doğrulama yok, olay yok (`HandleEvent` no-op'tur —
gizli alanlar kullanıcı girdisinden değil, sunucu mantığından
programatik olarak ayarlanır).

```go
type HiddenInput struct {
    core.BaseComponent
    CommonAttrs
    Value string
}
```

```go
csrf := forms.HiddenInput{
    CommonAttrs: forms.CommonAttrs{Name: "csrf_token"},
    Value:       "abc123",
}
```

```html
<input name="csrf_token" type="hidden" value="abc123">
```

---

## `Textarea`

```go
type Textarea struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value         string
    Placeholder   string
    Rows          int
    Cols          int
    Wrap          string
    MinLength     int
    MaxLength     int
    EventName     string
    DebounceMS    int
    OnChange      func(newValue string)
    ShowCharCount bool
    HelperText    string
}
```

```go
message := forms.Textarea{
    CommonAttrs:   forms.CommonAttrs{Name: "message", ID: "message"},
    Rows:          4,
    MaxLength:     500,
    ShowCharCount: true,
}
```

```html
<textarea id="message" name="message" rows="4" maxlength="500"
          g-change="message" g-input="message" g-debounce="100"></textarea>
<p class="goui-char-count text-sm">0 / 500</p>
```

---

## `Select`, `Option`, `Optgroup`

```go
type Option struct {
    Value    string
    Label    string
    Selected bool
    Disabled bool
}

type Optgroup struct {
    Label    string
    Disabled bool
    Options  []Option
}

type Select struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value     string
    Multiple  bool
    Size      int
    Options   []Option
    Groups    []Optgroup
    EventName string
    OnChange  func(newValue string)
}
```

```go
country := forms.Select{
    CommonAttrs: forms.CommonAttrs{Name: "country", ID: "country"},
    Options: []forms.Option{
        {Value: "", Label: "Select a country"},
        {Value: "tr", Label: "Türkiye"},
        {Value: "us", Label: "United States"},
    },
}
```

```html
<select id="country" name="country" g-change="country">
  <option value="">Select a country</option>
  <option value="tr">Türkiye</option>
  <option value="us">United States</option>
</select>
```

`Optgroup`, üst seviye `Options`'tan sonra iç içe bir
`<optgroup label="...">` bloğu olarak render edilir. `HandleEvent`, gelen
değerden her `Option.Selected`'ı (hem `Options` hem de her
`Optgroup.Options` içinde) yeniden türetir, böylece struct sonraki
render'lar için tutarlı kalır.

---

## `Button`

`type=submit|button|reset|image`'i kapsar.

```go
type Button struct {
    core.BaseComponent
    CommonAttrs
    Type      string
    Value     string
    Text      string
    Src       string // type=image
    Alt       string
    EventName string // g-click olayı
}
```

```go
save := forms.Button{Type: "button", Text: "Save", EventName: "save"}
```

```html
<button type="button" class="goui-button ..." g-click="save">Save</button>
```

`Type == "image"`, bir `<button>` elemanı yerine
`<input type="image" src="..." alt="...">` render eder.

---

## `Form`, `Fieldset`, `Legend`, `Label`

Kendine ait sunucu tarafı durumu olmayan yapısal konteynerler — öznitelik
artı çağıranın sağladığı iç HTML'i (`forms.JoinHTML` ile birleştirilmiş)
render ederler.

```go
type Form struct {
    core.BaseComponent
    CommonAttrs
    Action    string
    Method    string
    EncType   string
    InnerHTML string
    OnSubmit  string // g-submit olay adı
}

type Fieldset struct {
    core.BaseComponent
    CommonAttrs
    InnerHTML string
}

type Legend struct {
    core.BaseComponent
    CommonAttrs
    Text string
}

type Label struct {
    core.BaseComponent
    CommonAttrs
    For  string
    Text string
}
```

```go
nameLabel, _ := (&forms.Label{For: "name", Text: "Name"}).Render()
nameInput, _ := (&forms.TextInput{CommonAttrs: forms.CommonAttrs{Name: "name", ID: "name"}}).Render()

form := &forms.Form{
    Method:   "post",
    OnSubmit: "save",
    InnerHTML: forms.JoinHTML(
        `<div class="field">`, nameLabel, nameInput, `</div>`,
    ),
}
html, err := form.Render()
```

```html
<form class="goui-form ..." method="post" g-submit="save">
  <div class="field">
    <label class="goui-label ..." for="name">Name</label>
    <input id="name" name="name" type="text" value="">
  </div>
</form>
```

`g-submit`, native form submit'ini yakalar (`preventDefault`) ve
payload'ı her adlandırılmış form kontrolünden `FormData` aracılığıyla
toplanmış `{"fields": {...}}` olan bir `event` frame'i gönderir — bunu
uçtan uca nasıl bağlayacağınıza dair tam bir form örneği için
[`06-validation.md`](06-validation.md)'ye bakın.

---

## `Datalist`

Bir girdinin `list="<id>"` özniteliğini, native autocomplete önerileri
için destekler.

```go
type DatalistOption struct {
    Value string
    Label string
}

type Datalist struct {
    core.BaseComponent
    CommonAttrs
    Options []DatalistOption
}
```

```go
cities := forms.Datalist{
    CommonAttrs: forms.CommonAttrs{ID: "cities"},
    Options: []forms.DatalistOption{
        {Value: "ist", Label: "İstanbul"},
        {Value: "ank", Label: "Ankara"},
    },
}
```

```html
<datalist id="cities">
  <option value="ist" label="İstanbul">İstanbul</option>
  <option value="ank" label="Ankara">Ankara</option>
</datalist>
<input list="cities">
```

---

## `Output`

Bir hesaplama sonucunu görüntüler (örn. `for` aracılığıyla bağlanmış
`<input>`ları olan bir `<form>`'un sonucu).

```go
type Output struct {
    core.BaseComponent
    CommonAttrs
    For   string
    Form  string
    Value string
    Text  string
}
```

```go
summary := forms.Output{
    CommonAttrs: forms.CommonAttrs{Name: "summary"},
    Text:        "Name: Ada | Email: ada@example.com",
}
```

```html
<output name="summary">Name: Ada | Email: ada@example.com</output>
```

---

## `Meter`

Bilinen bir aralık içindeki skaler bir ölçümü temsil eder.

```go
type Meter struct {
    core.BaseComponent
    CommonAttrs
    Value   float64
    Min     float64
    Max     float64
    Low     float64
    High    float64
    Optimum float64
}
```

```go
disk := forms.Meter{Value: 0.6, Min: 0, Max: 1, High: 0.9, Optimum: 0.3}
```

```html
<meter value="0.6" min="0" max="1" high="0.9" optimum="0.3"></meter>
```

---

## `Progress`

Görev tamamlanmasını temsil eder. `Max`, sıfır değerinde bırakıldığında
`1`'e varsayılan olur.

```go
type Progress struct {
    core.BaseComponent
    CommonAttrs
    Value float64
    Max   float64
}
```

```go
upload := forms.Progress{Value: 0.42}
```

```html
<progress value="0.42" max="1"></progress>
```

---

## Hepsini bir araya getirme

Birkaç Tier 1 alanını kompoze eden minimal, kayıtlı bir bileşen:

```go
type ContactForm struct {
    core.BaseComponent
    Name  forms.TextInput
    Email forms.TextInput
}

func NewContactForm() *ContactForm {
    return &ContactForm{
        Name:  forms.TextInput{CommonAttrs: forms.CommonAttrs{Name: "name", ID: "name"}},
        Email: forms.TextInput{CommonAttrs: forms.CommonAttrs{Name: "email", ID: "email"}, Type: "email"},
    }
}

func (c *ContactForm) Mount(_ context.Context) error   { return nil }
func (c *ContactForm) Unmount(_ context.Context) error { return nil }

func (c *ContactForm) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
    switch event {
    case "name":
        return c.Name.HandleEvent(ctx, event, payload)
    case "email":
        return c.Email.HandleEvent(ctx, event, payload)
    }
    return nil
}

func (c *ContactForm) Render() (string, error) {
    nameHTML, _ := c.Name.Render()
    emailHTML, _ := c.Email.Render()
    return forms.JoinHTML(`<div class="field">`, nameHTML, `</div>`,
        `<div class="field">`, emailHTML, `</div>`), nil
}
```

Her Tier 1 alanının `HandleEvent`'i, `"change"`/`"input"`'a (uygunsa) *ve*
kendi yapılandırılmış olay adına (`EventName`, varsayılan olarak `Name`)
yanıt verir — dolayısıyla `event` string'ine göre yukarıdaki gibi
doğrudan eşleşen alanın `HandleEvent`'ine dispatch etmek, ekstra bir
bağlantı gerektirmeden çalışır. Doğrulama ve toast'larla tam, çalıştırılabilir
bir versiyon için gönderilen `examples/contact-form`'a bakın.
</contents>
