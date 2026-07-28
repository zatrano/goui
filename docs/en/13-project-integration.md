# 13. Project Integration

This is the most important chapter in these docs: it walks through dropping
GoUI into a real, already-existing HTTP application — routing, adapter
selection, multi-tenancy, PostgreSQL, building a new component from scratch,
composing a realistic multi-tier form, and running it in production without Docker.

Module path used throughout this document: `github.com/zatrano/goui`.

```go
import (
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/validation"
	"github.com/zatrano/goui/ws"
)
```

---

## 1. Integrating into an existing HTTP application

GoUI does not want its own HTTP server or its own app struct — it registers a
handful of routes on the router you already have, next to whatever routes
your application already serves (REST APIs, server-rendered pages, health
checks, etc.). There are exactly three things to wire up:

1. **Static assets** — the client runtime (`client/goui.js` and
   `client/modules/*.js`) and the base stylesheet (`forms/style.css`) are
   plain files; serve them with your framework's static-file middleware or
   `http.FileServer`.
2. **The WebSocket endpoint** — build `ws.NewServer(hub, registry, tr)` and
   mount `GET /goui/ws` (`ws.Path`) through an adapter.
3. **The upload endpoints** (only if you use `forms.DragDropUpload` /
   `forms.AvatarUpload`) — pass an `upload.Storage` as the adapter's
   `Store` option, or call `upload.Mount` on a `net/http` mux.

### 1.1 Adapter selection

| Stack | Module | Register |
|---|---|---|
| Fiber v3 | `github.com/zatrano/goui/adapters/fiber` | `gouifiber.Register(app, opts)` |
| `net/http` | `github.com/zatrano/goui/adapters/stdlib` | `gouistdlib.Register(mux, opts)` |
| Chi | `github.com/zatrano/goui/adapters/stdlib` | `gouistdlib.Mount(chiRouter, opts)` |
| Gin | `github.com/zatrano/goui/adapters/gin` | `gouigin.Register(r, opts)` |
| Echo | `github.com/zatrano/goui/adapters/echo` | `gouiecho.Register(e, opts)` |

Every adapter accepts the same options shape:

```go
type Options struct {
    Server *ws.Server      // required for WebSocket
    Store  upload.Storage  // optional — mounts POST /goui/upload and GET /goui/files/:id
}
```

Proof examples: `examples/adapters/{nethttp,chi,gin,echo}`. The main demos
under `examples/` use the Fiber adapter.

### 1.2 Snippets by stack

**Fiber** (also used by the shipped demos):

```go
import gouifiber "github.com/zatrano/goui/adapters/fiber"

hub := ws.NewHub()
server := ws.NewServer(hub, registry, tr)
store, _ := upload.NewLocalStore("./data/uploads", "/goui/files", 8<<20)

gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})
```

**net/http `ServeMux`:**

```go
import gouistdlib "github.com/zatrano/goui/adapters/stdlib"

gouistdlib.Register(mux, gouistdlib.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

**Chi** (via stdlib adapter `Mount`):

```go
import gouistdlib "github.com/zatrano/goui/adapters/stdlib"

