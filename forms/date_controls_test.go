package forms

import (
	"context"
	"strings"
	"testing"
)

func TestDateRangePicker_StartEnd(t *testing.T) {
	d := &DateRangePicker{
		CommonAttrs: CommonAttrs{Name: "leave"},
		EventName:   "leave",
	}
	ctx := context.Background()
	_ = d.HandleEvent(ctx, "leave.start", map[string]any{"value": "2026-07-10"})
	_ = d.HandleEvent(ctx, "leave.end", map[string]any{"value": "2026-07-15"})
	if d.RawValue() != "2026-07-10/2026-07-15" {
		t.Fatalf("raw=%q", d.RawValue())
	}
	html, _ := d.Render()
	if !strings.Contains(html, `type="date"`) || !strings.Contains(html, "goui-date-range") {
		t.Fatalf("render: %s", html)
	}
	d.End = "2026-07-01"
	if d.Validate() {
		t.Fatal("expected invalid range")
	}
}

func TestTimeRangePicker_StartEnd(t *testing.T) {
	tr := &TimeRangePicker{
		CommonAttrs: CommonAttrs{Name: "shift"},
		EventName:   "shift",
	}
	ctx := context.Background()
	_ = tr.HandleEvent(ctx, "shift.start", map[string]any{"value": "09:00"})
	_ = tr.HandleEvent(ctx, "shift.end", map[string]any{"value": "17:30"})
	if tr.RawValue() != "09:00/17:30" {
		t.Fatalf("raw=%q", tr.RawValue())
	}
	html, _ := tr.Render()
	if !strings.Contains(html, `type="time"`) {
		t.Fatalf("render: %s", html)
	}
}

func TestCalendarDatePicker_Select(t *testing.T) {
	c := &CalendarDatePicker{
		CommonAttrs: CommonAttrs{Name: "day"},
		EventName:   "day",
	}
	ctx := context.Background()
	_ = c.HandleEvent(ctx, "day.toggle", nil)
	if !c.Open {
		t.Fatal("expected open")
	}
	_ = c.HandleEvent(ctx, "day.select", map[string]any{"value": "2026-07-16"})
	if c.Value != "2026-07-16" || c.Open {
		t.Fatalf("value=%q open=%v", c.Value, c.Open)
	}
	html, _ := c.Render()
	if !strings.Contains(html, "2026-07-16") {
		t.Fatalf("trigger: %s", html)
	}
	_ = c.HandleEvent(ctx, "day.toggle", nil)
	html, _ = c.Render()
	if !strings.Contains(html, "data-goui-calendar-mount") || !strings.Contains(html, "data-select-event") {
		t.Fatalf("panel: %s", html)
	}
}
