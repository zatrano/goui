package forms

import (
	"context"
	"strings"
	"testing"
)

func TestTextInput_RenderAttrs(t *testing.T) {
	in := &TextInput{
		CommonAttrs: CommonAttrs{Name: "email", Required: true, Disabled: false},
		Type:        "email",
		Value:       "a@b.com",
		Placeholder: "Email",
		MaxLength:   100,
		Pattern:     `.+@.+\..+`,
	}
	html, err := in.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`type="email"`,
		`name="email"`,
		`value="a@b.com"`,
		`placeholder="Email"`,
		`required`,
		`maxlength="100"`,
		`g-change="email"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("Render missing %q in %s", want, html)
		}
	}
}

func TestTextInput_HandleEventOnChange(t *testing.T) {
	var got string
	in := &TextInput{
		CommonAttrs: CommonAttrs{Name: "name"},
		OnChange:    func(v string) { got = v },
	}
	if err := in.HandleEvent(context.Background(), "name", map[string]any{"value": "Serhan"}); err != nil {
		t.Fatal(err)
	}
	if in.Value != "Serhan" || got != "Serhan" {
		t.Fatalf("value=%q callback=%q", in.Value, got)
	}
}

func TestDateTimeInput_HandleEvent(t *testing.T) {
	var got string
	in := &DateTimeInput{
		CommonAttrs: CommonAttrs{Name: "dob"},
		Type:        "date",
		OnChange:    func(v string) { got = v },
	}
	html, _ := in.Render()
	if !strings.Contains(html, `type="date"`) {
		t.Fatalf("bad html: %s", html)
	}
	_ = in.HandleEvent(context.Background(), "dob", map[string]any{"value": "2020-01-02"})
	if in.Value != "2020-01-02" || got != "2020-01-02" {
		t.Fatalf("value=%q callback=%q", in.Value, got)
	}
}

func TestChoiceInput_HandleEvent(t *testing.T) {
	var checked bool
	var value string
	in := &ChoiceInput{
		CommonAttrs: CommonAttrs{Name: "subscribe"},
		Type:        "checkbox",
		Value:       "yes",
		OnChange: func(c bool, v string) {
			checked = c
			value = v
		},
	}
	html, _ := in.Render()
	if !strings.Contains(html, `type="checkbox"`) {
		t.Fatalf("bad html: %s", html)
	}
	_ = in.HandleEvent(context.Background(), "subscribe", map[string]any{"checked": true, "value": "yes"})
	if !in.Checked || !checked || value != "yes" {
		t.Fatalf("checked=%v callback=(%v,%q)", in.Checked, checked, value)
	}
}

func TestSelect_OptgroupRender(t *testing.T) {
	sel := &Select{
		CommonAttrs: CommonAttrs{Name: "country", Required: true},
		Value:       "tr",
		Options: []Option{
			{Value: "", Label: "Select..."},
		},
		Groups: []Optgroup{
			{
				Label: "Europe",
				Options: []Option{
					{Value: "tr", Label: "Türkiye"},
					{Value: "de", Label: "Germany"},
				},
			},
		},
	}
	html, err := sel.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<select`,
		`name="country"`,
		`required`,
		`<optgroup label="Europe">`,
		`value="tr"`,
		`selected`,
		`Türkiye`,
		`</optgroup>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}

	var got string
	sel.OnChange = func(v string) { got = v }
	_ = sel.HandleEvent(context.Background(), "country", map[string]any{"value": "de"})
	if sel.Value != "de" || got != "de" {
		t.Fatalf("value=%q callback=%q", sel.Value, got)
	}
}

func TestButton_Types(t *testing.T) {
	b := &Button{Type: "submit", Text: "Send", EventName: "save"}
	html, _ := b.Render()
	if !strings.Contains(html, `type="submit"`) || !strings.Contains(html, `g-click="save"`) {
		t.Fatalf("bad button: %s", html)
	}

	img := &Button{Type: "image", Src: "/x.png", Alt: "Go"}
	html, _ = img.Render()
	if !strings.HasPrefix(html, "<input") || !strings.Contains(html, `type="image"`) {
		t.Fatalf("bad image button: %s", html)
	}
}

func TestForm_Fieldset_Label_Render(t *testing.T) {
	label := &Label{For: "n", Text: "Name"}
	lh, _ := label.Render()
	if !strings.Contains(lh, `for="n"`) || !strings.Contains(lh, "Name") {
		t.Fatalf("label: %s", lh)
	}

	fs := &Fieldset{InnerHTML: lh}
	fh, _ := fs.Render()
	if !strings.Contains(fh, "<fieldset") || !strings.Contains(fh, lh) {
		t.Fatalf("fieldset: %s", fh)
	}

	form := &Form{Method: "post", OnSubmit: "save", InnerHTML: fh}
	outh, _ := form.Render()
	if !strings.Contains(outh, `method="post"`) || !strings.Contains(outh, `g-submit="save"`) {
		t.Fatalf("form: %s", outh)
	}
}

func TestNumericFileColorHiddenMeterProgressOutputDatalist(t *testing.T) {
	num := &NumericInput{CommonAttrs: CommonAttrs{Name: "age"}, Type: "number", Min: "0", Max: "120", Step: "1"}
	h, _ := num.Render()
	if !strings.Contains(h, `type="number"`) || !strings.Contains(h, `min="0"`) {
		t.Fatalf("numeric: %s", h)
	}

	file := &FileInput{CommonAttrs: CommonAttrs{Name: "cv"}, Accept: ".pdf", Multiple: true}
	h, _ = file.Render()
	if !strings.Contains(h, `type="file"`) || !strings.Contains(h, `accept=".pdf"`) || !strings.Contains(h, "multiple") {
		t.Fatalf("file: %s", h)
	}

	color := &ColorInput{CommonAttrs: CommonAttrs{Name: "c"}, Value: "#ff0000"}
	h, _ = color.Render()
	if !strings.Contains(h, `type="color"`) || !strings.Contains(h, `#ff0000`) {
		t.Fatalf("color: %s", h)
	}

	hid := &HiddenInput{CommonAttrs: CommonAttrs{Name: "token"}, Value: "abc"}
	h, _ = hid.Render()
	if !strings.Contains(h, `type="hidden"`) || !strings.Contains(h, `value="abc"`) {
		t.Fatalf("hidden: %s", h)
	}

	meter := &Meter{Value: 0.5, Min: 0, Max: 1}
	h, _ = meter.Render()
	if !strings.Contains(h, "<meter") || !strings.Contains(h, `value="0.5"`) {
		t.Fatalf("meter: %s", h)
	}

	prog := &Progress{Value: 40, Max: 100}
	h, _ = prog.Render()
	if !strings.Contains(h, "<progress") || !strings.Contains(h, `max="100"`) {
		t.Fatalf("progress: %s", h)
	}

	out := &Output{CommonAttrs: CommonAttrs{Name: "sum"}, Text: "42"}
	h, _ = out.Render()
	if !strings.Contains(h, "<output") || !strings.Contains(h, "42") {
		t.Fatalf("output: %s", h)
	}

	dl := &Datalist{CommonAttrs: CommonAttrs{ID: "cities"}, Options: []DatalistOption{{Value: "Istanbul"}}}
	h, _ = dl.Render()
	if !strings.Contains(h, `id="cities"`) || !strings.Contains(h, `value="Istanbul"`) {
		t.Fatalf("datalist: %s", h)
	}
}

func TestTextarea_Render(t *testing.T) {
	ta := &Textarea{CommonAttrs: CommonAttrs{Name: "msg", Required: true}, Value: "hi", Rows: 4}
	h, _ := ta.Render()
	if !strings.Contains(h, "<textarea") || !strings.Contains(h, "required") || !strings.Contains(h, ">hi<") {
		t.Fatalf("textarea: %s", h)
	}
}
