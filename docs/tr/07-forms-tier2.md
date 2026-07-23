# 07 — Forms Tier 2 (Zengin Kontroller)

Tier 2 kontrolleri, native HTML girdilerinin ötesine geçer: sunucu tarafı
arama, ağaç/graf seçimi, zengin metin/kod/markdown düzenleme, önizlemeli
dosya yüklemeleri, çizim pedleri ve daha fazlası. Select ailesi, telefon
girdisi, veri güdümlü picker'lar, editörler, upload ve görsel kontroller
dahil tamamı **`github.com/zatrano/goui/forms`** paketinde yaşar.

```go
import "github.com/zatrano/goui/forms"
```

Aşağıdaki her alan `core.BaseComponent` + `forms.CommonAttrs` +
`forms.FieldValidation`'ı gömer (belirtilmediği yerler hariç) ve
[`05-forms-tier1.md`](05-forms-tier1.md) ile
[`06-validation.md`](06-validation.md)'de açıklanan paylaşılan
`Name()/RawValue()/SetRawValue(string)`/`Validate() bool` sözleşmesini
uygular. Hepsi **sunucu tarafında render edilir**: Go struct'ı tek doğruluk
kaynağıdır, ve tarayıcı sadece aşağıda açıkça "UI-only (sadece UI)" olarak
not edilen küçük UI durumu dilimine sahiptir (örn. bir takvimin şu anda
hangi ayı gösterdiği).

---

## Select ailesi (`forms`)

Bunların hepsi `forms.BaseSelectField`'i gömer:

```go
type SelectItem struct {
    Value    string
    Label    string
    Disabled bool
}

type BaseSelectField struct {
    core.BaseComponent
    forms.CommonAttrs
    forms.FieldValidation

    Items       []SelectItem
    Filtered    []SelectItem // son sunucu tarafı filtre geçişi
    Query       string
    Open        bool
    Value       string
    Values      []string
    FilterMode  FilterMode // FilterServer (varsayılan) | FilterClient
    MaxResults  int        // varsayılan 50
    Placeholder string

    OnChange func(value string)
    OnQuery  func(query string)
}
```

**Sunucu vs. UI-only:** filtreleme varsayılan olarak **sunucu
taraflıdır** (`FilterServer`) — her tuş vuruşu bir `query` olayı gönderir,
sunucu `forms.FilterItems` aracılığıyla `Filtered`'ı yeniden hesaplar
(label/value üzerinde büyük/küçük harfe duyarsız alt dize eşleşmesi,
`MaxResults` ile sınırlandırılmış), ve yeni `<li>` listesini bir yama
olarak geri gönderir. `FilterMode: FilterClient`, küçük, sabit listeler
için vardır ama o durumda bile **sunucu seçim durumunun sahibidir**;
`forms`'de hiçbir şey seçenek DOM'unu saf olarak JavaScript'te
filtrelemez. İsteğe bağlı `selectable.js` istemci modülü bunu
güçlendirir: sadece sunucunun zaten render ettiği listenin üzerine
klavye vurgusu/Enter-ile-seçim ekler — **seçenekleri istemci tarafında
filtrelemez.**

### Searchable Select

```go
type SearchableSelect struct {
    BaseSelectField
    EventName string // olaylar için önek, örn. "city" → city.query / city.select
}
```

```go
city := forms.SearchableSelect{
    BaseSelectField: forms.BaseSelectField{
        CommonAttrs: forms.CommonAttrs{Name: "city", ID: "city"},
        Placeholder: "Select a city",
        Items: []forms.SelectItem{
            {Value: "ist", Label: "İstanbul"},
            {Value: "ank", Label: "Ankara"},
        },
    },
    EventName: "city",
}
```

`HandleEvent` eylemleri (`<eventName>.<action>` üzerinden dispatch
edilir): `toggle`, `open`, `close`, `query`, `select`. Kendine ait bir
istemci modülü dosyası yoktur — açma/kapama ve klavye davranışı genel
olarak `selectable.js` tarafından kapsanır.

### Multi Select

```go
type MultiSelect struct {
    BaseSelectField
    EventName string
}
```

`SearchableSelect` ile aynı şekle sahiptir ama `Values []string`'i takip
eder; seçili öğeleri kaldırılabilir `<span class="goui-chip">` etiketleri
olarak render eder. Eylemler: `toggle`, `open`, `close`, `query`,
`select` (üyeliği açar/kapatır), `remove`.

```go
cities := forms.MultiSelect{
    BaseSelectField: forms.BaseSelectField{
        CommonAttrs: forms.CommonAttrs{Name: "cities", ID: "cities"},
        Items:       cityItems,
    },
    EventName: "cities",
}
```

### Combobox

```go
type Combobox struct {
    BaseSelectField
    EventName      string
    RestrictToList bool // true olduğunda, serbest metni reddeder — Value yalnızca Items'tan
}
```

Filtrelenmiş bir öneri panelini de açan bir metin girdisi.
`RestrictToList` ayarlanmadıkça, her tuş vuruşu ham yazılan metni
`Value`'ya da ayarlar (serbest metin izinlidir); bir öneri seçmek
`Value`'yu öğeye ve `Query`'yi onun etiketine ayarlar. Eylemler:
`toggle`/`open`, `close`, `query`, `select`, `commit`.

