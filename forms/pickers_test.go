package forms

import (
	"context"
	"strings"
	"testing"
)

func TestPickers_HaveItems(t *testing.T) {
	if len(CountryItems()) < 5 || len(LanguageItems()) < 3 {
		t.Fatal("datasets too small")
	}
	c := NewCountryPicker("country", "country")
	if len(c.Items) == 0 || c.EventName != "country" {
		t.Fatalf("%#v", c)
	}
	_ = NewLanguagePicker("lang", "lang")
	_ = NewTimezonePicker("tz", "tz")
	_ = NewCurrencyPicker("cur", "cur")
}

func TestPhoneInput_Compose(t *testing.T) {
	p := NewPhoneInput("phone")
	ctx := context.Background()
	_ = p.HandleEvent(ctx, "phone_num", map[string]any{"value": "5320000000"})
	if !strings.Contains(p.RawValue(), "5320000000") {
		t.Fatalf("raw=%q", p.RawValue())
	}
	html, err := p.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "goui-phone") {
		t.Fatalf("render: %s", html)
	}
}

func TestVisualPickers(t *testing.T) {
	if len(EmojiItems()) < 3 || len(IconItems()) < 3 || len(FontItems()) < 3 {
		t.Fatal("datasets")
	}
	e := NewEmojiPicker("emo", "emo")
	if len(e.Items) == 0 || e.EventName != "emo" {
		t.Fatalf("%#v", e)
	}
	_ = NewIconPicker("ico", "ico")
	_ = NewFontPicker("font", "font")
}
