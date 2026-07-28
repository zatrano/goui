# 06 — Validation

GoUI's validation lives in two packages that work together:

- **`github.com/zatrano/goui/validation`** — pure, stateless `Rule` functions
  and the `Validate` runner.
- **`github.com/zatrano/goui/forms`** — `FieldValidation` (embedded in every
  Tier 1/Tier 2 field) and `ValidateAll`, which wire rules into the render
  pipeline (error messages, `aria-invalid`, error CSS class) and preserve
  submitted values on failure.

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

A rule takes the field's current **string** value and returns whether it
passed, plus an i18n message key to use when it did not. All rule
constructors below return a `Rule` closure.

### `Required()`

Fails when the value is empty or only whitespace.

```go
validation.Required() // → "validation.required" on failure
```

### `MinLength(n int)`

Fails when the **rune** length (not byte length — safe for multi-byte UTF-8
like Turkish `ğ, ü, ş, ö, ç, ı, İ`) is below `n`.

```go
validation.MinLength(3) // → "validation.min_length"
```

### `MaxLength(n int)`

Fails when the rune length is above `n`.

```go
validation.MaxLength(120) // → "validation.max_length"
```

### `Pattern(regex string)`

Fails when the value does not match the given regular expression. If
`regex` itself fails to compile, the returned rule **always fails** (fails
closed, not open) with `"validation.pattern"`.

```go
validation.Pattern(`^[A-Z]{2}\d{4}$`) // → "validation.pattern"
```

### `Email()`

Fails when the value does not match a simple `local@domain.tld` shape
(`^[^@\s]+@[^@\s]+\.[^@\s]+$`) — intentionally permissive, not a full RFC
5322 validator.

```go
validation.Email() // → "validation.email"
```

### `NumericRange(min, max float64)`

Parses the value as a `float64` and fails if it doesn't parse, or parses
outside `[min, max]` inclusive.

```go
validation.NumericRange(0, 100) // → "validation.numeric_range"
```

### `Custom(fn func(value string) bool, messageKey string)`

Wraps an arbitrary predicate with your own message key — the escape hatch
for anything the built-ins don't cover (uniqueness checks, cross-field
rules layered on top of `Validate`, business-specific formats, etc.).

```go
validation.Custom(func(v string) bool {
    return strings.HasPrefix(v, "TR")
}, "validation.custom")
```

## 2. Running rules: `Validate` and `ValidateAll`

### `validation.Validate`

```go
func Validate(value string, rules ...Rule) []string
```

Runs **every** rule against `value` — it does **not** stop at the first
failure — and returns the message keys for every rule that failed, in
order. `nil` rules in the slice are skipped safely.

```go
keys := validation.Validate("ab", validation.Required(), validation.MinLength(3), validation.Email())
// keys == ["validation.min_length", "validation.email"]
```

### `forms.ValidateAll`

```go
func ValidateAll(fields ...Validatable) bool
```

Where `Validatable` is:

```go
type Validatable interface {
    Validate() bool
}
```

Every Tier 1 and Tier 2 field type implements `Validate() bool` itself
(internally delegating to its embedded `FieldValidation.Run`), so you pass
pointers to the fields you want checked:

```go
ok := forms.ValidateAll(&c.Name, &c.Email, &c.Country, &c.Message, &c.Subscribe)
```

`ValidateAll` also does **not** short-circuit — like `validation.Validate`,
it calls `Validate()` on **every** field passed in (so every field's
`Errors` slice gets populated, not just the first failing one) and returns
`true` only if all of them passed. `nil` entries in the list are skipped.

### `forms.FieldValidation`

```go
type FieldValidation struct {
    Rules  []validation.Rule
    Errors []string
}
```

Embed this in your own field-like types to get validation for free:

```go
type MyField struct {
    core.BaseComponent
    forms.CommonAttrs
    forms.FieldValidation
    Value string
}

func (f *MyField) Validate() bool {
    return f.FieldValidation.run(f.Value, f.T) // package-private helper on FieldValidation
}
```