```go
role := forms.Combobox{
    BaseSelectField: forms.BaseSelectField{
        CommonAttrs: forms.CommonAttrs{Name: "role", ID: "role"},
        Items: []forms.SelectItem{{Value: "admin", Label: "Admin"}, {Value: "editor", Label: "Editor"}},
    },
    EventName: "role",
}
```

### Autocomplete

```go
type Autocomplete struct {
    BaseSelectField
    EventName string
}
```

`Combobox` gibidir ama yazarken `Value`'yu **ayarlamaz** — sadece bir
öneri seçimi (veya seçim olmadan `commit`, ki bu yazılan metne geri
döner) `Value`'yu ayarlar. Eylemler: `query`, `select`, `commit`,
`close`.

```go
suggest := forms.Autocomplete{
    BaseSelectField: forms.BaseSelectField{CommonAttrs: forms.CommonAttrs{Name: "suggest", ID: "suggest"}, Items: cityItems},
    EventName: "suggest",
}
```

### Tag Input / Chips Input

```go
type TagInput struct {
    core.BaseComponent
    forms.CommonAttrs
    forms.FieldValidation

    Values      []string
    Draft       string
    Placeholder string
    EventName   string
    OnChange    func(tags []string)
}

// ChipsInput, TagInput değerlerinin çip sunumunu vurgulayan bir takma addır.
type ChipsInput = TagInput
```

`BaseSelectField` üzerine inşa edilmemiştir — sabit bir `Items` listesi
üzerinde bir seçici değil, serbest metin etiket koleksiyonudur.
Büyük/küçük harfe duyarsız olarak yinelenenleri temizler. Eylemler:
`draft` (yazarken tampon), `add`/`commit` (virgülle ayrılmış girdi
desteklenir — `"go, rust"` her ikisini de ekler), `remove`.

```go
skills := forms.TagInput{
    CommonAttrs: forms.CommonAttrs{Name: "skills", ID: "skills"},
    Placeholder: "Add a tag (Enter/blur)",
    EventName:   "skills",
}
```

### Tree Select

```go
type TreeNode struct {
    Value    string
    Label    string
    Disabled bool
    Children []TreeNode
}

type TreeSelect struct {
    BaseSelectField
    Nodes     []TreeNode
    Expanded  map[string]bool
    EventName string
}
```

`Mount`, `Expanded`'i tembel (lazily) tahsis eder. Her dal düğümü için
genişlet/daralt geçişleriyle iç içe bir `<ul>` render eder. Eylemler:
`toggle` (panel açma/kapama), `close`, `expand` (bir düğümün genişletme
durumunu değiştirir), `select`.

```go
dept := forms.TreeSelect{
    BaseSelectField: forms.BaseSelectField{CommonAttrs: forms.CommonAttrs{Name: "dept", ID: "dept"}},
    EventName:       "dept",
    Nodes: []forms.TreeNode{
        {Value: "eng", Label: "Engineering", Children: []forms.TreeNode{
            {Value: "be", Label: "Backend"}, {Value: "fe", Label: "Frontend"},
        }},
    },
}
```

### Cascader

```go
type CascaderLevel struct {
    Items    []SelectItem
    Selected string
}

type Cascader struct {
    BaseSelectField
    EventName    string
    Levels       []CascaderLevel
    LoadChildren func(level int, parentValue string) []SelectItem
}
```

Her seçimin sonraki sütunu sunucu tarafında sizin `LoadChildren`
callback'inizle yüklediği çok sütunlu bir "içine inme" (drill-down)
kontrolüdür. `Mount`, `Levels` boşsa `Levels[0]`'ı `Items`'tan besler.
`RawValue()`, her seviyenin seçimini `/` ile birleştirir. Eylem: `pick`
(payload hem `value` hem `level` taşır); seçim yapmak herhangi bir daha
derin seviyeyi temizler ve ya yeni bir sütun ekler (`LoadChildren`'ın
döndürdüğü öğeler) ya da hiç çocuk yoksa commit eder (`OnChange`).

```go
loc := forms.Cascader{
    BaseSelectField: forms.BaseSelectField{
        CommonAttrs: forms.CommonAttrs{Name: "loc", ID: "loc"},
        Items:       []forms.SelectItem{{Value: "tr", Label: "Türkiye"}, {Value: "de", Label: "Almanya"}},
    },
    EventName: "loc",
    LoadChildren: func(level int, parent string) []forms.SelectItem {
        if level == 0 && parent == "tr" {
            return []forms.SelectItem{{Value: "ist", Label: "İstanbul"}, {Value: "ank", Label: "Ankara"}}
        }
        return nil
    },
}
```

### Dual Listbox

```go
type DualListbox struct {
    BaseSelectField
    EventName      string
    SelectedQuery  string
    SelectedFilter []SelectItem
}
```