gouistdlib.Mount(chiRouter, gouistdlib.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

**Gin:**

```go
import gouigin "github.com/zatrano/goui/adapters/gin"

gouigin.Register(r, gouigin.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

**Echo:**

```go
import gouiecho "github.com/zatrano/goui/adapters/echo"

gouiecho.Register(e, gouiecho.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

### 1.3 Full Fiber integration example

The example below matches a typical existing Fiber v3 app; swap
`gouifiber.Register` for your adapter from §1.2 if you use a different stack.

```go
package main

import (
	"log"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	gouifiber "github.com/zatrano/goui/adapters/fiber"
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/ws"

	"myapp/internal/httpapi" // your existing REST handlers, unrelated to GoUI
)

func main() {
	app := fiber.New()

	// --- your existing routes, untouched ---
	app.Use(myAuthMiddleware())
	httpapi.RegisterRoutes(app)

	// --- GoUI wiring lives next to them ---
	gouiRoot := "./vendor/goui" // wherever you vendor/checkout the goui module's client+forms assets
	app.Use("/client", static.New(filepath.Join(gouiRoot, "client")))
	app.Use("/forms", static.New(filepath.Join(gouiRoot, "forms"))) // forms/style.css

	tr := i18n.NewTranslator()
	_ = tr.LoadLocale("en", "./locales/en.json")
	_ = tr.LoadLocale("tr", "./locales/tr.json")

	registry := core.NewRegistry()
	mustRegisterComponents(registry, tr /*, db, etc. */)

	hub := ws.NewHub()
	server := ws.NewServer(hub, registry, tr)

	store, err := upload.NewLocalStore("./data/uploads", "/goui/files", 8<<20)
	if err != nil {
		log.Fatal(err)
	}

	gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})

	// A page that boots a GoUI component in the browser is just a normal
	// handler that serves HTML with a <script type="module"> bootstrap —
	// see the compose-form example in §5 for the bootstrap script itself.
	app.Get("/reseller/register", func(c fiber.Ctx) error {
		return c.SendFile("./views/reseller_register.html")
	})

	log.Fatal(app.Listen(":8080"))
}
```

Everything GoUI needs at runtime is a populated `*core.Registry`, a
`*ws.Server`, optional `upload.Storage`, static asset routes, and one adapter
`Register`/`Mount` call. Nothing about GoUI requires its own process, port, or
reverse proxy path beyond `/goui/*` and whatever static prefixes you chose
for `/client` and `/forms`.

> Note on the `/client` and `/forms` prefixes: they're your choice — mount
> those directories at whatever paths your HTTP stack uses for static files.
> Keep them stable once chosen, since the HTML you serve and `input.css`'s
> `@import` path (see [12-theming-and-tailwind.md](12-theming-and-tailwind.md))
> both need to agree with them.

---

## 2. Multi-tenant pattern — without touching Session code

A very common real-world shape: an organizational hierarchy like
**Holding → Company → Department → User**, where every dashboard/component
must only ever see and mutate data scoped to the tenant the logged-in user
belongs to. GoUI's `ws.Session` intentionally has **no concept of tenancy at
all** — and you should not add one by modifying `ws/session.go`. Session's
job is transport (frames, reconnection, prefetch bookkeeping); tenancy is
application data, and it belongs on your components, not on the transport.

### 2.1 Why `context.Context` alone doesn't carry this

It's tempting to reach for `ctx` inside `Mount(ctx context.Context)` and
expect your own Fiber middleware's `c.Locals("tenantID")` to somehow be
there. It won't be: `Session.prepareComponent` always calls
`c.Mount(context.Background())` — the WS upgrade handshake happens on its
own connection lifecycle, decoupled from whatever HTTP request context
existed when the page that *bootstraps* the WebSocket was served. There is
exactly one hand-off point between "your app's authenticated HTTP request"
and "a GoUI component instance," and it is the **registry component name**
requested when the browser opens the socket
(`?component=<name>` in the WS URL, or the `data-goui-prefetch`/
`data-goui-activate` attribute value). Design your multi-tenancy around that
hand-off point.

### 2.2 The pattern: tenant-qualified registry names + factory closures

Two ingredients:

1. **Register one factory per tenant scope**, capturing the tenant IDs (and
   whatever DB handle/services the component needs) as closure variables —
   this is exactly the "store tenant ID on component fields set in factory"
   approach: the `Component` instance that comes back from
   `registry.Create(name)` already has `HoldingID`/`CompanyID`/`DeptID`/
   `UserID` populated, before `Mount` is ever called.
2. **Encode the tenant scope into the registry name itself**, and have your
   own (existing, unmodified) authenticated page handler render that name
   into the bootstrap script it serves — this is the "context values set by
   your middleware before Mount" part: your middleware runs on the *page*
   request, well before the WS upgrade, and its decision (which tenant) is
   baked into the name the browser will ask for.

```go
// internal/dashboard/component.go
package dashboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zatrano/goui/core"
)

// DeptDashboard is scoped to exactly one department at construction time.
// Nothing about ws.Session or ws.Hub needs to know these fields exist.
type DeptDashboard struct {
	core.BaseComponent

	db *pgxpool.Pool

	HoldingID string
	CompanyID string
	DeptID    string
	UserID    string

	Employees []employeeRow
}

type employeeRow struct {
	ID, FullName, Role string
}
```

Registering tenant-scoped factories, lazily and idempotently, from your
**own** authenticated request handler (no changes to `ws` or `core`
required — `Register` simply refuses a duplicate name):

```go
// internal/dashboard/register.go
package dashboard

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zatrano/goui/core"
)

// componentName builds the registry-name hand-off point described in §2.1.
// Any deterministic, collision-free encoding of the tenant path works —
// this one is just "dept-dashboard:<deptID>".
func componentName(deptID string) string {
	return fmt.Sprintf("dept-dashboard:%s", deptID)
}

// EnsureRegistered lazily registers a tenant-scoped factory the first time
// a given department is seen. Safe to call on every request; registering
// twice is a no-op (ErrComponentAlreadyRegistered is swallowed).
func EnsureRegistered(registry *core.Registry, db *pgxpool.Pool, holdingID, companyID, deptID, userID string) string {
	name := componentName(deptID)
	err := registry.Register(name, func() core.Component {
		return &DeptDashboard{
			db:        db,
			HoldingID: holdingID,
			CompanyID: companyID,
			DeptID:    deptID,
			UserID:    userID,
		}
	})
	if err != nil && !errors.Is(err, core.ErrComponentAlreadyRegistered) {
		panic(err) // registration errors other than "already registered" indicate a bug
	}
	return name
}
```

Your existing authenticated page handler (untouched aside from calling
`EnsureRegistered` and passing the resulting name to the template) is the
one place tenancy and GoUI actually meet:

```go
app.Get("/dashboard", requireLogin(), func(c fiber.Ctx) error {
	user := currentUser(c) // your existing auth, unrelated to GoUI
	name := dashboard.EnsureRegistered(registry, db,
		user.HoldingID, user.CompanyID, user.DeptID, user.ID)

	return c.Render("dashboard", fiber.Map{
		"ComponentName": name, // e.g. "dept-dashboard:D-42"
	})
})
```

```html
<!-- dashboard.html, rendered server-side with ComponentName injected -->
<div id="app"></div>
<script type="module">
  import { GoUIClient } from '/client/goui.js';
  const client = new GoUIClient('/goui/ws', '{{.ComponentName}}', { mount: '#app', locale: 'en' });
  client.connect();
</script>
```

When the browser opens the socket with `?component=dept-dashboard:D-42`,
`registry.Create` runs *that specific department's* factory closure — the
component that comes out already knows exactly which holding, company,
department, and user it belongs to, with **zero changes to `ws/session.go`,
`ws/hub.go`, or `ws/server.go`.** `Mount` and `HandleEvent` then simply use
`d.HoldingID`/`d.CompanyID`/`d.DeptID`/`d.UserID` as SQL parameters (§3) —
they never need to consult `ctx` for tenancy, because it's already a field.

### 2.3 Narrative recap: Holding → Company → Dept → User

- **Holding** — the top-level legal entity; owns many Companies.
- **Company** — one legal/operating business unit under a Holding; owns
  many Departments.
- **Dept** — an organizational unit under a Company (Sales, Support,
  Engineering, ...); has many Users and is the natural "workspace" scope
  for most day-to-day dashboards.
- **User** — an individual with a role inside exactly one Department (in
  this simplified model — real systems often allow cross-department roles,
  which just means capturing more IDs in the same closure).

Every level of that hierarchy is just another field captured by the
factory closure and another `WHERE` clause parameter in your SQL (§3). You
can register components scoped at whichever level makes sense for a given
view — a company-wide rollup dashboard keyed by `CompanyID` alone, a
department dashboard keyed by `DeptID`, or a personal "my tasks" component
keyed by `UserID` — the pattern is identical at every level; only the
registry-name encoding and the SQL `WHERE` clause change.

### 2.4 Cleaning up registrations

`*core.Registry` has no `Unregister`; entries accumulate for the lifetime of
the process. For a long-running multi-tenant server with a large or
unbounded number of departments, prefer registering **role-level** names
(`"dept-dashboard"`) that read the *specific* department ID out of something
else stable per-connection — such as a short-lived, signed token embedded
in the component name (e.g. `"dept-dashboard:" + signedDeptToken`), verified
and decoded inside the factory before returning the component — rather than
registering one literal name per department forever. Whether that's worth
the added complexity depends entirely on your tenant cardinality; for
tens or low hundreds of departments, the straightforward lazy-registration
shown above is simpler and entirely adequate.

---

## 3. PostgreSQL: reading in `Mount`, writing in `HandleEvent`

Continuing the `DeptDashboard` example — this SQL is illustrative (adjust
to your schema/driver), but the shape — read on `Mount`, validate+write on
the event that submits/changes something — is the pattern to follow for any
component backed by a real database.

```go
func (d *DeptDashboard) Mount(ctx context.Context) error {
	rows, err := d.db.Query(ctx, `
		SELECT id, full_name, role
		FROM employees
		WHERE holding_id = $1 AND company_id = $2 AND dept_id = $3
		ORDER BY full_name
	`, d.HoldingID, d.CompanyID, d.DeptID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var e employeeRow
		if err := rows.Scan(&e.ID, &e.FullName, &e.Role); err != nil {
			return err
		}
		d.Employees = append(d.Employees, e)
	}
	return rows.Err()
}

func (d *DeptDashboard) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch event {
	case "promote":
		employeeID, _ := payload["value"].(string)
		if employeeID == "" {
			return nil
		}
		// Always re-scope every write by the tenant fields captured at
		// construction time — never trust anything from payload for tenancy.
		tag, err := d.db.Exec(ctx, `
			UPDATE employees
			SET role = 'lead'
			WHERE id = $1 AND holding_id = $2 AND company_id = $3 AND dept_id = $4
		`, employeeID, d.HoldingID, d.CompanyID, d.DeptID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			// Either the ID doesn't exist, or (more importantly) it belongs
			// to a different tenant — treat both the same: no-op, no leak.
			d.ToastT("error", "dashboard.promote_denied")
			return nil
		}
		for i := range d.Employees {
			if d.Employees[i].ID == employeeID {
				d.Employees[i].Role = "lead"
			}
		}
		d.ToastT("success", "dashboard.promote_ok")
		d.MarkDirty()
	}
	return nil
}
```

The critical security property: **every write's `WHERE` clause repeats the
tenant fields**, not just the row's primary key. Because those fields came
from the factory closure (§2.2) rather than from the WS event payload, a
malicious client cannot widen its own scope by tampering with the JSON it
sends — the worst it can do is name an `employeeID` that doesn't belong to
its own department, which the `RowsAffected() == 0` check turns into a
harmless no-op instead of a cross-tenant mutation.

---

## 4. Building a new component, step by step

1. **Decide what state the component owns.** Keep it to exactly what
   `Render()` needs plus whatever `HandleEvent` mutates — nothing else.

2. **Embed `core.BaseComponent`** for `MarkDirty`/`IsDirty`/`ResetDirty`,
   `T`/`ToastT`, and the translator/pusher plumbing the session wires up
   automatically.

3. **Implement the four `core.Component` methods:**

   ```go
   type Counter struct {
       core.BaseComponent
       Value int
       Step  int
   }

   func NewCounter(step int) *Counter {
       if step == 0 {
           step = 1
       }
       return &Counter{Step: step}
   }

   func (c *Counter) Mount(_ context.Context) error   { return nil } // no external state to load
   func (c *Counter) Unmount(_ context.Context) error { return nil } // nothing to release

   func (c *Counter) HandleEvent(_ context.Context, event string, _ map[string]any) error {
       switch event {
       case "inc":
           c.Value += c.Step
       case "dec":
           c.Value -= c.Step
       default:
           return nil // unknown events are ignored, not errors
       }
       c.MarkDirty()
       return nil
   }

   func (c *Counter) Render() (string, error) {
       return fmt.Sprintf(
           `<div data-goui-ignore="false"><span>%d</span> `+
               `<button type="button" g-click="dec">-</button> `+
               `<button type="button" g-click="inc">+</button></div>`,
           c.Value,
       ), nil
   }
   ```

   Remember from [10-diffing-internals.md](10-diffing-internals.md) §3:
   `Render()` should return **exactly one root element** so GoUI never needs
   to inject a synthetic wrapper `<div>`.

4. **Register a factory, not an instance:**

   ```go
   registry.Register("counter", func() core.Component { return NewCounter(1) })
   ```

   `Register` fails with `core.ErrComponentAlreadyRegistered` on a duplicate
   name and `Create` fails with `core.ErrComponentNotRegistered` for an
   unknown one — check for these where you register/activate dynamically
   (as in §2.4).

5. **If the component needs translated text or toasts**, call
   `c.SetTranslator(tr)` on construction (or let the factory close over
   `tr` and set it before returning) — `Session.prepareComponent` also
   calls `SetTranslator`/`SetPusher` automatically on every `Mount`/
   `Activate`, so this step is mostly relevant if you want translated
   strings to be available even in tests that construct the component
   directly without a session.

6. **Wire up the composed HTML.** For anything beyond trivial markup,
   either build it with plain string concatenation/`strings.Builder` (the
   convention used throughout `forms/*`), or use `core.RenderTemplate` for
   more template-like ergonomics:

   ```go
   html, err := core.RenderTemplate(`
     <div>
       <span>{{.Value}}</span>
       <button type="button" g-click="dec">-</button>
       <button type="button" g-click="inc">+</button>
     </div>`, c)
   ```

   `RenderTemplate` caches parsed templates by a hash of the template
   string, so calling it every `Render()` with the same literal template
   text does not re-parse on every call.

7. **Write a test that exercises `HandleEvent` → `Render` directly**,
   without a `Session` at all — components are plain Go values with no
   hidden dependency on the WS layer:

   ```go
   func TestCounter_Increment(t *testing.T) {
       c := NewCounter(1)
       if err := c.HandleEvent(context.Background(), "inc", nil); err != nil {
           t.Fatal(err)
       }
       if !c.IsDirty() {
           t.Fatal("expected dirty after inc")
       }
       html, err := c.Render()
       if err != nil || !strings.Contains(html, ">1<") {
           t.Fatalf("html = %q, err = %v", html, err)
       }
   }
   ```

8. **Mount it in a page** by giving the browser its registry name (directly,
   or via the tenant-qualified pattern from §2) and, if it's a secondary
   view, consider `data-goui-prefetch`/`data-goui-activate`
   ([09-prefetch.md](09-prefetch.md)) instead of a plain link.

---

## 5. Compose form: Tier 1 + Tier 2 + validation — "3CX reseller registration"

A realistic example that combines plain fields (Tier 1: `TextInput`,
`Select`, `ChoiceInput`), a composed Tier 2 control (`forms.PhoneInput`,
which itself wires a `SearchableSelect` dial-code picker plus a
`TextInput`), and server-side validation via `forms.ValidateAll` — the same
pattern `examples/contact-form` uses, scaled up to a form with more fields
and cross-field concerns.

```go
package reseller

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/validation"
)

// RegistrationForm is a "become a 3CX reseller" application form:
// company identity, contact channel, and program tier selection.
type RegistrationForm struct {
	core.BaseComponent

	db *pgxpool.Pool

	CompanyName forms.TextInput
	Website     forms.TextInput
	Email       forms.TextInput
	Phone       *forms.PhoneInput
	Country     forms.SearchableSelect
	Tier        forms.Select
	MonthlySeats forms.NumericInput
	AgreeTerms  forms.ChoiceInput

	Submitted bool
	Summary   string
}

func NewRegistrationForm(db *pgxpool.Pool, tr *i18n.Translator) *RegistrationForm {
	f := &RegistrationForm{
		db: db,
		CompanyName: forms.TextInput{
			CommonAttrs:     forms.CommonAttrs{Name: "company_name", ID: "company_name", Required: true},
			Type:            "text",
			Placeholder:     "Acme Telecom Ltd.",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required(), validation.MinLength(2)}},
		},
		Website: forms.TextInput{
			CommonAttrs: forms.CommonAttrs{Name: "website", ID: "website"},
			Type:        "url",
			Placeholder: "https://acmetelecom.example",
		},
		Email: forms.TextInput{
			CommonAttrs:     forms.CommonAttrs{Name: "email", ID: "email", Required: true},
			Type:            "email",
			Placeholder:     "sales@acmetelecom.example",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required(), validation.Email()}},
		},
		Phone: forms.NewPhoneInput("phone"), // Tier 2: dial-code SearchableSelect + national number TextInput
		Country: forms.SearchableSelect{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "country", ID: "country", Required: true},
				Placeholder: "Search country…",
				Items:       countryItems(), // []forms.SelectItem
			},
			EventName: "country",
		},
		Tier: forms.Select{
			CommonAttrs: forms.CommonAttrs{Name: "tier", ID: "tier", Required: true},
			Options: []forms.Option{
				{Value: "", Label: "Select a reseller tier"},
				{Value: "silver", Label: "Silver — up to 50 seats/mo"},
				{Value: "gold", Label: "Gold — up to 250 seats/mo"},
				{Value: "platinum", Label: "Platinum — unlimited"},
			},
		},
		MonthlySeats: forms.NumericInput{
			CommonAttrs: forms.CommonAttrs{Name: "monthly_seats", ID: "monthly_seats"},
			Type:        "number",
			Min:         "1",
			Step:        "1",
		},
		AgreeTerms: forms.ChoiceInput{
			CommonAttrs:     forms.CommonAttrs{Name: "agree_terms", ID: "agree_terms", Required: true},
			Type:            "checkbox",
			Value:           "yes",
			LabelText:       "I agree to the 3CX Reseller Program terms",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required()}},
		},
	}
	for _, field := range []interface{ SetTranslator(*i18n.Translator) }{
		&f.CompanyName, &f.Website, &f.Email, &f.Country, &f.Tier, &f.MonthlySeats, &f.AgreeTerms,
	} {
		field.SetTranslator(tr)
	}
	f.Phone.Number.SetTranslator(tr)
	f.SetTranslator(tr)
	return f
}

func (f *RegistrationForm) Mount(_ context.Context) error   { return nil }
func (f *RegistrationForm) Unmount(_ context.Context) error { return nil }

func (f *RegistrationForm) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch {
	case event == "company_name":
		return f.CompanyName.HandleEvent(ctx, event, payload)
	case event == "website":
		return f.Website.HandleEvent(ctx, event, payload)
	case event == "email":
		return f.Email.HandleEvent(ctx, event, payload)
	case event == "country" || hasPrefix(event, "country."):
		return f.Country.HandleEvent(ctx, event, payload)
	case event == "tier":
		return f.Tier.HandleEvent(ctx, event, payload)
	case event == "monthly_seats":
		return f.MonthlySeats.HandleEvent(ctx, event, payload)
	case event == "agree_terms":
		return f.AgreeTerms.HandleEvent(ctx, event, payload)
	case hasPrefix(event, "phone_dial") || hasPrefix(event, "phone_num"):
		return f.Phone.HandleEvent(ctx, event, payload)
	case event == "register":
		return f.submit(ctx)
	}
	return nil
}

func (f *RegistrationForm) submit(ctx context.Context) error {
	// Cross-field rule: Platinum tier requires a monthly seat estimate.
	seats, _ := strconv.Atoi(f.MonthlySeats.Value)
	if f.Tier.Value == "platinum" && seats <= 0 {
		f.MonthlySeats.Errors = []string{f.T("reseller.seats_required_for_platinum")}
	} else {
		f.MonthlySeats.Errors = nil
	}

	ok := forms.ValidateAll(
		&f.CompanyName, &f.Email, &f.Country, &f.Tier, &f.AgreeTerms, f.Phone,
	)
	if len(f.MonthlySeats.Errors) > 0 {
		ok = false
	}
	if !ok {
		f.Submitted = false
		f.MarkDirty()
		return nil
	}

	_, err := f.db.Exec(ctx, `
		INSERT INTO reseller_applications
			(company_name, website, email, phone, country, tier, monthly_seats)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, f.CompanyName.Value, f.Website.Value, f.Email.Value, f.Phone.RawValue(),
		f.Country.Value, f.Tier.Value, seats)
	if err != nil {
		f.ToastT("error", "reseller.submit_failed")
		return nil
	}

	f.Submitted = true
	f.Summary = f.CompanyName.Value + " — " + f.Tier.Value + " tier"
	f.ToastT("success", "reseller.submit_success")
	f.MarkDirty()
	return nil
}

