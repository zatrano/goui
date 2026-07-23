package diff

import (
	"reflect"
	"testing"
)

func mustParse(t *testing.T, html string) *Node {
	t.Helper()
	n, err := ParseHTML(html)
	if err != nil {
		t.Fatalf("ParseHTML(%q): %v", html, err)
	}
	return n
}

func TestDiff_TagChange(t *testing.T) {
	patches := Diff(mustParse(t, `<div>Hello</div>`), mustParse(t, `<span>Hello</span>`))
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != OpReplace {
		t.Fatalf("op = %q, want %q", patches[0].Op, OpReplace)
	}
	if !reflect.DeepEqual(patches[0].Path, []int{0}) {
		t.Fatalf("path = %v, want [0]", patches[0].Path)
	}
}

func TestDiff_TextChange(t *testing.T) {
	patches := Diff(mustParse(t, `<div><span>Hello</span></div>`), mustParse(t, `<div><span>Hi</span></div>`))
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != OpUpdateText {
		t.Fatalf("op = %q, want %q", patches[0].Op, OpUpdateText)
	}
	if !reflect.DeepEqual(patches[0].Path, []int{0, 0, 0}) {
		t.Fatalf("path = %v, want [0 0 0]", patches[0].Path)
	}
	if patches[0].Text != "Hi" {
		t.Fatalf("text = %q, want Hi", patches[0].Text)
	}
}

func TestDiff_AttrAddedChangedRemoved(t *testing.T) {
	oldNode := mustParse(t, `<div class="old" data-role="box"></div>`)
	newNode := mustParse(t, `<div class="new" id="main"></div>`)

	patches := Diff(oldNode, newNode)
	if len(patches) != 3 {
		t.Fatalf("patch count = %d, want 3", len(patches))
	}

	want := map[PatchOp]map[string]string{
		OpSetAttr: {
			"class": "new",
			"id":    "main",
		},
		OpRemoveAttr: {
			"data-role": "",
		},
	}

	for _, patch := range patches {
		if !reflect.DeepEqual(patch.Path, []int{0}) {
			t.Fatalf("attr patch path = %v, want [0]", patch.Path)
		}
		attrs, ok := want[patch.Op]
		if !ok {
			t.Fatalf("unexpected op %q", patch.Op)
		}
		val, ok := attrs[patch.Attr]
		if !ok {
			t.Fatalf("unexpected attr %q for op %q", patch.Attr, patch.Op)
		}
		if patch.Op == OpSetAttr && patch.Value != val {
			t.Fatalf("value for %q = %q, want %q", patch.Attr, patch.Value, val)
		}
	}
}

func TestDiff_NoChange(t *testing.T) {
	patches := Diff(mustParse(t, `<div><span class="x">Hello</span></div>`), mustParse(t, `<div><span class="x">Hello</span></div>`))
	if len(patches) != 0 {
		t.Fatalf("patch count = %d, want 0", len(patches))
	}
}

func TestDiff_ChildInsertedAtEnd(t *testing.T) {
	patches := Diff(mustParse(t, `<ul><li>A</li></ul>`), mustParse(t, `<ul><li>A</li><li>B</li></ul>`))
	assertSingleInsert(t, patches, []int{0, 1}, "<li>B</li>")
}

func TestDiff_ChildInsertedAtStart(t *testing.T) {
	patches := Diff(mustParse(t, `<ul><li>B</li></ul>`), mustParse(t, `<ul><li>A</li><li>B</li></ul>`))
	if len(patches) != 2 {
		t.Fatalf("patch count = %d, want 2", len(patches))
	}
	if patches[0].Op != OpUpdateText || !reflect.DeepEqual(patches[0].Path, []int{0, 0, 0}) || patches[0].Text != "A" {
		t.Fatalf("unexpected first patch: %+v", patches[0])
	}
	if patches[1].Op != OpInsert || !reflect.DeepEqual(patches[1].Path, []int{0, 1}) || patches[1].HTML != "<li>B</li>" {
		t.Fatalf("unexpected second patch: %+v", patches[1])
	}
}

func TestDiff_ChildInsertedInMiddle(t *testing.T) {
	patches := Diff(mustParse(t, `<ul><li>A</li><li>C</li></ul>`), mustParse(t, `<ul><li>A</li><li>B</li><li>C</li></ul>`))
	if len(patches) != 2 {
		t.Fatalf("patch count = %d, want 2", len(patches))
	}
	if patches[0].Op != OpUpdateText || !reflect.DeepEqual(patches[0].Path, []int{0, 1, 0}) || patches[0].Text != "B" {
		t.Fatalf("unexpected first patch: %+v", patches[0])
	}
	if patches[1].Op != OpInsert || !reflect.DeepEqual(patches[1].Path, []int{0, 2}) || patches[1].HTML != "<li>C</li>" {
		t.Fatalf("unexpected second patch: %+v", patches[1])
	}
}

func TestDiff_ChildRemoved(t *testing.T) {
	patches := Diff(mustParse(t, `<ul><li>A</li><li>B</li></ul>`), mustParse(t, `<ul><li>A</li></ul>`))
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != OpRemove {
		t.Fatalf("op = %q, want %q", patches[0].Op, OpRemove)
	}
	if !reflect.DeepEqual(patches[0].Path, []int{0, 1}) {
		t.Fatalf("path = %v, want [0 1]", patches[0].Path)
	}
}

func assertSingleInsert(t *testing.T, patches []Patch, wantPath []int, wantHTML string) {
	t.Helper()
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != OpInsert {
		t.Fatalf("op = %q, want %q", patches[0].Op, OpInsert)
	}
	if !reflect.DeepEqual(patches[0].Path, wantPath) {
		t.Fatalf("path = %v, want %v", patches[0].Path, wantPath)
	}
	if patches[0].HTML != wantHTML {
		t.Fatalf("html = %q, want %q", patches[0].HTML, wantHTML)
	}
}
