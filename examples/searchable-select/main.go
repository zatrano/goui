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

func cityItems() []forms.SelectItem {
	return []forms.SelectItem{
		{Value: "ist", Label: "İstanbul"},
		{Value: "ank", Label: "Ankara"},
		{Value: "izm", Label: "İzmir"},
		{Value: "bur", Label: "Bursa"},
		{Value: "ant", Label: "Antalya"},
		{Value: "ada", Label: "Adana"},
		{Value: "gaz", Label: "Gaziantep"},
		{Value: "kon", Label: "Konya"},
	}
}

// Tier2ADemo hosts all Alt-Grup A controls for manual verification.
type Tier2ADemo struct {
	core.BaseComponent
	Searchable forms.SearchableSelect
	Multi      forms.MultiSelect
	Combo      forms.Combobox
	Auto       forms.Autocomplete
	Tags       forms.TagInput
	Tree       forms.TreeSelect
	Cascade    forms.Cascader
	Dual       forms.DualListbox
}

func NewTier2ADemo() *Tier2ADemo {
	return &Tier2ADemo{
		Searchable: forms.SearchableSelect{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "city", ID: "city"},
				Placeholder: "Şehir seçin",
				Items:       cityItems(),
			},
			EventName: "city",
		},
		Multi: forms.MultiSelect{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "cities", ID: "cities"},
				Placeholder: "Şehirler",
				Items:       cityItems(),
			},
			EventName: "cities",
		},
		Combo: forms.Combobox{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "role", ID: "role"},
				Placeholder: "Rol yazın veya seçin",
				Items: []forms.SelectItem{
					{Value: "admin", Label: "Admin"},
					{Value: "editor", Label: "Editor"},
					{Value: "viewer", Label: "Viewer"},
				},
			},
			EventName: "role",
		},
		Auto: forms.Autocomplete{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "suggest", ID: "suggest"},
				Placeholder: "Öneri ara...",
				Items:       cityItems(),
			},
			EventName: "suggest",
		},
		Tags: forms.TagInput{
			CommonAttrs: forms.CommonAttrs{Name: "skills", ID: "skills"},
			Placeholder: "Etiket ekle (Enter/blur)",
			EventName:   "skills",
		},
		Tree: forms.TreeSelect{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "dept", ID: "dept"},
				Placeholder: "Departman",
			},
			EventName: "dept",
			Nodes: []forms.TreeNode{
				{
					Value: "eng", Label: "Engineering",
					Children: []forms.TreeNode{
						{Value: "be", Label: "Backend"},
						{Value: "fe", Label: "Frontend"},
					},
				},
				{
					Value: "ops", Label: "Operations",
					Children: []forms.TreeNode{
						{Value: "sre", Label: "SRE"},
						{Value: "sup", Label: "Support"},
					},
				},
			},
		},
		Cascade: forms.Cascader{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "loc", ID: "loc"},
				Placeholder: "Ülke / şehir",
				Items: []forms.SelectItem{
					{Value: "tr", Label: "Türkiye"},
					{Value: "de", Label: "Almanya"},
				},
			},
			EventName: "loc",
			LoadChildren: func(level int, parent string) []forms.SelectItem {
				if level != 0 {
					return nil
				}
				switch parent {
				case "tr":
					return []forms.SelectItem{{Value: "ist", Label: "İstanbul"}, {Value: "ank", Label: "Ankara"}}
				case "de":
					return []forms.SelectItem{{Value: "ber", Label: "Berlin"}, {Value: "muc", Label: "Münih"}}
				}
				return nil
			},
		},
		Dual: forms.DualListbox{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "perms", ID: "perms"},
				Placeholder: "Ara...",
				Items:       cityItems(),
			},
			EventName: "perms",
		},
	}
}

func (d *Tier2ADemo) Mount(_ context.Context) error   { return nil }
func (d *Tier2ADemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2ADemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	var err error
	switch {
	case strings.HasPrefix(event, "city."):
		err = d.Searchable.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "cities."):
		err = d.Multi.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "role."):
		err = d.Combo.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "suggest."):
		err = d.Auto.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "skills."):
		err = d.Tags.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "dept."):
		err = d.Tree.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "loc."):
		err = d.Cascade.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "perms."):
		err = d.Dual.HandleEvent(ctx, event, payload)
	}
	d.MarkDirty()
	return err
}

func section(title, body string) string {
	return `<section class="demo-section"><h2>` + html.EscapeString(title) + `</h2>` + body + `</section>`
}

func (d *Tier2ADemo) Render() (string, error) {
	parts := make([]string, 0, 8)

	s, err := d.Searchable.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("1. Searchable Select", s+`<p class="hint">Server-side filtre. Seçili: <strong>`+html.EscapeString(d.Searchable.Value)+`</strong></p>`))

	s, err = d.Multi.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("2. Multi Select", s+`<p class="hint">Seçili: <strong>`+html.EscapeString(d.Multi.RawValue())+`</strong></p>`))

	s, err = d.Combo.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("3. Combobox", s+`<p class="hint">Değer: <strong>`+html.EscapeString(d.Combo.Value)+`</strong></p>`))

	s, err = d.Auto.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("4. Autocomplete", s+`<p class="hint">Değer: <strong>`+html.EscapeString(d.Auto.Value)+`</strong></p>`))

	s, err = d.Tags.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("5. Tag Input / Chips", s+`<p class="hint">Etiketler: <strong>`+html.EscapeString(d.Tags.RawValue())+`</strong></p>`))

	s, err = d.Tree.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("6. Tree Select", s+`<p class="hint">Seçili: <strong>`+html.EscapeString(d.Tree.Value)+`</strong></p>`))

	s, err = d.Cascade.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("7. Cascader", s+`<p class="hint">Yol: <strong>`+html.EscapeString(d.Cascade.RawValue())+`</strong></p>`))

	s, err = d.Dual.Render()
	if err != nil {
		return "", err
	}
	parts = append(parts, section("8. Dual Listbox", s+`<p class="hint">Seçili: <strong>`+html.EscapeString(d.Dual.RawValue())+`</strong></p>`))

	return `<div class="demo"><h1>Select ailesi</h1><p class="lead">Arama/filtreleme varsayılan olarak server-side.</p>` + strings.Join(parts, "") + `</div>`, nil
}

func main() {
	root := repoRoot()
	registry := core.NewRegistry()
	if err := registry.Register("city", func() core.Component { return NewTier2ADemo() }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "searchable-select", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator())})

	log.Println("GoUI searchable-select demo at http://localhost:3002")
	log.Fatal(app.Listen(":3002"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
