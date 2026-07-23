# 07 — Forms Tier 2 (Rich Controls)

Tier 2 controls go beyond native HTML inputs: server-side search, tree/graph
selection, rich text/code/markdown editing, file uploads with previews,
drawing pads, and more. All controls live in
**`github.com/zatrano/goui/forms`**, including the select family, phone
input, data-driven pickers, editors, uploads, and visual controls.

```go
import "github.com/zatrano/goui/forms"
```

Every field below embeds `core.BaseComponent` + `forms.CommonAttrs` +
`forms.FieldValidation` (unless noted) and implements the shared
`Name()/RawValue()/SetRawValue(string)`/`Validate() bool` contract described
in [`05-forms-tier1.md`](05-forms-tier1.md) and
[`06-validation.md`](06-validation.md). All are **server-rendered**: the Go
struct is the single source of truth, and the browser only owns the small
slice of UI state noted explicitly below as "UI-only" (e.g. which month a
calendar is currently showing).

---

## The select family (`forms`)

All of these embed `forms.BaseSelectField`:

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
    Filtered    []SelectItem // last server-side filter pass
    Query       string
    Open        bool
    Value       string
    Values      []string
    FilterMode  FilterMode // FilterServer (default) | FilterClient
    MaxResults  int        // default 50
    Placeholder string

    OnChange func(value string)
    OnQuery  func(query string)
}
```

**Server vs. UI-only:** filtering is **server-side by default**
(`FilterServer`) — every keystroke sends a `query` event, the server
recomputes `Filtered` via `forms.FilterItems` (case-insensitive
substring match on label/value, capped at `MaxResults`), and streams back
the new `<li>` list as a patch. `FilterMode: FilterClient` exists for small,
fixed lists but even then **the server still owns selection state**; nothing
in `forms` filters the option DOM purely in JavaScript. The
optional `selectable.js` client module reinforces this: it only adds
keyboard highlight/Enter-to-select on top of whatever list the server
already rendered — **it does not filter options client-side.**

### Searchable Select

```go
type SearchableSelect struct {
    BaseSelectField
    EventName string // prefix for events, e.g. "city" → city.query / city.select
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

`HandleEvent` actions (dispatched via `<eventName>.<action>`): `toggle`,
`open`, `close`, `query`, `select`. No client module file of its own — its
open/close/keyboard behavior is covered generically by `selectable.js`.

### Multi Select

```go
type MultiSelect struct {
    BaseSelectField
    EventName string
}
```

Same shape as `SearchableSelect` but tracks `Values []string`; renders
selected items as removable `<span class="goui-chip">` tags. Actions: `toggle`,
`open`, `close`, `query`, `select` (toggles membership), `remove`.

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
    RestrictToList bool // when true, rejects free text — Value only from Items
}
```

A text input that also opens a filtered suggestion panel. Unless
`RestrictToList` is set, every keystroke also sets `Value` to the raw typed
text (free text allowed); picking a suggestion sets `Value` to the item and
`Query` to its label. Actions: `toggle`/`open`, `close`, `query`, `select`,
`commit`.

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

Like `Combobox` but does **not** set `Value` while typing — only a
suggestion pick (or `commit` with no pick, which falls back to the typed
text) sets `Value`. Actions: `query`, `select`, `commit`, `close`.

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

// ChipsInput is an alias highlighting chip presentation of TagInput values.
type ChipsInput = TagInput
```

Not built on `BaseSelectField` — it's free-text tag collection, not a
picker over a fixed `Items` list. Deduplicates case-insensitively. Actions:
`draft` (as-you-type buffer), `add`/`commit` (comma-separated input
supported — `"go, rust"` adds both), `remove`.

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

