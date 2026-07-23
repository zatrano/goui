package diff

import "testing"

func TestParseHTML_Basic(t *testing.T) {
	root, err := ParseHTML(`<div><span>Hello</span><p>World</p></div>`)
	if err != nil {
		t.Fatalf("ParseHTML: %v", err)
	}

	if len(root.Children) != 1 {
		t.Fatalf("root child count = %d, want 1", len(root.Children))
	}

	div := root.Children[0]
	if div.Tag != "div" {
		t.Fatalf("root child tag = %q, want div", div.Tag)
	}
	if len(div.Children) != 2 {
		t.Fatalf("div child count = %d, want 2", len(div.Children))
	}
	if div.Children[0].Tag != "span" || div.Children[0].Children[0].Text != "Hello" {
		t.Fatalf("unexpected first subtree: %+v", div.Children[0])
	}
	if div.Children[1].Tag != "p" || div.Children[1].Children[0].Text != "World" {
		t.Fatalf("unexpected second subtree: %+v", div.Children[1])
	}
}

func TestParseHTML_Attrs(t *testing.T) {
	root, err := ParseHTML(`<input type="email" class="field" data-key="user-1"></input>`)
	if err != nil {
		t.Fatalf("ParseHTML: %v", err)
	}

	input := root.Children[0]
	if input.Attrs["type"] != "email" {
		t.Fatalf("type attr = %q, want email", input.Attrs["type"])
	}
	if input.Attrs["class"] != "field" {
		t.Fatalf("class attr = %q, want field", input.Attrs["class"])
	}
	if input.Key != "user-1" {
		t.Fatalf("key = %q, want user-1", input.Key)
	}
}