func (f *RegistrationForm) Render() (string, error) {
	companyL, _ := (&forms.Label{For: "company_name", Text: f.T("reseller.company_name")}).Render()
	companyI, _ := f.CompanyName.Render()
	webL, _ := (&forms.Label{For: "website", Text: f.T("reseller.website")}).Render()
	webI, _ := f.Website.Render()
	emailL, _ := (&forms.Label{For: "email", Text: f.T("reseller.email")}).Render()
	emailI, _ := f.Email.Render()
	phoneL, _ := (&forms.Label{For: "phone", Text: f.T("reseller.phone")}).Render()
	phoneI, _ := f.Phone.Render()
	countryL, _ := (&forms.Label{For: "country", Text: f.T("reseller.country")}).Render()
	countryI, _ := f.Country.Render()
	tierL, _ := (&forms.Label{For: "tier", Text: f.T("reseller.tier")}).Render()
	tierI, _ := f.Tier.Render()
	seatsL, _ := (&forms.Label{For: "monthly_seats", Text: f.T("reseller.monthly_seats")}).Render()
	seatsI, _ := f.MonthlySeats.Render()
	agreeI, _ := f.AgreeTerms.Render()
	btn, _ := (&forms.Button{Type: "button", Text: f.T("reseller.submit"), EventName: "register"}).Render()

	result := ""
	if f.Submitted {
		o, _ := (&forms.Output{CommonAttrs: forms.CommonAttrs{Name: "summary"}, Text: f.Summary}).Render()
		result = `<div class="result">` + o + `</div>`
	}

	inner := forms.JoinHTML(
		`<div class="field">`, companyL, companyI, `</div>`,
		`<div class="field">`, webL, webI, `</div>`,
		`<div class="field">`, emailL, emailI, `</div>`,
		`<div class="field">`, phoneL, phoneI, `</div>`,
		`<div class="field">`, countryL, countryI, `</div>`,
		`<div class="field">`, tierL, tierI, `</div>`,
		`<div class="field">`, seatsL, seatsI, `</div>`,
		`<div class="field choice">`, agreeI, `</div>`,
		`<div class="actions">`, btn, `</div>`,
		result,
	)
	return (&forms.Form{Method: "post", OnSubmit: "register", InnerHTML: inner}).Render()
}

func hasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }

func countryItems() []forms.SelectItem {
	return []forms.SelectItem{
		{Value: "tr", Label: "Türkiye"},
		{Value: "de", Label: "Germany"},
		{Value: "us", Label: "United States"},
		{Value: "gb", Label: "United Kingdom"},
		// ...
	}
}
```

Registering and mounting it is identical to any other component:

```go
registry.Register("reseller-registration", func() core.Component {
	return reseller.NewRegistrationForm(db, tr)
})
```

Notes on the composition:

- **Tier 1 fields** (`TextInput`, `Select`, `ChoiceInput`, `NumericInput`)
  each own their own `FieldValidation` and forward their own WS event to
  their own `HandleEvent`; the parent form just routes by event name.
- **`forms.PhoneInput`** is a Tier 2 *composition helper*, not a new
  control family — it internally owns a `SearchableSelect` (dial code) and
  a `TextInput` (national number) and exposes a single `RawValue()`
  (`"+90 5xx..."`) and a single `Validate()` that validates both children.
  Its two internal events are prefixed (`phone_dial`, `phone_num`), which is
  why the parent's `HandleEvent` dispatches by prefix rather than exact
  match.
- **Cross-field validation** (Platinum tier requiring a seat count) is just
  regular Go logic in `submit`, layered on top of `forms.ValidateAll` —
  `ValidateAll` handles the "did every individual field pass its own rules"
  half; anything spanning multiple fields is your component's own
  responsibility, exactly like `contact-form`'s existing per-field checks.
- **The actual `INSERT`** only runs after `ok` is confirmed true, and only
  ever reads validated field values — never raw, unvalidated event payload
  data — for the SQL parameters.

---

## 6. Production deployment — Dockerless

Everything below assumes a single static Linux binary built with
`go build`, no containers, running behind Nginx as a TLS-terminating
reverse proxy, supervised by systemd.

### 6.1 Build

```bash
GOOS=linux GOARCH=amd64 go build -o /opt/myapp/bin/myapp ./cmd/myapp
```

Copy the compiled binary plus whatever static assets your app serves
(`client/`, `forms/`, your own templates/locales) to the target host, e.g.
under `/opt/myapp/`.

### 6.2 systemd unit

```ini
# /etc/systemd/system/myapp.service
[Unit]
Description=MyApp (GoUI-based HTTP app)
After=network.target