Taşıma eylemleriyle birlikte iki bağımsız olarak aranabilir sütun
("mevcut" / "seçili"). Her iki taraf da sunucu tarafında filtrelenir
(`ApplyAvailableQuery`, `ApplySelectedQuery`). Eylemler:
`query_left`/`query` (mevcut taraf), `query_right` (seçili taraf), `add`,
`remove`, `add_all`, `remove_all`.

```go
perms := forms.DualListbox{
    BaseSelectField: forms.BaseSelectField{CommonAttrs: forms.CommonAttrs{Name: "perms", ID: "perms"}, Items: permItems},
    EventName: "perms",
}
```

---

## Phone Input (`forms`)

```go
type PhoneInput struct {
    forms.CommonAttrs
    forms.FieldValidation

    Dial   SearchableSelect // çevirme kodu
    Number forms.TextInput  // ulusal numara
}

func NewPhoneInput(name string) *PhoneInput
```

Yeni bir kontrol ailesi değildir — bir `SearchableSelect`'i (çevirme
kodu, `forms.DialCodeItems()`'tan önceden yüklenmiş, varsayılan
`+90`) bir `forms.TextInput` (`type=tel`) yanına bağlayan bir kompozisyon
yardımcısıdır. `RawValue()`, bir E.164 benzeri `"<dial> <number>"`
string'i döndürür. `HandleEvent`, olay önekini ait olduğu alt alanla
eşleştirerek dispatch eder.

```go
phone := forms.NewPhoneInput("phone") // *PhoneInput
```

---

## Country / Language / Timezone / Currency Picker (`forms`)

Bunlar ayrı struct türleri **değildir** — curated (derlenmiş)
`[]SelectItem` verisiyle (`forms.CountryItems()`, `LanguageItems()`,
`TimezoneItems()`, `CurrencyItems()`) önceden yüklenmiş
`SearchableSelect` factory fonksiyonlarıdır:

```go
func NewCountryPicker(name, event string) SearchableSelect
func NewLanguagePicker(name, event string) SearchableSelect
func NewTimezonePicker(name, event string) SearchableSelect
func NewCurrencyPicker(name, event string) SearchableSelect
```

```go
country := forms.NewCountryPicker("country", "country")   // SearchableSelect
language := forms.NewLanguagePicker("lang", "lang")
tz := forms.NewTimezonePicker("tz", "tz")
currency := forms.NewCurrencyPicker("cur", "cur")
```

`SearchableSelect` için yukarıda belgelenen her şey (sunucu tarafı
filtre, sadece `selectable.js` klavye navigasyonu, istemci tarafında
filtreleme yok) değişmeden geçerlidir.

---

## Emoji / Icon / Font Picker (`forms`)

Yukarıdaki seçicilerle aynı desen — curated öğe kümeleri
(`EmojiItems()`, `IconItems()`, `FontItems()`) üzerinde `SearchableSelect`
factory'leri:

```go
func NewEmojiPicker(name, event string) SearchableSelect
func NewIconPicker(name, event string) SearchableSelect
func NewFontPicker(name, event string) SearchableSelect
```

```go
emoji := forms.NewEmojiPicker("emoji", "emoji")
icon := forms.NewIconPicker("icon", "icon")
font := forms.NewFontPicker("font", "font")
```

`FontItems()`, `Value` olarak tam CSS `font-family` yığınlarını
döndürür (örn. `"Georgia, serif"`), böylece `Value`'yu doğrudan satır içi
`style="font-family:..."` olarak canlı bir önizleme için
uygulayabilirsiniz, `examples/misc-controls`'un yaptığı gibi.

`forms.MentionUsers()` — örnek kullanıcıların küçük, curated bir
`[]SelectItem` dizini — aynı dosyada yaşar ve kendisi bir seçici olarak
render edilmek için değil, aşağıdaki `forms.MentionTextarea` için
`MentionUser` listelerini beslemek içindir.

---

## Currency Input (`forms`)

```go
type CurrencyInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation

    Value     float64
    Currency  string // ISO kodu, varsayılan TRY
    Locale    string // varsayılan "tr"
    Decimals  int    // varsayılan 2
    Draft     string // yazarken ham metin
    EventName string
    OnChange  func(value float64)
}
```

Bir `float64` saklar; **tüm görüntüleme biçimlendirmesi sunucu
taraflıdır** (`forms.NumberFormat`/`forms.ParseLocalizedNumber` — `tr`
`1.234,56` tarzı gruplama kullanır, `en` `1,234.56` kullanır). Yazarken,
ham metin `Draft`'ta tutulur ve sadece ayrıştırılabiliyorsa
blur/change'de `Value`'ya commit edilir; aksi halde ayrıştırılamayan
`Draft` görünür kalır, böylece kullanıcı bir yazım hatasını düzeltebilir.

```go
price := forms.CurrencyInput{
    CommonAttrs: forms.CommonAttrs{Name: "price", ID: "price"},
    Currency:    "TRY",
    Locale:      "tr",
    Value:       1250.5,
    EventName:   "price",
}
```

## Percentage Input (`forms`)

```go
type PercentageInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation

    Value     float64 // yüzde puanı, örn. 45.5, 45,5% anlamına gelir
    Locale    string
    Decimals  int      // varsayılan 1
    Min, Max  *float64
    Draft     string
    EventName string
    OnChange  func(value float64)
}
```

