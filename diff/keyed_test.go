package diff

import "testing"

func TestDiff_KeyedList_Reorder(t *testing.T) {
	oldNode := mustParse(t, `<ul><li data-key="a">A</li><li data-key="b">B</li><li data-key="c">C</li></ul>`)
	newNode := mustParse(t, `<ul><li data-key="b">B</li><li data-key="a">A</li><li data-key="c">C</li></ul>`)

	patches := Diff(oldNode, newNode)
	if len(patches) != 2 {
		t.Fatalf("patch count = %d, want 2 moves", len(patches))
	}
	for _, patch := range patches {
		if patch.Op != OpMove {
			t.Fatalf("unexpected op %q, want only move", patch.Op)
		}
	}
}

func TestDiff_KeyedList_InsertMiddle(t *testing.T) {
	oldNode := mustParse(t, `<ul><li data-key="a">A</li><li data-key="c">C</li></ul>`)
	newNode := mustParse(t, `<ul><li data-key="a">A</li><li data-key="b">B</li><li data-key="c">C</li></ul>`)

	patches := Diff(oldNode, newNode)
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != OpInsert {
		t.Fatalf("op = %q, want %q", patches[0].Op, OpInsert)
	}
	if patches[0].Key != "b" {
		t.Fatalf("key = %q, want b", patches[0].Key)
	}
}

func TestDiff_KeyedList_Remove(t *testing.T) {
	oldNode := mustParse(t, `<ul><li data-key="a">A</li><li data-key="b">B</li><li data-key="c">C</li></ul>`)
	newNode := mustParse(t, `<ul><li data-key="a">A</li><li data-key="c">C</li></ul>`)

	patches := Diff(oldNode, newNode)
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != OpRemove {
		t.Fatalf("op = %q, want %q", patches[0].Op, OpRemove)
	}
	if patches[0].Key != "b" {
		t.Fatalf("key = %q, want b", patches[0].Key)
	}
}
