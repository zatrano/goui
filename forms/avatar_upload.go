package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// AvatarUpload: pick image → optional client crop (1:1) → upload → store ref.
type AvatarUpload struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Avatar    UploadedRef
	UploadURL string
	EventName string
	OnChange  func(ref UploadedRef)
}

func (a *AvatarUpload) Name() string         { return a.CommonAttrs.Name }
func (a *AvatarUpload) RawValue() string     { return a.Avatar.ID }
func (a *AvatarUpload) SetRawValue(v string) { a.Avatar.ID = v }

func (a *AvatarUpload) Mount(_ context.Context) error   { return nil }
func (a *AvatarUpload) Unmount(_ context.Context) error { return nil }

func (a *AvatarUpload) Validate() bool {
	return a.FieldValidation.Run(a.RawValue(), a.T)
}

func (a *AvatarUpload) eventName() string {
	if a.EventName != "" {
		return a.EventName
	}
	return a.CommonAttrs.Name
}

func (a *AvatarUpload) ev(action string) string {
	base := a.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (a *AvatarUpload) uploadURL() string {
	if a.UploadURL != "" {
		return a.UploadURL
	}
	return "/goui/upload"
}

func (a *AvatarUpload) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, a.eventName())
	switch action {
	case "uploaded":
		a.Avatar = refFromPayload(payload)
		a.MarkDirty()
		if a.OnChange != nil {
			a.OnChange(a.Avatar)
		}
	case "clear":
		a.Avatar = UploadedRef{}
		a.MarkDirty()
		if a.OnChange != nil {
			a.OnChange(a.Avatar)
		}
	}
	return nil
}

func (a *AvatarUpload) Render() (string, error) {
	attrs := Attrs{}
	attrs = a.CommonAttrs.Apply(attrs)
	attrs = a.FieldValidation.ApplyErrorState(attrs, "goui-avatar")
	attrs = attrs.Set("data-goui-avatar", "1")
	attrs = attrs.Set("data-upload-url", a.uploadURL())
	attrs = attrs.Set("data-upload-event", a.ev("uploaded"))

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(`<div class="goui-avatar-preview">`)
	if a.Avatar.URL != "" {
		b.WriteString(`<img src="` + html.EscapeString(a.Avatar.URL) + `" alt="avatar">`)
	} else {
		b.WriteString(`<span class="goui-avatar-empty">Avatar</span>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<input class="goui-avatar-input" type="file" accept="image/*">`)
	if a.Avatar.ID != "" {
		b.WriteString(` <button type="button" g-click="` + html.EscapeString(a.ev("clear")) + `">Temizle</button>`)
	}
	b.WriteString(`<button type="button" class="goui-upload-carrier" hidden g-click="` + html.EscapeString(a.ev("uploaded")) + `"></button>`)
	b.WriteString(`<div class="goui-crop-overlay" hidden>`)
	b.WriteString(`<canvas class="goui-crop-canvas" width="280" height="280"></canvas>`)
	b.WriteString(`<div class="goui-crop-actions">`)
	b.WriteString(`<button type="button" class="goui-crop-apply">Kırp &amp; Yükle</button>`)
	b.WriteString(`<button type="button" class="goui-crop-cancel">İptal</button>`)
	b.WriteString(`</div></div>`)
	b.WriteString(`</div>`)
	b.WriteString(a.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