`CurrencyInput` ile aynı draft/commit/locale-biçimlendirme deseni,
isteğe bağlı `Min`/`Max` sınırlamasıyla.

```go
max, min := 100.0, 0.0
vat := forms.PercentageInput{
    CommonAttrs: forms.CommonAttrs{Name: "vat", ID: "vat"},
    Value: 20, Min: &min, Max: &max, EventName: "vat",
}
```

## Rating (`forms`)

```go
type Rating struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation

    Value     int // 0..Max
    Max       int // varsayılan 5
    Icon      string // varsayılan ★
    EmptyIcon string // varsayılan ☆
    EventName string
    OnChange  func(value int)
}
```

`Max` `<button>` simgesi render eder; şu anda seçili yıldıza tıklamak
onu `0`'a geri döndürür ("un-rating" — puanı geri almaya olanak tanır).
İstemci modülü yok — her yıldız için `data-goui-value` ile saf
`g-click`.

```go
score := forms.Rating{CommonAttrs: forms.CommonAttrs{Name: "score", ID: "score"}, Value: 3, Max: 5, EventName: "score"}
```

---

## Date Range / Time Range Picker (`forms`)

```go
type DateRangePicker struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Start, End, Min, Max string
    EventName             string
    OnChange              func(start, end string)
}

type TimeRangePicker struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Start, End, Min, Max, Step string
    EventName                   string
    OnChange                    func(start, end string)
}
```

Yan yana iki native `<input type="date">`/`<input type="time">` elemanı;
`End < Start` olduğunda `Validate()` ekstra bir hata ekler
(`forms.date_range.invalid` / `forms.time_range.invalid`). İstemci
modülü yok — her ikisi de `g-change` ile düz native girdilerdir.

```go
leave := forms.DateRangePicker{
    CommonAttrs: forms.CommonAttrs{Name: "leave", ID: "leave"},
    Start: "2026-07-10", End: "2026-07-15", EventName: "leave",
}
shift := forms.TimeRangePicker{
    CommonAttrs: forms.CommonAttrs{Name: "shift", ID: "shift"},
    Start: "09:00", End: "17:30", EventName: "shift",
}
```

## Calendar Date Picker (`forms`)

```go
type CalendarDatePicker struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value, Min, Max string
    Open            bool
    Placeholder     string
    EventName       string
    OnChange        func(value string)
}
```

**Sunucu vs. UI-only:** seçili `Value` (`YYYY-MM-DD`), `Min`/`Max`
sınırları, ve açık/kapalı durum sunucu sahiplidir. **Ay/yıl navigasyonu
yalnızca istemci taraflıdır** — `client/modules/calendar.js`'deki
(`enhanceCalendar`) `‹`/`›` başlık düğmeleri yerel bir `view` değişkenini
hareket ettirir ve ızgarayı tamamen tarayıcıda yeniden render eder, ay
değişikliği başına **hiçbir** ağ gidiş-dönüşü olmadan. Sadece nihai
**gün tıklaması**, panelde `data-select-event` içindeki olay adı
aracılığıyla sunucuya bir `g-click` (`data-goui-value="<ymd>"`) geri
gönderir. Bu yüzden sunucu tarafında render edilen panel, `calendar.js`
mount olup devralana kadar sadece bir yer tutucudur
(`<div class="goui-calendar-placeholder">Loading…</div>`) — bu, o alt
ağacı sunucudan kasıtlı olarak asla yeniden render etmez.

```go
day := forms.CalendarDatePicker{
    CommonAttrs: forms.CommonAttrs{Name: "day", ID: "day"},
    Value: "2026-07-16", Placeholder: "Pick a date", EventName: "day",
}
```

İstemci modülü: `client/modules/calendar.js` (`enhanceCalendar(root)`).

---

## OTP / PIN Input (`forms`)

```go
type OTPInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Length    int // varsayılan 6
    Value     string
    Masked    bool // parola tarzı hücreler (PIN)
    EventName string
    OnChange  func(value string)
}

// PINInput bir takma addır; PIN UX'i için Masked: true ayarlayın.
type PINInput = OTPInput
```

`Length` tek karakterli `<input>` hücresi render eder (`Masked`
olduğunda `type=password`). Tam kod sunucu tarafında `Value`'da yaşar;
hücre başına düzenlemeler `digit` eylemini kullanır (payload `index` +
`value` taşır), tam değiştirme `commit`/`paste`/`change`/`input`
kullanır. Toplanan uzunluk `Length` ile eşleşmediğinde `Validate()`,
`forms.otp.incomplete`'i ekler.

```go
otp := forms.OTPInput{CommonAttrs: forms.CommonAttrs{Name: "otp", ID: "otp"}, Length: 6, EventName: "otp"}
pin := forms.PINInput{CommonAttrs: forms.CommonAttrs{Name: "pin", ID: "pin"}, Length: 4, Masked: true, EventName: "pin"}
```