[Service]
Type=simple
User=myapp
Group=myapp
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/bin/myapp
Restart=on-failure
RestartSec=2
Environment=PORT=8080
Environment=APP_ENV=production
# If your app reads a DATABASE_URL, secrets, etc., put them in an EnvironmentFile instead:
# EnvironmentFile=/opt/myapp/.env
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now myapp
sudo systemctl status myapp
```

`LimitNOFILE` is worth raising explicitly: every open WebSocket connection
holds a file descriptor for the process's lifetime, and the default
per-process limit on many distros (1024) is easy to exceed once you have a
few hundred concurrent users with a live tab open.

### 6.3 Nginx reverse proxy — the WebSocket-specific bits

The one thing that trips people up deploying any WebSocket app behind
Nginx: **the `Upgrade`/`Connection` headers are not proxied by default** and
must be forwarded explicitly, or the WS handshake at `/goui/ws` will fail
(typically visible client-side as a connection that never reaches `open`,
or an immediate close — see
[14-troubleshooting.md](14-troubleshooting.md) §1).

```nginx
# /etc/nginx/sites-available/myapp.conf

# Required once, usually in the http{} block or a shared snippet, so the
# $connection_upgrade map below can produce the right Connection header
# for both Upgrade and non-Upgrade requests on the same server block.
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 80;
    server_name myapp.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name myapp.example.com;

    ssl_certificate     /etc/letsencrypt/live/myapp.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/myapp.example.com/privkey.pem;

    # Everything else in your app (REST APIs, server-rendered pages, static
    # assets) — a plain reverse proxy, no special handling needed.
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # The GoUI WebSocket endpoint specifically needs the Upgrade dance.
    location /goui/ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSockets are long-lived; the default proxy timeouts (often 60s)
        # will silently kill idle connections. Raise them well above your
        # expected idle time (or above the grace period from §7, whichever
        # is larger) so Nginx doesn't disconnect sessions Nginx itself.
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

