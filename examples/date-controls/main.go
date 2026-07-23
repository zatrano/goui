package main

import (
	"context"
	"html"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	gouifiber "github.com/zatrano/goui/adapters/fiber"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

type Tier2BDemo struct {
	core.BaseComponent
	Leave forms.DateRangePicker
	Shift forms.TimeRangePicker
	Day   forms.CalendarDatePicker
}

func NewTier2BDemo() *Tier2BDemo {
	return &Tier2BDemo{
		Leave: forms.DateRangePicker{
			CommonAttrs: forms.CommonAttrs{Name: "leave", ID: "leave"},
			Start:       "2026-07-10",
			End:         "2026-07-15",
			EventName:   "leave",
		},
		Shift: forms.TimeRangePicker{
			CommonAttrs: forms.CommonAttrs{Name: "shift", ID: "shift"},
			Start:       "09:00",
			End:         "17:30",
			EventName:   "shift",
		},
		Day: forms.CalendarDatePicker{
			CommonAttrs: forms.CommonAttrs{Name: "day", ID: "day"},
			Value:       "2026-07-16",
			Placeholder: "Tarih seçin",
			EventName:   "day",
		},
	}
}

func (d *Tier2BDemo) Mount(_ context.Context) error   { return nil }
func (d *Tier2BDemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2BDemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	var err error
	switch {
	case strings.HasPrefix(event, "leave."):
		err = d.Leave.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "shift."):
		err = d.Shift.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "day."):
		err = d.Day.HandleEvent(ctx, event, payload)
	}
	d.MarkDirty()
	return err
}

func section(title, body string) string {
	return `<section class="demo-section"><h2>` + html.EscapeString(title) + `</h2>` + body + `</section>`
}

func (d *Tier2BDemo) Render() (string, error) {
	leaveL, _ := (&forms.Label{For: "leave", Text: "İzin (Date Range)"}).Render()
	leave, err := d.Leave.Render()
	if err != nil {
		return "", err
	}
	shiftL, _ := (&forms.Label{For: "shift", Text: "Vardiya (Time Range)"}).Render()
	shift, err := d.Shift.Render()
	if err != nil {
		return "", err
	}
	dayL, _ := (&forms.Label{For: "day", Text: "Gün (Calendar)"}).Render()
	day, err := d.Day.Render()
	if err != nil {
		return "", err
	}

	parts := []string{
		section("1. Date Range Picker", leaveL+leave+`<p class="hint">Aralık: <strong>`+html.EscapeString(d.Leave.RawValue())+`</strong></p>`),
		section("2. Time Range Picker", shiftL+shift+`<p class="hint">Aralık: <strong>`+html.EscapeString(d.Shift.RawValue())+`</strong></p>`),
		section("3. Calendar Date Picker", dayL+day+`<p class="hint">Seçili: <strong>`+html.EscapeString(d.Day.Value)+`</strong> — ay gezinme client-side</p>`),
	}
	return `<div class="demo"><h1>Tarih/Zaman</h1><p class="lead">Range native input; görsel takvimde ay UI-only, seçim server'a gider.</p>` + strings.Join(parts, "") + `</div>`, nil
}

func main() {
	root := repoRoot()
	registry := core.NewRegistry()
	if err := registry.Register("dates", func() core.Component { return NewTier2BDemo() }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "date-controls", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator())})

	log.Println("GoUI date-controls demo at http://localhost:3005")
	log.Fatal(app.Listen(":3005"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