İstemci modülü: `client/modules/otp.js` (`enhanceOTP`) — **sadece
UI'ye ait** otomatik-sonraki-hücreye-geçme, backspace-ile-öncekine-dönme,
ok tuşu navigasyonu, ve yapıştırma-hücreler-arasında-bölünür. Her hücre
için native `input` olayları ateşler, böylece mevcut `g-input`
delegasyonu her rakamı sunucuya göndermeye devam eder; kendisi
WebSocket ile doğrudan konuşmaz.

---

## Rich Text Editor (`forms`)

```go
type RichTextEditor struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value      string // HTML
    EventName  string
    DebounceMS int    // varsayılan 350
    OnChange   func(value string)
}
```

**Sunucu vs. UI-only:** `Value` (HTML içeriği) sunucuda yetkilidir, ama
*düzenleme yüzeyinin* kendisi tamamen istemciye aittir — bir CDN'den
yüklenen bir [Quill](https://quilljs.com/) örneği
(`client/modules/richtext.js`, `enhanceRichText`/`mountQuill`). Render
edilen markup, sarmalayıcı üzerinde `data-goui-ignore` taşır, böylece
diff-yama istemcisi **onun içine asla uzlaştırma (reconcile) yapmaz**
(bkz. `client/goui.js`'deki `applyPatch`'in `isGoUIIgnored` kontrolü) —
Quill'in canlı DOM'unu yamalamak, imleç konumunu, geri alma geçmişini ve
seçimi bozardı.

İçerik senkronizasyonu, Quill'in her `text-change`'de yazdığı ve üzerinde
`g-debounce` ile debounce edilmiş sentetik bir `input` olayı ateşlediği
gizli bir `<textarea class="goui-editor-sync">` üzerinden çalışır. Sunucu
tarafında, `HandleEvent`'in `sync` eylemi, `Value`'yu **`MarkDirty()`'yi
çağırmadan** güncelleyerek — ve karşılık gelen demo ek olarak bu kontrol
için üst bileşenin `HandleEvent`'inden `core.ErrSkipRender` döndürerek —
bu şekilde rich-text senkronizasyon olayları için **hiçbir `render`
frame'i asla geri gönderilmez**:

```go
case strings.HasPrefix(event, "rt."):
    _ = d.Rich.HandleEvent(ctx, event, payload)
    // Quill DOM'un sahibidir — yamalamak yeniden mount eder ve HTML'i iki kez escape eder.
    return core.ErrSkipRender
```

```go
rich := forms.RichTextEditor{CommonAttrs: forms.CommonAttrs{Name: "rt", ID: "rt"}, Value: "<p>Hello</p>", EventName: "rt"}
```

İstemci modülü: `client/modules/richtext.js`.

## Markdown Editor (`forms`)

```go
type MarkdownEditor struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value       string
    PreviewHTML string
    Rows        int    // varsayılan 10
    Placeholder string
    EventName   string
    DebounceMS  int    // varsayılan 250
    OnChange    func(value string)
}
```

**Sunucu vs. UI-only:** kaynak `<textarea>`, normal bir sunucu tarafında
render edilen Tier-1-tarzı kontroldür (istemci modülü yok,
`data-goui-ignore` yok) — her tuş vuruşu normal bir `Textarea` gibi
`g-input`/`sync` üzerinden gidiş-dönüş yapar. Canlı önizleme paneli, dışa
açık yardımcı aracılığıyla [goldmark](https://github.com/yuin/goldmark)
kullanılarak **tamamen sunucuda** render edilir:

```go
func RenderMarkdown(source string) string
```

`Mount` ve her `sync` `HandleEvent` çağrısı, `PreviewHTML = RenderMarkdown(Value)`'yu
ayarlayan `refreshPreview()`'i çağırır; `Render()` bu HTML'i doğrudan bir
`<div class="goui-markdown-preview">` içine yayar. Bu normal (yok
sayılmamış) bir alt ağaç olduğundan, diff motoru onu diğer herhangi bir
sunucu tarafında render edilmiş HTML gibi seve seve yamalar.

```go
md := forms.MarkdownEditor{
    CommonAttrs: forms.CommonAttrs{Name: "md", ID: "md"},
    Value:       "# Hello\n\n**Markdown** rendered server-side.",
    Rows:        12,
    EventName:   "md",
}
```

## Code Editor (`forms`)

```go
type CodeEditor struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value      string
    Language   string // örn. javascript, go, htmlmixed — varsayılan javascript
    EventName  string
    DebounceMS int    // varsayılan 350
    OnChange   func(value string)
}
```

**Sunucu vs. UI-only:** `RichTextEditor` ile aynı desen — bir CDN'den
[CodeMirror 5](https://codemirror.net/5/) örneği
(`client/modules/codeeditor.js`, `enhanceCodeEditor`/`mountCM`)
düzenleme yüzeyinin sahibidir, yamaların onu asla dokunmaması için
`data-goui-ignore` işaretlidir, ve `g-debounce` ile debounce edilmiş
gizli bir `<textarea class="goui-editor-sync">` aracılığıyla senkronize
olur. Tam olarak rich text gibi, üst bileşenin `HandleEvent`'i
`code.*` senkronizasyon olayları için `core.ErrSkipRender` döndürmelidir:

```go
case strings.HasPrefix(event, "code."):
    _ = d.Code.HandleEvent(ctx, event, payload)
    return core.ErrSkipRender
```

```go
code := forms.CodeEditor{
    CommonAttrs: forms.CommonAttrs{Name: "code", ID: "code"},
    Value:       "function hello() {\n  return 'GoUI';\n}\n",
    Language:    "javascript",
    EventName:   "code",
}
```

İstemci modülü: `client/modules/codeeditor.js`.

---

## Drag & Drop Upload / Image Upload (`forms`)

```go
type UploadedRef struct {
    ID          string
    Name        string
    URL         string
    ContentType string
    Size        int64
}

type DragDropUpload struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Files      []UploadedRef
    Accept     string
    Multiple   bool
    ShowThumbs bool
    UploadURL  string // varsayılan /goui/upload
    EventName  string
    OnChange   func(files []UploadedRef)
}

// ImageUpload preset'i: Accept "image/*", ShowThumbs true.
func NewImageUpload(name, event string) DragDropUpload
```

**Sunucu vs. UI-only:** ikili baytlar WebSocket üzerinden asla
yolculuk etmez. `client/modules/upload.js` (`enhanceUpload`),
sürükle/bırak ve dosya girdisi `change`'ini yakalar, ham dosyayı
`data-upload-url`'e (varsayılan olarak) `POST` eder
(adapter'ınızın `Store` seçeneği veya `upload.Mount` aracılığıyla, ki bu bir
`upload.Storage`'a —
örn. `upload.LocalStore` — yazar ve JSON `Meta`'yı döndürür), ardından
metadata'yı `data-goui-*` özniteliklerinde taşıyan gizli bir
`<button class="goui-upload-carrier" g-click="<event>.uploaded">`
üzerinde sentetik bir tıklama üretir, böylece mevcut
`g-click`/`collectPayload` delegasyonu `id`, `name`, `url`, `size`,
`contentType` içeren bir `event` frame'i gönderir — soket üzerinden
**sadece küçük JSON referansı** yolculuk eder. Sunucu tarafında,
`uploaded` eylemi bir `forms.UploadedRef`'i ekler/değiştirir; `remove`
eylemi ID ile bir tanesini düşürür.

```go
docs := forms.DragDropUpload{
    CommonAttrs: forms.CommonAttrs{Name: "docs", ID: "docs"},
    Multiple:    true,
    Accept:      ".pdf,.txt,.png,.jpg",
    ShowThumbs:  true,
    EventName:   "docs",
}
images := forms.NewImageUpload("images", "images") // DragDropUpload preset'i
```

HTTP tarafını uygulama başına bir kez kaydedin:

```go
store, err := upload.NewLocalStore("./.goui-uploads", "/goui/files", 8<<20)
gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})
// POST /goui/upload, GET /goui/files/:id
```

İstemci modülü: `client/modules/upload.js` (kendi `postFile`/`notifyUploaded`
çağırıları için aşağıdaki `avatar.js` ve `signature.js` tarafından da
import edilir).

## Avatar Upload + Image Cropper (`forms`)

```go
type AvatarUpload struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Avatar    UploadedRef
    UploadURL string
    EventName string
    OnChange  func(ref UploadedRef)
}
```

**Sunucu vs. UI-only:** nihai saklanan `Avatar` referansı sunucu
durumudur; kırpma etkileşiminin kendisi tamamen istemci taraflıdır.
`client/modules/avatar.js` (`enhanceAvatar`), dosya seçiminde bir
`<canvas>` katmanı açar, kullanıcının 1:1 bir kareyi
kaydırmasına (`pointerdown`/`pointermove`) izin verir, ve **"Kırp &
Yükle" (Crop & Upload)** üzerinde `canvas.toBlob(...)`'u çağırarak
kırpmayı istemci tarafında bir PNG `Blob`'una rasterize eder, bunu
`upload.js`'in sunduğu aynı `postFile`/`notifyUploaded` yardımcılarıyla
yükler, ardından katmanı gizler. Sunucu asla kırpılmamış pikselleri veya
kırpma koordinatlarını görmez — sadece nihai kırpılmış PNG dosya
referansını (`action: "uploaded"`) veya onu kaldırmak için bir
`"clear"` eylemini görür.

```go
avatar := forms.AvatarUpload{CommonAttrs: forms.CommonAttrs{Name: "avatar", ID: "avatar"}, EventName: "avatar"}
```

İstemci modülü: `client/modules/avatar.js`.

## Signature Pad (`forms`)

```go
type SignaturePad struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    File      UploadedRef
    UploadURL string
    EventName string
    OnChange  func(ref UploadedRef)
}
```

**Sunucu vs. UI-only:** çizmenin kendisi (bir `<canvas>` üzerinde
`pointerdown`/`pointermove` darbeleri) %100 istemci taraflıdır
(`client/modules/signature.js`, `enhanceSignature`/`mountPad`). "Kaydet"e
tıklamak, canvas'ı bir PNG blob'una rasterize eder ve onu tam olarak
`AvatarUpload`'ın yaptığı gibi yükler, başarıda `action: "uploaded"`'ı
ateşler; "Temizle" (yerel temizleme) sadece sunucu gidiş-dönüşü olmadan
canvas piksellerini temizler; ayrı bir sunucuya bağlı "Kaydı sil" düğmesi
(sadece `File.ID` ayarlandığında render edilir) saklanan referansı
düşürmek için `action: "clear"` gönderir.

```go
sig := forms.SignaturePad{CommonAttrs: forms.CommonAttrs{Name: "sig", ID: "sig"}, EventName: "sig"}
```

İstemci modülü: `client/modules/signature.js`.

---

## Mention (`forms`)

```go
type MentionUser struct {
    ID    string
    Label string
}

type MentionTextarea struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value       string
    Placeholder string
    Rows        int // varsayılan 4
    Users       []MentionUser // tam dizin
    Filtered    []MentionUser
    Query       string        // @'dan sonraki metin
    Open        bool
    EventName   string
    OnChange    func(value string)
}
```

İmleç konumunda tamamlanmamış bir `@fragment` algılayan (`mentionQuery`
aracılığıyla, string tabanlı — gerçek imleç konumuna değil tüm
`Value`'ya bakar, dolayısıyla basitleştirilmiş bir "son `@`" sezgiseli
(heuristic)) ve **sunucu tarafında filtrelenmiş** bir öneri listesi
(`filterUsers`, ID/etiket üzerinde alt dize eşleşmesi, 8 ile
sınırlandırılmış) açan bir `<textarea>`. Bir öneri seçmek (`pick`
eylemi), `@fragment`'i `@<id> ` ile değiştirir. İstemci modülü yok — bir
düz `Textarea`-tarzı kontrol, ardından koşullu olarak render edilmiş bir
`<ul>`.

```go
mention := forms.MentionTextarea{
    CommonAttrs: forms.CommonAttrs{Name: "mention", ID: "mention"},
    Placeholder: "Tag someone with @...",
    Users:       []forms.MentionUser{{ID: "ayse", Label: "Ayşe Yılmaz"}},
    EventName:   "mention",
}
```

---

## Color (Swatch) Picker (`forms`)

```go
type SwatchColorPicker struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value     string // #rrggbb
    Swatches  []string
    EventName string
    OnChange  func(value string)
}
```

Tier 1'deki native `forms.ColorInput`'a gelişmiş bir alternatif: bir sıra
önceden ayarlı swatch düğmesi artı serbest metin bir hex alanı.
`Swatches` boşsa 10 önceden ayarlı swatch'a varsayılan olur. Eylemler:
`pick`/`select` (bir swatch'tan), `hex`/`change`/`input` (metin alanından,
`normalizeHex` aracılığıyla normalize edilmiş — küçük harfli,
`#`-önekli).

```go
color := forms.SwatchColorPicker{
    CommonAttrs: forms.CommonAttrs{Name: "color", ID: "color"},
    Value:       "#2563eb",
    EventName:   "color",
}
```

## Gradient Picker (`forms`)

```go
type GradientPicker struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    From, To, Angle string // örn. Angle "135deg"
    EventName       string
    OnChange        func(css string)
}

func (g *GradientPicker) CSS() string // "linear-gradient(<angle>, <from>, <to>)"
```

İki native `<input type="color">` swatch'ı artı serbest metin bir açı
alanı; `Render()`, canlı bir önizleme `<div>`'i ve üretilen CSS'i
`<code>` olarak gösterir. Eylemler: `from`, `to`, `angle`.

```go
grad := forms.GradientPicker{
    CommonAttrs: forms.CommonAttrs{Name: "grad", ID: "grad"},
    From: "#2563eb", To: "#db2777", Angle: "135deg", EventName: "grad",
}
```

---

## Character Counter (`ShowCharCount`)

Ayrı bir struct değildir — mevcut Tier 1 kontrolleri üzerinde **bir
alan**: `forms.TextInput.ShowCharCount` ve `forms.Textarea.ShowCharCount`.
`true` olduğunda, `Render()`, `MaxLength`'i aştığında hata olarak
renklendirilmiş, `len(value) / MaxLength`'i (rune sayılmış) gösteren bir
`<p class="goui-char-count">` ekler. Her iki bayrağı ayarlamak da, kendiniz
`DebounceMS`'i ayarlamadıysanız `g-debounce`'u `100`'e varsayılan yapar,
böylece sayaç her tuş vuruşunda olayları spamlamadan duyarlı bir şekilde
güncellenir.

```go
bio := forms.Textarea{
    CommonAttrs:   forms.CommonAttrs{Name: "bio", ID: "bio"},
    Rows:          4,
    MaxLength:     120,
    ShowCharCount: true,
    HelperText:    "Up to 120 characters",
}
```

## Password Strength (`ShowStrength`)

Ayrıca `forms.TextInput` üzerinde bir alandır: `ShowStrength bool`, ki bu
sadece `Type == "password"` olduğunda render edilir. Puanlama, küçük bir
sunucu tarafı sezgiseldir (`forms.PasswordStrength`, 0–4: uzunluk
≥8/≥12, karakter sınıfı çeşitliliği) ve şu şekilde açığa çıkarılmıştır:

```go
type PasswordStrengthLevel int
const (
    StrengthEmpty PasswordStrengthLevel = iota
    StrengthWeak
    StrengthFair
    StrengthGood
    StrengthStrong
)
func PasswordStrength(password string) PasswordStrengthLevel
```

`Render()`, bir çevrilmiş etiketle (`forms.password_strength.*` i18n
key'leri — bkz. [`03-i18n.md`](03-i18n.md)) birlikte bir
`<div class="goui-password-strength <level>">` çubuğu ekler (genişlik =
`level*25%`).

```go
pw := forms.TextInput{
    CommonAttrs:  forms.CommonAttrs{Name: "pw", ID: "pw"},
    Type:         "password",
    ShowStrength: true,
}
```

---

## Özet tablo

| Kontrol | Paket | Struct | İstemci modülü | Notlar |
|---|---|---|---|---|
| Searchable Select | `forms` | `SearchableSelect` | — (`selectable.js` kullanır) | sunucu tarafı filtre |
| Multi Select | `forms` | `MultiSelect` | — (`selectable.js` kullanır) | `Values`'ın çipleri |
| Combobox | `forms` | `Combobox` | — (`selectable.js` kullanır) | `RestrictToList` olmadıkça serbest metin |
| Autocomplete | `forms` | `Autocomplete` | — (`selectable.js` kullanır) | `Value` yalnızca seçim/commit'te ayarlanır |
| Tag Input / Chips Input | `forms` | `TagInput` / `ChipsInput` (takma ad) | — | yinelenen temizleme, virgülle bölme |
| Tree Select | `forms` | `TreeSelect` | — | sunucu sahipli `Expanded` map'i |
| Cascader | `forms` | `Cascader` | — | `LoadChildren` callback'i |
| Dual Listbox | `forms` | `DualListbox` | — | iki bağımsız filtrelenmiş taraf |
| Phone | `forms` | `PhoneInput` | — | `SearchableSelect` + `TextInput`'i kompoze eder |
| Country/Language/Timezone/Currency Picker | `forms` | `SearchableSelect` (`NewXPicker` üzerinden) | — | curated `SelectItem` verisi |
| Emoji/Icon/Font Picker | `forms` | `SearchableSelect` (`NewXPicker` üzerinden) | — | curated `SelectItem` verisi |
| Currency Input | `forms` | `CurrencyInput` | — | sunucu locale biçimlendirmesi |
| Percentage Input | `forms` | `PercentageInput` | — | sunucu locale biçimlendirmesi |
| Rating | `forms` | `Rating` | — | saf `g-click` |
| Date Range | `forms` | `DateRangePicker` | — | iki native `<input type=date>` |
| Time Range | `forms` | `TimeRangePicker` | — | iki native `<input type=time>` |
| Calendar | `forms` | `CalendarDatePicker` | `calendar.js` | **ay navigasyonu yalnızca istemci taraflıdır** |
| OTP / PIN | `forms` | `OTPInput` / `PINInput` (takma ad) | `otp.js` | sadece UI otomatik-ilerleme/yapıştırma |
| Rich Text | `forms` | `RichTextEditor` | `richtext.js` | Quill; `ErrSkipRender` + `data-goui-ignore` |
| Markdown | `forms` | `MarkdownEditor` | — | goldmark aracılığıyla sunucu tarafında render edilir |
| Code Editor | `forms` | `CodeEditor` | `codeeditor.js` | CodeMirror; `ErrSkipRender` + `data-goui-ignore` |
| DragDrop Upload | `forms` | `DragDropUpload` | `upload.js` | ikili HTTP üzerinden, referans WS üzerinden |
| Image Upload | `forms` | `DragDropUpload` (`NewImageUpload` üzerinden) | `upload.js` | preset: `image/*` + küçük resimler |
| Avatar Upload | `forms` | `AvatarUpload` | `avatar.js` | kırpma katmanını içerir |
| Image Cropper | `forms` | (`AvatarUpload`'ın bir parçası) | `avatar.js` | istemci tarafı canvas kırpma |
| Color (Swatch) | `forms` | `SwatchColorPicker` | — | swatch'lar + hex alanı |
| Gradient | `forms` | `GradientPicker` | — | iki renk + açı |
| Signature | `forms` | `SignaturePad` | `signature.js` | canvas çizim → PNG yükleme |
| Mention | `forms` | `MentionTextarea` | — | sunucu tarafında filtrelenmiş `@` önerileri |
| Character Counter | `forms` | `TextInput.ShowCharCount` / `Textarea.ShowCharCount` | — | alan bayrağı, struct değil |
| Password Strength | `forms` | `TextInput.ShowStrength` | — | alan bayrağı, `Type: "password"` gerektirir |

Yukarıdaki her kontrolün tam olarak bağlanmış, çalıştırılabilir
versiyonları için `examples/` dizinine bakın (portlar ve eşleme
[`01-getting-started.md`](01-getting-started.md)'de belgelenmiştir) —
özellikle `searchable-select` (3002), `numeric-controls` (3003),
`field-meta` (3004), `date-controls` (3005), `identity-inputs` (3006),
`editors` (3007), `media-upload` (3008), ve `misc-controls` (3009).
</contents>
