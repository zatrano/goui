package forms

import (
	"context"
	"html"
	"strconv"
	"strings"

	"github.com/zatrano/goui/core"
)

// UploadedRef is server-side metadata after HTTP upload completes.
type UploadedRef struct {
	ID          string
	Name        string
	URL         string
	ContentType string
	Size        int64
}

// DragDropUpload collects uploaded file refs (binary via /goui/upload, refs via WS).
type DragDropUpload struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Files      []UploadedRef
	Accept     string
	Multiple   bool
	ShowThumbs bool
	UploadURL  string
	EventName  string
	OnChange   func(files []UploadedRef)
}

func (d *DragDropUpload) Name() string { return d.CommonAttrs.Name }

func (d *DragDropUpload) RawValue() string {
	ids := make([]string, 0, len(d.Files))
	for _, f := range d.Files {
		ids = append(ids, f.ID)
	}
	return strings.Join(ids, ",")
}

func (d *DragDropUpload) SetRawValue(v string) { _ = v }

func (d *DragDropUpload) Mount(_ context.Context) error   { return nil }
func (d *DragDropUpload) Unmount(_ context.Context) error { return nil }

func (d *DragDropUpload) Validate() bool {
	return d.FieldValidation.Run(d.RawValue(), d.T)
}

func (d *DragDropUpload) eventName() string {
	if d.EventName != "" {
		return d.EventName
	}
	return d.CommonAttrs.Name
}

func (d *DragDropUpload) ev(action string) string {
	base := d.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (d *DragDropUpload) uploadURL() string {
	if d.UploadURL != "" {
		return d.UploadURL
	}
	return "/goui/upload"
}

func (d *DragDropUpload) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, d.eventName())
	switch action {
	case "uploaded":
		ref := refFromPayload(payload)
		if ref.ID == "" {
			return nil
		}
		if !d.Multiple {
			d.Files = []UploadedRef{ref}
		} else {
			d.Files = append(d.Files, ref)
		}
		d.MarkDirty()
		if d.OnChange != nil {
			d.OnChange(d.Files)
		}
	case "remove":
		id := payloadString(payload, "value", "id")
		out := make([]UploadedRef, 0, len(d.Files))
		for _, f := range d.Files {
			if f.ID != id {
				out = append(out, f)
			}
		}
		d.Files = out
		d.MarkDirty()
		if d.OnChange != nil {
			d.OnChange(d.Files)
		}
	}
	return nil
}

func refFromPayload(payload map[string]any) UploadedRef {
	return UploadedRef{
		ID:          payloadString(payload, "id", "value"),
		Name:        payloadString(payload, "name"),
		URL:         payloadString(payload, "url"),
		ContentType: payloadString(payload, "contentType", "type"),
		Size:        int64(payloadInt(payload, "size")),
	}
}

func (d *DragDropUpload) Render() (string, error) {
	attrs := Attrs{}
	attrs = d.CommonAttrs.Apply(attrs)
	attrs = d.FieldValidation.ApplyErrorState(attrs, "goui-upload")
	attrs = attrs.Set("data-goui-upload", "1")
	attrs = attrs.Set("data-upload-url", d.uploadURL())
	attrs = attrs.Set("data-upload-event", d.ev("uploaded"))
	attrs = attrs.Set("data-accept", d.Accept)
	if d.Multiple {
		attrs = attrs.Set("data-multiple", "1")
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(`<div class="goui-upload-drop border border-goui-border rounded-goui">`)
	b.WriteString(`<p class="goui-upload-hint">Dosyayı sürükleyin veya seçin</p>`)
	b.WriteString(`<input class="goui-upload-input" type="file"`)
	if d.Accept != "" {
		b.WriteString(` accept="` + html.EscapeString(d.Accept) + `"`)
	}
	if d.Multiple {
		b.WriteString(` multiple`)
	}
	b.WriteString(`>`)
	b.WriteString(`</div>`)
	b.WriteString(`<button type="button" class="goui-upload-carrier" hidden g-click="` + html.EscapeString(d.ev("uploaded")) + `"></button>`)
	if d.ShowThumbs && len(d.Files) > 0 {
		b.WriteString(`<div class="goui-upload-thumbs">`)
		for _, f := range d.Files {
			if strings.HasPrefix(f.ContentType, "image/") || looksLikeImage(f.Name) {
				b.WriteString(`<img class="goui-upload-thumb" src="` + html.EscapeString(f.URL) + `" alt="` + html.EscapeString(f.Name) + `">`)
			}
		}
		b.WriteString(`</div>`)
	}
	if len(d.Files) > 0 {
		b.WriteString(`<ul class="goui-upload-list">`)
		for _, f := range d.Files {
			b.WriteString(`<li>`)
			b.WriteString(html.EscapeString(f.Name))
			b.WriteString(` <span class="goui-upload-size">(` + strconv.FormatInt(f.Size, 10) + ` B)</span>`)
			b.WriteString(` <button type="button" class="goui-upload-remove" g-click="` + html.EscapeString(d.ev("remove")) + `" data-goui-value="` + html.EscapeString(f.ID) + `">×</button>`)
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(d.FieldValidation.ErrorsHTML())
	return b.String(), nil
}

func looksLikeImage(name string) bool {
	n := strings.ToLower(name)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp"} {
		if strings.HasSuffix(n, ext) {
			return true
		}
	}
	return false
}

// ImageUpload is a DragDropUpload preset for images with thumbnails.
func NewImageUpload(name, event string) DragDropUpload {
	if event == "" {
		event = name
	}
	return DragDropUpload{
		CommonAttrs: CommonAttrs{Name: name, ID: name},
		Accept:      "image/*",
		ShowThumbs:  true,
		EventName:   event,
	}
}
