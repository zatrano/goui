package forms

import (
	"context"
	"html"
	"strings"
)

// TreeNode is a hierarchical selectable item.
type TreeNode struct {
	Value    string
	Label    string
	Disabled bool
	Children []TreeNode
}

// TreeSelect selects a single node from a tree (server expands/collapses).
type TreeSelect struct {
	BaseSelectField
	Nodes     []TreeNode
	Expanded  map[string]bool
	EventName string
}

func (t *TreeSelect) Name() string         { return t.CommonAttrs.Name }
func (t *TreeSelect) RawValue() string     { return t.Value }
func (t *TreeSelect) SetRawValue(v string) { t.Value = v }

func (t *TreeSelect) Mount(_ context.Context) error {
	if t.Expanded == nil {
		t.Expanded = map[string]bool{}
	}
	return nil
}
func (t *TreeSelect) Unmount(_ context.Context) error { return nil }

func (t *TreeSelect) Validate() bool {
	return t.FieldValidation.Run(t.Value, t.T)
}

func (t *TreeSelect) eventName() string {
	if t.EventName != "" {
		return t.EventName
	}
	return t.CommonAttrs.Name
}

func (t *TreeSelect) ev(action string) string {
	base := t.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (t *TreeSelect) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if t.Expanded == nil {
		t.Expanded = map[string]bool{}
	}
	action := eventAction(event, t.eventName(), payload)
	switch action {
	case "toggle":
		t.Open = !t.Open
		t.MarkDirty()
	case "close":
		t.Open = false
		t.MarkDirty()
	case "expand":
		id := payloadString(payload, "value")
		t.Expanded[id] = !t.Expanded[id]
		t.MarkDirty()
	case "select":
		val := payloadString(payload, "value")
		t.Value = val
		t.Open = false
		t.MarkDirty()
		if t.OnChange != nil {
			t.OnChange(val)
		}
	}
	return nil
}

func (t *TreeSelect) selectedLabel() string {
	var walk func([]TreeNode) string
	walk = func(nodes []TreeNode) string {
		for _, n := range nodes {
			if n.Value == t.Value {
				if n.Label != "" {
					return n.Label
				}
				return n.Value
			}
			if s := walk(n.Children); s != "" {
				return s
			}
		}
		return ""
	}
	return walk(t.Nodes)
}

func (t *TreeSelect) Render() (string, error) {
	if t.Expanded == nil {
		t.Expanded = map[string]bool{}
	}
	attrs := Attrs{}
	attrs = t.CommonAttrs.Apply(attrs)
	attrs = t.FieldValidation.ApplyErrorState(attrs, "goui-searchable goui-tree")

	label := t.selectedLabel()
	if label == "" {
		label = t.Placeholder
		if label == "" {
			label = "Seçin..."
		}
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="tree">`)
	b.WriteString(`<button type="button" class="goui-searchable-trigger border border-goui-border rounded-goui px-goui-field py-goui-field w-full" g-click="` + html.EscapeString(t.ev("toggle")) + `">`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</button>`)
	if t.Open {
		b.WriteString(`<div class="goui-searchable-backdrop" g-click="` + html.EscapeString(t.ev("close")) + `"></div>`)
		b.WriteString(`<div class="goui-searchable-panel border border-goui-border rounded-goui"><ul class="goui-tree-list">`)
		b.WriteString(renderTreeNodes(t.Nodes, t, 0))
		b.WriteString(`</ul></div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(t.FieldValidation.ErrorsHTML())
	return b.String(), nil
}

func renderTreeNodes(nodes []TreeNode, t *TreeSelect, depth int) string {
	var b strings.Builder
	for _, n := range nodes {
		pad := strings.Repeat("— ", depth)
		hasKids := len(n.Children) > 0
		expanded := t.Expanded[n.Value]
		b.WriteString(`<li class="goui-tree-node">`)
		if hasKids {
			b.WriteString(`<button type="button" class="goui-tree-expand" g-click="` + html.EscapeString(t.ev("expand")) + `" data-goui-value="` + html.EscapeString(n.Value) + `">`)
			if expanded {
				b.WriteString(`▼`)
			} else {
				b.WriteString(`▶`)
			}
			b.WriteString(`</button>`)
		} else {
			b.WriteString(`<span class="goui-tree-spacer"></span>`)
		}
		b.WriteString(`<button type="button" class="goui-tree-label`)
		if n.Value == t.Value {
			b.WriteString(` is-selected`)
		}
		b.WriteString(`" g-click="` + html.EscapeString(t.ev("select")) + `" data-goui-value="` + html.EscapeString(n.Value) + `">`)
		b.WriteString(html.EscapeString(pad + displayLabel(SelectItem{Value: n.Value, Label: n.Label})))
		b.WriteString(`</button>`)
		if hasKids && expanded {
			b.WriteString(`<ul class="goui-tree-list">`)
			b.WriteString(renderTreeNodes(n.Children, t, depth+1))
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</li>`)
	}
	return b.String()
}
