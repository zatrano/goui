package forms

import (
	"context"

	"github.com/zatrano/goui/core"
)

// FileInput covers type=file.
type FileInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation
	Accept    string
	Capture   string
	Multiple  bool
	EventName string
	OnChange  func(fileNames string)
	// Value holds last selected file name(s) for display/state (actual upload is later phase).
	Value string
}

func (f *FileInput) Name() string         { return f.CommonAttrs.Name }
func (f *FileInput) RawValue() string     { return f.Value }
func (f *FileInput) SetRawValue(v string) { f.Value = v }

func (f *FileInput) Mount(_ context.Context) error   { return nil }
func (f *FileInput) Unmount(_ context.Context) error { return nil }

func (f *FileInput) Validate() bool {
	return f.FieldValidation.run(f.Value, f.T)
}

func (f *FileInput) eventName() string {
	if f.EventName != "" {
		return f.EventName
	}
	return f.CommonAttrs.Name
}

func (f *FileInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = f.CommonAttrs.apply(attrs)
	attrs = f.FieldValidation.applyErrorState(attrs, classInput)
	attrs = attrs.Set("type", "file")
	attrs = attrs.Set("accept", f.Accept)
	attrs = attrs.Set("capture", f.Capture)
	attrs = attrs.SetBool("multiple", f.Multiple)
	if ev := f.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
	}
	return "<input" + attrs.String() + ">" + f.FieldValidation.errorsHTML(), nil
}

func (f *FileInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if event != "change" && event != f.eventName() {
		return nil
	}
	val := payloadString(payload, "value")
	f.Value = val
	f.MarkDirty()
	if f.OnChange != nil {
		f.OnChange(val)
	}
	return nil
}
