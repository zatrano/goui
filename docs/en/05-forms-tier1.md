# 05 — Forms Tier 1 (Native Controls)

Tier 1 controls map directly onto native HTML form elements. They live in
`github.com/zatrano/goui/forms` and share two embedded helper types:

```go
import "github.com/zatrano/goui/forms"
```

- **`forms.CommonAttrs`** — the attributes almost every control accepts:
  `ID`, `Class`, `Title`, `TabIndex *int`, `Spellcheck *bool`,
  `Draggable *bool`, `AriaLabel`, `AriaDescribedBy`, `Autocomplete`,
  `Disabled`, `ReadOnly`, `Required`, `Autofocus`, `Name`.
- **`forms.FieldValidation`** — `Rules []validation.Rule` and
  `Errors []string`; see [`06-validation.md`](06-validation.md) for the full
  validation story. Embedding it gives you `Validate() bool` on the field.

Every Tier 1 field also implements the shared `forms.FieldValue` contract
(`Name() string`, `RawValue() string`, `SetRawValue(string)`), which is what
`forms.ValidateAll` and generic form-binding code rely on.

All examples below assume:

```go
import (
    "github.com/zatrano/goui/core"
    "github.com/zatrano/goui/forms"
)
```

---

## `TextInput`

Covers `type=text|password|email|search|tel|url`.

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
    EventName     string // g-change / g-input event name (defaults to Name)
    DebounceMS    int
    OnChange      func(newValue string)
    ShowCharCount bool
    ShowStrength  bool   // password strength meter (server-side)
    HelperText    string // hint below the field
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

`ShowCharCount`/`ShowStrength` render extra `<p>`/`<div>` metadata below the
input (see [`forms.fieldMetaHTML`](../../forms/field_meta.go)); `ShowStrength`
only takes effect when `Type == "password"`.

---

## `NumericInput`

Covers `type=number|range`.

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

Covers `type=date|time|datetime-local|month|week`.

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

Covers `type=checkbox|radio`. `CheckboxInput` and `RadioInput` are type
aliases of the same struct — purely for readability at the call site.

```go
type ChoiceInput struct {
    core.BaseComponent
    CommonAttrs
    FieldValidation
    Type      string // checkbox | radio
    Value     string // submitted value when checked
    Checked   bool
    EventName string
    LabelText string // optional adjacent label text
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

Covers `type=file`. Only tracks the last selected file name(s) client-side;
actual binary upload is a separate flow (see `forms.DragDropUpload` /
`forms.AvatarUpload` in [`07-forms-tier2.md`](07-forms-tier2.md) and the
`upload` package).

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
    Value     string // last selected file name(s) for display/state
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

Covers `type=color`. Not validated (no `FieldValidation` embed); defaults to
`#000000` when `Value` is empty.

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

Covers `type=hidden`. No validation, no events (`HandleEvent` is a no-op —
hidden fields are set programmatically from server logic, not user input).

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

`Optgroup` renders as a nested `<optgroup label="...">` block after the
top-level `Options`. `HandleEvent` re-derives every `Option.Selected` (in
both `Options` and every `Optgroup.Options`) from the incoming value, so the
struct stays consistent for subsequent renders.

---

## `Button`

Covers `type=submit|button|reset|image`.

```go
type Button struct {
    core.BaseComponent
    CommonAttrs
    Type      string
    Value     string
    Text      string
    Src       string // type=image
    Alt       string
    EventName string // g-click event
}
```

```go
save := forms.Button{Type: "button", Text: "Save", EventName: "save"}
```

```html
<button type="button" class="goui-button ..." g-click="save">Save</button>
```

`Type == "image"` renders an `<input type="image" src="..." alt="...">`
instead of a `<button>` element.

---

## `Form`, `Fieldset`, `Legend`, `Label`

Structural containers with no server-side state of their own — they render
attributes plus caller-supplied inner HTML (composed with `forms.JoinHTML`).

```go
type Form struct {
    core.BaseComponent
    CommonAttrs
    Action    string
    Method    string
    EncType   string
    InnerHTML string
    OnSubmit  string // g-submit event name
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

`g-submit` intercepts the native form submission (`preventDefault`) and
sends an `event` frame whose payload is `{"fields": {...}}` gathered from
every named form control via `FormData` — see
[`06-validation.md`](06-validation.md) for a complete form example wiring
this up end-to-end.

---

## `Datalist`

Backs an input's `list="<id>"` attribute for native autocomplete
suggestions.

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

Displays a calculation result (e.g. the result of a `<form>` with linked
`<input>`s via `for`).

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

Represents a scalar measurement within a known range.

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

Represents task completion. `Max` defaults to `1` when left at the zero
value.

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

## Putting it together

A minimal registered component composing several Tier 1 fields:

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

Every Tier 1 field's `HandleEvent` responds to `"change"`/`"input"` (where
applicable) *and* its own configured event name (`EventName`, defaulting to
`Name`) — so dispatching by `event` string directly to the matching field's
`HandleEvent`, as above, works without any extra plumbing. See the shipped
`examples/contact-form` for a complete, runnable version with validation and
toasts.