`Mount` lazily allocates `Expanded`. Renders a nested `<ul>` with
expand/collapse toggles per branch node. Actions: `toggle` (panel
open/close), `close`, `expand` (toggle a node's expanded state), `select`.

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

Multi-column drill-down where each pick loads the next column server-side
via your `LoadChildren` callback. `Mount` seeds `Levels[0]` from `Items` if
`Levels` is empty. `RawValue()` joins every level's selection with `/`.
Action: `pick` (payload carries both `value` and `level`); picking clears
any deeper levels and either appends a new column (`LoadChildren` returned
items) or commits (`OnChange`) if there are no children.

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

Two independently-searchable columns ("available" / "selected") with move
actions. Both sides are filtered server-side (`ApplyAvailableQuery`,
`ApplySelectedQuery`). Actions: `query_left`/`query` (available side),
`query_right` (selected side), `add`, `remove`, `add_all`, `remove_all`.

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

    Dial   SearchableSelect // dial code
    Number forms.TextInput  // national number
}

func NewPhoneInput(name string) *PhoneInput
```

Not a new control family — a composition helper wiring a
`SearchableSelect` (dial code, preloaded from `forms.DialCodeItems()`,
default `+90`) next to a `forms.TextInput` (`type=tel`). `RawValue()`
returns an E.164-ish `"<dial> <number>"` string. `HandleEvent` dispatches
by matching event prefix to whichever sub-field it belongs to.

```go
phone := forms.NewPhoneInput("phone") // *PhoneInput
```

---

## Country / Language / Timezone / Currency Picker (`forms`)

These are **not** separate struct types — they are `SearchableSelect`
factory functions preloaded with curated `[]SelectItem` data
(`forms.CountryItems()`, `LanguageItems()`, `TimezoneItems()`,
`CurrencyItems()`):

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

Everything documented above for `SearchableSelect` (server-side filter,
`selectable.js` keyboard nav only, no client-side filtering) applies
unchanged.

---

## Emoji / Icon / Font Picker (`forms`)

Same pattern as the pickers above — `SearchableSelect` factories over
curated item sets (`EmojiItems()`, `IconItems()`, `FontItems()`):

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

`FontItems()` returns full CSS `font-family` stacks as `Value` (e.g.
`"Georgia, serif"`) so you can apply `Value` directly as an inline
`style="font-family:..."` for a live preview, as `examples/misc-controls`
does.

`forms.MentionUsers()` — a small curated `[]SelectItem` directory of
sample users — lives in the same file and is meant to seed `MentionUser`
lists for `forms.MentionTextarea` (below), not to be rendered as a picker
itself.

---

## Currency Input (`forms`)

```go
type CurrencyInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation

    Value     float64
    Currency  string // ISO code, default TRY
    Locale    string // default "tr"
    Decimals  int    // default 2
    Draft     string // raw text while typing
    EventName string
    OnChange  func(value float64)
}
```

Stores a `float64`; **all display formatting is server-side**
(`forms.NumberFormat`/`forms.ParseLocalizedNumber` — `tr` uses
`1.234,56`-style grouping, `en` uses `1,234.56`). While typing, raw text is
held in `Draft` and only committed to `Value` on blur/change if it parses;
otherwise the unparsed `Draft` stays visible so the user can fix a typo.

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

    Value     float64 // percentage points, e.g. 45.5 means 45,5%
    Locale    string
    Decimals  int      // default 1
    Min, Max  *float64
    Draft     string
    EventName string
    OnChange  func(value float64)
}
```

