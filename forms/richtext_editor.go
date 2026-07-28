package forms

import (
	"context"
	"strconv"

	"github.com/zatrano/goui/core"
)

// RichTextEditor stores HTML content on the server; Quill UI is client-side (CDN).
// Render output is intentionally stable (no live Value in the DOM) so patches do not
// remount Quill; use data-goui-ignore + ErrSkipRender on sync events.
type RichTextEditor struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value      string // HTML
	EventName  string
	DebounceMS int
	OnChange   func(value string)
}

func (r *RichTextEditor) Name() string         { return r.CommonAttrs.Name }
func (r *RichTextEditor) RawValue() string     { return r.Value }
func (r *RichTextEditor) SetRawValue(v string) { r.Value = v }

func (r *RichTextEditor) Mount(_ context.Context) error   { return nil }
func (r *RichTextEditor) Unmount(_ context.Context) error { return nil }

func (r *RichTextEditor) Validate() bool {
	return r.FieldValidation.Run(r.Value, r.T)
}

func (r *RichTextEditor) eventName() string {
	if r.EventName != "" {
		return r.EventName
	}
	return r.CommonAttrs.Name
}

func (r *RichTextEditor) debounce() int {
	if r.DebounceMS > 0 {
		return r.DebounceMS
	}
	return 350
}

func (r *RichTextEditor) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, r.eventName())
	switch action {
	case "sync", "change", "input", r.eventName():
		r.Value = payloadString(payload, "value")
		if r.OnChange != nil {
			r.OnChange(r.Value)
		}
		// Do not MarkDirty — client owns the editor DOM.
	}
	return nil
}

func (r *RichTextEditor) Render() (string, error) {
	attrs := Attrs{}
	attrs = r.CommonAttrs.Apply(attrs)
	attrs = r.FieldValidation.ApplyErrorState(attrs, "goui-richtext")
	attrs = attrs.Set("data-goui-richtext", "1")
	attrs = attrs.Set("data-goui-ignore", "1")
	attrs = attrs.Set("data-initial", r.Value)

	sync := Attrs{}
	sync = sync.Set("class", "goui-editor-sync")
	sync = sync.Set("aria-hidden", "true")
	if ev := r.eventName(); ev != "" {
		sync = sync.Set("g-input", ev+".sync")
		sync = sync.Set("g-debounce", strconv.Itoa(r.debounce()))
	}

	// Sync textarea stays empty in HTML so re-renders stay stable; Quill fills it.
	return `<div` + attrs.String() + `>` +
		`<div class="goui-richtext-mount" data-goui-quill-mount></div>` +
		`<textarea` + sync.String() + `></textarea>` +
		`</div>` + r.FieldValidation.ErrorsHTML(), nil
}
