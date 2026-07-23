package forms

import (
	"context"
	"strings"
	"testing"
)

func TestOTPInput_DigitsAndCommit(t *testing.T) {
	o := &OTPInput{
		CommonAttrs: CommonAttrs{Name: "otp"},
		Length:      4,
		EventName:   "otp",
	}
	ctx := context.Background()
	_ = o.HandleEvent(ctx, "otp.digit", map[string]any{"index": "0", "value": "1"})
	_ = o.HandleEvent(ctx, "otp.digit", map[string]any{"index": "1", "value": "2"})
	_ = o.HandleEvent(ctx, "otp.digit", map[string]any{"index": "2", "value": "3"})
	_ = o.HandleEvent(ctx, "otp.digit", map[string]any{"index": "3", "value": "4"})
	if o.Value != "1234" {
		t.Fatalf("value=%q", o.Value)
	}
	html, _ := o.Render()
	if strings.Count(html, "goui-otp-cell") != 4 {
		t.Fatalf("cells: %s", html)
	}
	_ = o.HandleEvent(ctx, "otp.commit", map[string]any{"value": "9876"})
	if o.Value != "9876" {
		t.Fatalf("commit=%q", o.Value)
	}
}

func TestOTPInput_IncompleteValidate(t *testing.T) {
	o := &OTPInput{Length: 6, Value: "12"}
	if o.Validate() {
		t.Fatal("expected incomplete")
	}
}