Same draft/commit/locale-formatting pattern as `CurrencyInput`, with
optional `Min`/`Max` clamping.

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
    Max       int // default 5
    Icon      string // default ★
    EmptyIcon string // default ☆
    EventName string
    OnChange  func(value int)
}
```

Renders `Max` `<button>` icons; clicking the currently-selected star toggles
it back to `0` (allows "un-rating"). No client module — pure `g-click` per
star with `data-goui-value`.

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

Two native `<input type="date">`/`<input type="time">` elements side by
side; `Validate()` adds an extra error (`forms.date_range.invalid` /
`forms.time_range.invalid`) when `End < Start`. No client module — both are
plain native inputs with `g-change`.

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

**Server vs. UI-only:** the selected `Value` (`YYYY-MM-DD`), `Min`/`Max`
bounds, and open/closed state are server-owned. **Month/year navigation is
client-only** — the `‹`/`›` header buttons in `client/modules/calendar.js`
(`enhanceCalendar`) move a local `view` variable and re-render the grid
purely in the browser, with **no** network round-trip per month change.
Only the final **day click** sends a `g-click` (`data-goui-value="<ymd>"`)
back to the server, via the event name in `data-select-event` on the panel.
This is why the server-rendered panel is just a placeholder
(`<div class="goui-calendar-placeholder">Loading…</div>`) until
`calendar.js` mounts and takes over — it purposefully never re-renders that
subtree from the server.

```go
day := forms.CalendarDatePicker{
    CommonAttrs: forms.CommonAttrs{Name: "day", ID: "day"},
    Value: "2026-07-16", Placeholder: "Pick a date", EventName: "day",
}
```

Client module: `client/modules/calendar.js` (`enhanceCalendar(root)`).

---

## OTP / PIN Input (`forms`)

```go
type OTPInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Length    int // default 6
    Value     string
    Masked    bool // password-style cells (PIN)
    EventName string
    OnChange  func(value string)
}

// PINInput is an alias; set Masked: true for PIN UX.
type PINInput = OTPInput
```

Renders `Length` single-character `<input>` cells (`type=password` when
`Masked`). The full code lives server-side in `Value`; per-cell edits use
action `digit` (payload carries `index` + `value`), full replace uses
`commit`/`paste`/`change`/`input`. `Validate()` adds
`forms.otp.incomplete` when the collected length doesn't match `Length`.

```go
otp := forms.OTPInput{CommonAttrs: forms.CommonAttrs{Name: "otp", ID: "otp"}, Length: 6, EventName: "otp"}
pin := forms.PINInput{CommonAttrs: forms.CommonAttrs{Name: "pin", ID: "pin"}, Length: 4, Masked: true, EventName: "pin"}
```

Client module: `client/modules/otp.js` (`enhanceOTP`) — **UI-only**
auto-advance-to-next-cell, backspace-to-previous, arrow-key navigation, and
paste-splits-across-cells. It fires native `input` events per cell so the
existing `g-input` delegation still ships every digit to the server; it
does not itself talk to the WebSocket.

---

## Rich Text Editor (`forms`)

```go
type RichTextEditor struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value      string // HTML
    EventName  string
    DebounceMS int    // default 350
    OnChange   func(value string)
}
```

**Server vs. UI-only:** `Value` (the HTML content) is authoritative on the
server, but the *editing surface* is entirely client-owned — a
[Quill](https://quilljs.com/) instance loaded from a CDN
(`client/modules/richtext.js`, `enhanceRichText`/`mountQuill`). The rendered
markup carries `data-goui-ignore` on the wrapper so the diff-patch client
**never reconciles into it** (see `applyPatch`'s `isGoUIIgnored` check in
`client/goui.js`) — patching Quill's live DOM would tear down cursor
position, undo history, and selection.

Content sync works through a hidden `<textarea class="goui-editor-sync">`
that Quill writes into on every `text-change` and dispatches a synthetic
`input` event on, debounced by `g-debounce`. Server-side,
`HandleEvent`'s `sync` action updates `Value` **without calling
`MarkDirty()`** — and the corresponding demo additionally returns
`core.ErrSkipRender` from the parent component's `HandleEvent` for this
control, so **no `render` frame is ever sent back** for rich-text sync
events:

```go
case strings.HasPrefix(event, "rt."):
    _ = d.Rich.HandleEvent(ctx, event, payload)
    // Quill owns the DOM — patching would remount and double-escape HTML.
    return core.ErrSkipRender