With this in place, the browser connects to `wss://myapp.example.com/goui/ws`
— TLS terminates at Nginx, and the traffic between Nginx and your Go
process is plain `ws://127.0.0.1:8080/goui/ws`. `client/goui.js` derives the
scheme itself:

```js
// client/goui.js
const base = this.wsUrl.startsWith('ws')
  ? this.wsUrl
  : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}${this.wsUrl}`;
```

so as long as your bootstrap script constructs `GoUIClient` with a
same-origin relative path (`'/goui/ws'`, as in every example in this
repository) rather than a hardcoded `ws://` URL, serving the page over
`https://` automatically upgrades the socket to `wss://` with no additional
client-side configuration.

### 6.4 TLS certificates

Any standard, Dockerless TLS setup works — most simply, Certbot's Nginx
plugin:

```bash
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot --nginx -d myapp.example.com
```

Certbot rewrites the `server { listen 443 ssl ... }` block for you and sets
up auto-renewal; the WS-specific `location /goui/ws { ... }` block above is
untouched by that process — just make sure it exists in the same
`server {}` block Certbot configures for TLS.

---

## 7. Performance in production

Three levers, all covered in depth elsewhere in these docs — summarized
here as a checklist for a real deployment:

- **Prefetch primary navigation.** Put `data-goui-prefetch`/
  `data-goui-activate` on links to views users are likely to open next
  (tabs, wizard steps, "view details" from a list) so their `Mount` has
  already run by the time the user clicks — see
  [09-prefetch.md](09-prefetch.md). Do **not** prefetch anything whose
  `Mount` has side effects, and remember the per-session cap is
  `ws.MaxPrefetch` (5), enforced via LRU eviction — prefetch is for a
  handful of "likely next" destinations, not a way to warm your entire
  navigation menu.

