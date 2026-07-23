package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// SignaturePad captures a drawn signature; client uploads PNG then notifies via WS.
type SignaturePad struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	File      UploadedRef
	UploadURL string
	EventName string
	OnChange  func(ref UploadedRef)
}

func (s *SignaturePad) Name() string         { return s.CommonAttrs.Name }
func (s *SignaturePad) RawValue() string     { return s.File.ID }
func (s *SignaturePad) SetRawValue(v string) { s.File.ID = v }

func (s *SignaturePad) Mount(_ context.Context) error   { return nil }
func (s *SignaturePad) Unmount(_ context.Context) error { return nil }

func (s *SignaturePad) Validate() bool {
	return s.FieldValidation.Run(s.RawValue(), s.T)
}

func (s *SignaturePad) eventName() string {
	if s.EventName != "" {
		return s.EventName
	}
	return s.CommonAttrs.Name
}

func (s *SignaturePad) ev(action string) string {
	base := s.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (s *SignaturePad) uploadURL() string {
	if s.UploadURL != "" {
		return s.UploadURL
	}
	return "/goui/upload"
}

func (s *SignaturePad) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, s.eventName())
	switch action {
	case "uploaded":
		s.File = refFromPayload(payload)
		s.MarkDirty()
		if s.OnChange != nil {
			s.OnChange(s.File)
		}
	case "clear":
		s.File = UploadedRef{}
		s.MarkDirty()
		if s.OnChange != nil {
			s.OnChange(s.File)
		}
	}
	return nil
}

func (s *SignaturePad) Render() (string, error) {
	attrs := Attrs{}
	attrs = s.CommonAttrs.Apply(attrs)
	attrs = s.FieldValidation.ApplyErrorState(attrs, "goui-signature")
	attrs = attrs.Set("data-goui-signature", "1")
	attrs = attrs.Set("data-upload-url", s.uploadURL())

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(`<canvas class="goui-signature-canvas border border-goui-border rounded-goui" width="360" height="160"></canvas>`)
	b.WriteString(`<div class="goui-signature-actions">`)
	b.WriteString(`<button type="button" class="goui-signature-save">Kaydet</button> `)
	b.WriteString(`<button type="button" class="goui-signature-clear-local">Temizle</button>`)
	if s.File.ID != "" {
		b.WriteString(` <button type="button" g-click="` + html.EscapeString(s.ev("clear")) + `">Kaydı sil</button>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<button type="button" class="goui-upload-carrier" hidden g-click="` + html.EscapeString(s.ev("uploaded")) + `"></button>`)
	if s.File.URL != "" {
		b.WriteString(`<p class="goui-helper-text text-sm">Kayıt: <a href="` + html.EscapeString(s.File.URL) + `" target="_blank" rel="noopener">` + html.EscapeString(s.File.Name) + `</a></p>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(s.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