```

```go
rich := forms.RichTextEditor{CommonAttrs: forms.CommonAttrs{Name: "rt", ID: "rt"}, Value: "<p>Hello</p>", EventName: "rt"}
```

Client module: `client/modules/richtext.js`.

## Markdown Editor (`forms`)

```go
type MarkdownEditor struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Value       string
    PreviewHTML string
    Rows        int    // default 10
    Placeholder string
    EventName   string
    DebounceMS  int    // default 250
    OnChange    func(value string)
}
```

**Server vs. UI-only:** the source `<textarea>` is a normal server-rendered
Tier-1-style control (no client module, no `data-goui-ignore`) — every
keystroke round-trips through `g-input`/`sync` like a regular `Textarea`.
The live preview pane is rendered **entirely on the server** using
[goldmark](https://github.com/yuin/goldmark) via the exported helper:

```go
func RenderMarkdown(source string) string
```

`Mount` and every `sync` `HandleEvent` call `refreshPreview()`, which sets
`PreviewHTML = RenderMarkdown(Value)`; `Render()` emits that HTML directly
into a `<div class="goui-markdown-preview">`. Because this is a normal
(non-ignored) subtree, the diff engine happily patches it like any other
server-rendered HTML.

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
    Language   string // e.g. javascript, go, htmlmixed — default javascript
    EventName  string
    DebounceMS int    // default 350
    OnChange   func(value string)
}
```

**Server vs. UI-only:** identical pattern to `RichTextEditor` — a
[CodeMirror 5](https://codemirror.net/5/) instance from a CDN
(`client/modules/codeeditor.js`, `enhanceCodeEditor`/`mountCM`) owns the
editing surface, marked `data-goui-ignore` so patches never touch it, and
syncs through a hidden `<textarea class="goui-editor-sync">` debounced by
`g-debounce`. Just like rich text, the parent component's `HandleEvent`
should return `core.ErrSkipRender` for `code.*` sync events:

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

Client module: `client/modules/codeeditor.js`.

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
    UploadURL  string // default /goui/upload
    EventName  string
    OnChange   func(files []UploadedRef)
}

// ImageUpload preset: Accept "image/*", ShowThumbs true.
func NewImageUpload(name, event string) DragDropUpload
```

**Server vs. UI-only:** binary bytes never travel over the WebSocket.
`client/modules/upload.js` (`enhanceUpload`) intercepts drag/drop and file
input `change`, `POST`s the raw file to `data-upload-url` (via your adapter's `Store` option or `upload.Mount`,
which writes to an `upload.Storage` — e.g.
`upload.LocalStore` — and returns JSON `Meta`), then synthesizes a click on
a hidden `<button class="goui-upload-carrier" g-click="<event>.uploaded">`
carrying the metadata as `data-goui-*` attributes so the existing
`g-click`/`collectPayload` delegation ships an `event` frame with `id`,
`name`, `url`, `size`, `contentType` — **only the small JSON reference**
travels over the socket. Server-side, action `uploaded` appends/replaces a
`forms.UploadedRef`; action `remove` drops one by ID.

```go
docs := forms.DragDropUpload{
    CommonAttrs: forms.CommonAttrs{Name: "docs", ID: "docs"},
    Multiple:    true,
    Accept:      ".pdf,.txt,.png,.jpg",
    ShowThumbs:  true,
    EventName:   "docs",
}
images := forms.NewImageUpload("images", "images") // DragDropUpload preset
```

Register the HTTP side once per app:

```go
store, err := upload.NewLocalStore("./.goui-uploads", "/goui/files", 8<<20)
gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})
// POST /goui/upload, GET /goui/files/:id
```

Client module: `client/modules/upload.js` (also imported by `avatar.js`
and `signature.js` below for their own `postFile`/`notifyUploaded` calls).

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

**Server vs. UI-only:** the final stored `Avatar` ref is server state; the
crop interaction itself is entirely client-side. `client/modules/avatar.js`
(`enhanceAvatar`) opens a `<canvas>` overlay on file pick, lets the user
pan (`pointerdown`/`pointermove`) a 1:1 square, and on **"Kırp & Yükle"
(Crop & Upload)** calls `canvas.toBlob(...)` to rasterize the crop client-side
into a PNG `Blob`, uploads that with the same `postFile`/`notifyUploaded`
helpers `upload.js` exposes, then hides the overlay. The server never sees
uncropped pixels or crop coordinates — only the final cropped PNG file
reference (`action: "uploaded"`) or a `"clear"` action to remove it.

```go
avatar := forms.AvatarUpload{CommonAttrs: forms.CommonAttrs{Name: "avatar", ID: "avatar"}, EventName: "avatar"}
```

Client module: `client/modules/avatar.js`.

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

**Server vs. UI-only:** drawing itself (`pointerdown`/`pointermove`
strokes on a `<canvas>`) is 100% client-side
(`client/modules/signature.js`, `enhanceSignature`/`mountPad`). Clicking
"Kaydet" (Save) rasterizes the canvas to a PNG blob and uploads it exactly
like `AvatarUpload` does, firing `action: "uploaded"` on success; "Temizle"
(local clear) only clears the canvas pixels without any server round-trip;
a separate server-bound "Kaydı sil" button (rendered only once `File.ID` is
set) sends `action: "clear"` to drop the stored reference.

```go
sig := forms.SignaturePad{CommonAttrs: forms.CommonAttrs{Name: "sig", ID: "sig"}, EventName: "sig"}
```

Client module: `client/modules/signature.js`.

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
    Rows        int // default 4
    Users       []MentionUser // full directory
    Filtered    []MentionUser
    Query       string        // text after @
    Open        bool
    EventName   string
    OnChange    func(value string)
}
```