(Tier 1/Tier 2 fields inside the `forms`/`forms` packages call the
private `run`/exported `Run` variant directly; from outside those packages
use the exported `Run`:)

```go
func (f *MyField) Validate() bool {
    return f.FieldValidation.Run(f.Value, f.T)
}
```

`Run`/`run` does three things:

1. Calls `validation.Validate(value, f.Rules...)` to get failing message keys.
2. Translates each key via the `translate` function you pass (typically
   `f.T`, so error text respects the component's `Locale`) into
   `f.Errors []string`. If `translate` is `nil`, errors fall back to the
   raw `"[[key]]"` form.
3. Returns `true` only if there were zero failing keys.

Two more exported helpers on `FieldValidation` that Tier 1/Tier 2 `Render()`
implementations call directly:

```go
func (f *FieldValidation) ApplyErrorState(attrs Attrs, baseClass string) Attrs
func (f *FieldValidation) ErrorsHTML() string
```

- **`ApplyErrorState`** — if `Errors` is empty, ensures `baseClass` is
  applied as `class` (when the caller hasn't set a custom one) and returns
  `attrs` unchanged otherwise. If `Errors` is non-empty, sets
  `aria-invalid="true"` and appends the `border-goui-error` CSS class,
  merging with any existing `class` value.
- **`ErrorsHTML`** — renders each entry in `Errors` as
  `<p class="goui-field-error text-goui-error text-sm">...</p>` (HTML-escaped),
  concatenated. Every Tier 1/Tier 2 field's `Render()` appends this call's
  result right after the control's own markup.

## 3. State preserved on failure — no `old()` gymnastics

In a classic PHP/Laravel-style request/response form, a failed validation
means the whole page reloads and you must manually re-populate every field
from `old('field')` plus render error bags keyed by field name, or the user
loses everything they typed.

GoUI does not have this problem **by construction**, because the component
*is* the state — there is no request/response cycle to lose data across.
When a `save`/`submit` event fails validation:

1. `HandleEvent` calls `forms.ValidateAll(...)`.
2. `ValidateAll` returns `false`. Every field's own `Value` (or `Checked`,
   `Values`, etc.) is **already** whatever the user last typed — it was set
   incrementally by each field's own `HandleEvent` (`g-change`/`g-input`)
   as the user interacted with it, well before the submit event ever fired.
3. Each field's `Errors []string` is now populated (from step 1's
   `Validate()` calls).
4. You call `MarkDirty()` and return `nil` (or your own error) — **no**
   value needs to be copied back into anything.
5. The next `Render()` shows the exact same values the user typed, now with
   the field's own inline error(s) rendered right after it, and
   `aria-invalid`/error-border styling applied automatically by
   `ApplyErrorState`.

There is no separate "flash" store, no `old()` helper, and no risk of a
stale/expired session losing form input — the value lives in the same Go
struct for the entire lifetime of the WebSocket session (and survives
reconnects within the grace period, see
[`04-sessions-and-websocket.md`](04-sessions-and-websocket.md)).

## 4. Full form example

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
    // Sub-fields don't get SetTranslator automatically from ws.Session —
    // only the top-level BaseComponent does, so wire each field explicitly.
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
            // Nothing to restore: Name.Value, Email.Value, Message.Value are
            // already what the user typed. Each field's Errors is populated.
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

Notice what is absent: there is no code anywhere that copies a submitted
value back into `Name.Value` after a failed `save` — it was never removed
in the first place. The `Errors` slice on each field is the only thing
validation *adds* to the existing state.

For the fully wired, runnable version of this exact pattern (registry,
translator, hub, HTML page, broadcast endpoint), see
`examples/contact-form/main.go` and run it with:

```bash
go run ./examples/contact-form
# http://localhost:3001
```
