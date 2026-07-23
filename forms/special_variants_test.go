package forms

import (
	"context"
	"strings"
	"testing"
)

func TestTagInput_AddRemove(t *testing.T) {
	tag := &TagInput{
		CommonAttrs: CommonAttrs{Name: "skills"},
		EventName:   "skills",
	}
	ctx := context.Background()
	_ = tag.HandleEvent(ctx, "skills.add", map[string]any{"value": "go, goui"})
	if len(tag.Values) != 2 {
		t.Fatalf("values=%#v", tag.Values)
	}
	_ = tag.HandleEvent(ctx, "skills.remove", map[string]any{"value": "go"})
	if tag.RawValue() != "goui" {
		t.Fatalf("raw=%q", tag.RawValue())
	}
	html, _ := tag.Render()
	if !strings.Contains(html, "goui-chip") || !strings.Contains(html, "goui") {
		t.Fatalf("render: %s", html)
	}
}

func TestTreeSelect_ExpandAndSelect(t *testing.T) {
	tr := &TreeSelect{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: "dept"},
		},
		EventName: "dept",
		Nodes: []TreeNode{
			{
				Value: "eng", Label: "Engineering",
				Children: []TreeNode{
					{Value: "be", Label: "Backend"},
					{Value: "fe", Label: "Frontend"},
				},
			},
		},
	}
	ctx := context.Background()
	_ = tr.HandleEvent(ctx, "dept.toggle", nil)
	_ = tr.HandleEvent(ctx, "dept.expand", map[string]any{"value": "eng"})
	if !tr.Expanded["eng"] {
		t.Fatal("expected expanded")
	}
	_ = tr.HandleEvent(ctx, "dept.select", map[string]any{"value": "be"})
	if tr.Value != "be" || tr.Open {
		t.Fatalf("value=%q open=%v", tr.Value, tr.Open)
	}
	html, _ := tr.Render()
	if !strings.Contains(html, "Backend") && !strings.Contains(html, "Seçin") {
		// closed after select — trigger shows Backend label
		if !strings.Contains(html, "Backend") {
			t.Fatalf("render: %s", html)
		}
	}
}

func TestCascader_LoadChildren(t *testing.T) {
	c := &Cascader{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: "loc"},
			Items: []SelectItem{
				{Value: "tr", Label: "Türkiye"},
				{Value: "de", Label: "Almanya"},
			},
		},
		EventName: "loc",
		LoadChildren: func(level int, parent string) []SelectItem {
			if level == 0 && parent == "tr" {
				return []SelectItem{{Value: "ist", Label: "İstanbul"}, {Value: "ank", Label: "Ankara"}}
			}
			return nil
		},
	}
	_ = c.Mount(context.Background())
	ctx := context.Background()
	_ = c.HandleEvent(ctx, "loc.toggle", nil)
	_ = c.HandleEvent(ctx, "loc.pick", map[string]any{"value": "tr", "level": "0"})
	if len(c.Levels) != 2 {
		t.Fatalf("levels=%d", len(c.Levels))
	}
	_ = c.HandleEvent(ctx, "loc.pick", map[string]any{"value": "ank", "level": "1"})
	if c.RawValue() != "tr/ank" {
		t.Fatalf("raw=%q", c.RawValue())
	}
	if c.Open {
		t.Fatal("should close on leaf")
	}
}

func TestDualListbox_MoveAndFilter(t *testing.T) {
	d := &DualListbox{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: "perms"},
			Items:       sampleCities(),
		},
		EventName: "perms",
	}
	ctx := context.Background()
	_ = d.HandleEvent(ctx, "perms.add", map[string]any{"value": "ank"})
	if d.RawValue() != "ank" {
		t.Fatalf("raw=%q", d.RawValue())
	}
	_ = d.HandleEvent(ctx, "perms.query_left", map[string]any{"value": "izm"})
	if len(d.Filtered) != 1 || d.Filtered[0].Value != "izm" {
		t.Fatalf("left filter: %#v", d.Filtered)
	}
	_ = d.HandleEvent(ctx, "perms.add", map[string]any{"value": "izm"})
	_ = d.HandleEvent(ctx, "perms.query_right", map[string]any{"value": "ank"})
	if len(d.SelectedFilter) != 1 || d.SelectedFilter[0].Value != "ank" {
		t.Fatalf("right filter: %#v", d.SelectedFilter)
	}
	_ = d.HandleEvent(ctx, "perms.remove", map[string]any{"value": "ank"})
	if d.RawValue() != "izm" {
		t.Fatalf("after remove: %q", d.RawValue())
	}
	html, _ := d.Render()
	if !strings.Contains(html, "goui-dual-listbox") {
		t.Fatalf("render: %s", html)
	}
}