A `<textarea>` that detects an unfinished `@fragment` at the cursor
position (via `mentionQuery`, string-based — looks at the whole `Value`,
not real cursor position, so it's a simplified "last `@`" heuristic) and
opens a **server-filtered** suggestion list (`filterUsers`, substring match
on ID/label, capped at 8). Picking a suggestion (`pick` action) replaces
the `@fragment` with `@<id> `. No client module — a plain `Textarea`-style
control with a conditionally-rendered `<ul>` after it.

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

An advanced alternative to the native `forms.ColorInput` from Tier 1: a row
of preset swatch buttons plus a free-text hex field. Defaults to 10 preset
swatches if `Swatches` is empty. Actions: `pick`/`select` (from a swatch),
`hex`/`change`/`input` (from the text field, normalized via
`normalizeHex` — lower-cased, `#`-prefixed).

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
    From, To, Angle string // e.g. Angle "135deg"
    EventName       string
    OnChange        func(css string)
}

func (g *GradientPicker) CSS() string // "linear-gradient(<angle>, <from>, <to>)"
```

Two native `<input type="color">` swatches plus a free-text angle field;
`Render()` shows a live preview `<div>` and the generated CSS as `<code>`.
Actions: `from`, `to`, `angle`.

```go
grad := forms.GradientPicker{
    CommonAttrs: forms.CommonAttrs{Name: "grad", ID: "grad"},
    From: "#2563eb", To: "#db2777", Angle: "135deg", EventName: "grad",
}
```

---

## Character Counter (`ShowCharCount`)

Not a separate struct — a **field on existing Tier 1 controls**:
`forms.TextInput.ShowCharCount` and `forms.Textarea.ShowCharCount`. When
`true`, `Render()` appends a `<p class="goui-char-count">` showing
`len(value) / MaxLength` (rune-counted), colored as an error once the count
exceeds `MaxLength`. Setting either flag also defaults `g-debounce` to
`100` if you haven't set `DebounceMS` yourself, so the counter updates
responsively without spamming events on every keystroke.

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

Also a field on `forms.TextInput`: `ShowStrength bool`, which only renders
when `Type == "password"`. Scoring is a small server-side heuristic
(`forms.PasswordStrength`, 0–4: length ≥8/≥12, character-class diversity)
exposed as:

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

`Render()` appends a `<div class="goui-password-strength <level>">` bar
(width = `level*25%`) plus a translated label (`forms.password_strength.*`
i18n keys — see [`03-i18n.md`](03-i18n.md)).

```go
pw := forms.TextInput{
    CommonAttrs:  forms.CommonAttrs{Name: "pw", ID: "pw"},
    Type:         "password",
    ShowStrength: true,
}
```

---

## Summary table

| Control | Package | Struct | Client module | Notes |
|---|---|---|---|---|
| Searchable Select | `forms` | `SearchableSelect` | — (uses `selectable.js`) | server-side filter |
| Multi Select | `forms` | `MultiSelect` | — (uses `selectable.js`) | chips of `Values` |
| Combobox | `forms` | `Combobox` | — (uses `selectable.js`) | free text unless `RestrictToList` |
| Autocomplete | `forms` | `Autocomplete` | — (uses `selectable.js`) | `Value` only set on pick/commit |
| Tag Input / Chips Input | `forms` | `TagInput` / `ChipsInput` (alias) | — | dedupe, comma-split |
| Tree Select | `forms` | `TreeSelect` | — | server-owned `Expanded` map |
| Cascader | `forms` | `Cascader` | — | `LoadChildren` callback |
| Dual Listbox | `forms` | `DualListbox` | — | two independently-filtered sides |
| Phone | `forms` | `PhoneInput` | — | composes `SearchableSelect` + `TextInput` |
| Country/Language/Timezone/Currency Picker | `forms` | `SearchableSelect` (via `NewXPicker`) | — | curated `SelectItem` data |
| Emoji/Icon/Font Picker | `forms` | `SearchableSelect` (via `NewXPicker`) | — | curated `SelectItem` data |
| Currency Input | `forms` | `CurrencyInput` | — | server locale formatting |
| Percentage Input | `forms` | `PercentageInput` | — | server locale formatting |
| Rating | `forms` | `Rating` | — | pure `g-click` |
| Date Range | `forms` | `DateRangePicker` | — | two native `<input type=date>` |
| Time Range | `forms` | `TimeRangePicker` | — | two native `<input type=time>` |
| Calendar | `forms` | `CalendarDatePicker` | `calendar.js` | **month nav is client-only** |
| OTP / PIN | `forms` | `OTPInput` / `PINInput` (alias) | `otp.js` | UI-only auto-advance/paste |
| Rich Text | `forms` | `RichTextEditor` | `richtext.js` | Quill; `ErrSkipRender` + `data-goui-ignore` |
| Markdown | `forms` | `MarkdownEditor` | — | server-rendered via goldmark |
| Code Editor | `forms` | `CodeEditor` | `codeeditor.js` | CodeMirror; `ErrSkipRender` + `data-goui-ignore` |
| DragDrop Upload | `forms` | `DragDropUpload` | `upload.js` | binary via HTTP, ref via WS |
| Image Upload | `forms` | `DragDropUpload` (via `NewImageUpload`) | `upload.js` | preset: `image/*` + thumbnails |
| Avatar Upload | `forms` | `AvatarUpload` | `avatar.js` | includes crop overlay |
| Image Cropper | `forms` | (part of `AvatarUpload`) | `avatar.js` | client-side canvas crop |
| Color (Swatch) | `forms` | `SwatchColorPicker` | — | swatches + hex field |
| Gradient | `forms` | `GradientPicker` | — | two colors + angle |
| Signature | `forms` | `SignaturePad` | `signature.js` | canvas draw → PNG upload |
| Mention | `forms` | `MentionTextarea` | — | server-filtered `@` suggestions |
| Character Counter | `forms` | `TextInput.ShowCharCount` / `Textarea.ShowCharCount` | — | field flag, not a struct |
| Password Strength | `forms` | `TextInput.ShowStrength` | — | field flag, requires `Type: "password"` |

For fully wired, runnable versions of every control above, see the
`examples/` directory (ports and mapping documented in
[`01-getting-started.md`](01-getting-started.md)) — in particular
`searchable-select` (3002), `numeric-controls` (3003), `field-meta` (3004),
`date-controls` (3005), `identity-inputs` (3006), `editors` (3007),
`media-upload` (3008), and `misc-controls` (3009).