- **`data-key` on large/reorderable lists.** Any list backed by more than a
  handful of rows that can be filtered, sorted, or spliced in the middle
  (not just appended to) should carry a stable `data-key` on each row —
  otherwise every such change degrades into replacing everything from the
  first differing index onward. See
  [10-diffing-internals.md](10-diffing-internals.md) §5 and §7 for exactly
  how keyed diffing behaves and where it still has sharp edges (heavy
  simultaneous multi-item reorders).

- **Tune the reconnect grace period with `ws.NewHubWithGracePeriod`.** By
  default, `ws.NewHub()` uses `ws.DefaultGracePeriod` (60 seconds) — a
  disconnected session (laptop lid closed, brief network drop, mobile tab
  backgrounded) keeps its mounted components alive and its outbound queue
  buffering for that long, so a reconnect is a fast, transparent resume
  rather than a full remount. In production you generally want to reason
  about this explicitly rather than accept the default silently:

  ```go
  // A longer grace period for a mobile-heavy user base that frequently
  // backgrounds tabs, at the cost of holding mounted state (and DB
  // connections, goroutines, etc. per component) longer after a drop.
  hub := ws.NewHubWithGracePeriod(3 * time.Minute)
  server := ws.NewServer(hub, registry, tr)
  gouifiber.Register(app, gouifiber.Options{Server: server})
  ```

  Trade-off to make deliberately: a **longer** grace period means smoother
  reconnects (state and prefetch survive brief drops) but more server-side
  resources held per disconnected-but-not-yet-expired session; a
  **shorter** one frees resources faster but turns brief network blips into
  full remounts (fresh `Mount`, fresh first render, any prefetch progress
  lost). Whatever value you choose, also raise your Nginx
  `proxy_read_timeout`/`proxy_send_timeout` (§6.3) comfortably above it —
  there's no benefit to a generous application-level grace period if the
  reverse proxy in front of it drops idle connections sooner.
