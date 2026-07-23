package forms

import (
	"context"
	"strings"
	"testing"
)

func TestDragDropUpload_Uploaded(t *testing.T) {
	u := &DragDropUpload{
		CommonAttrs: CommonAttrs{Name: "docs"},
		EventName:   "docs",
		Multiple:    true,
	}
	ctx := context.Background()
	_ = u.HandleEvent(ctx, "docs.uploaded", map[string]any{
		"id": "abc", "name": "a.pdf", "url": "/goui/files/abc", "size": "12", "contentType": "application/pdf",
	})
	if len(u.Files) != 1 || u.Files[0].ID != "abc" {
		t.Fatalf("%#v", u.Files)
	}
	html, _ := u.Render()
	if !strings.Contains(html, "data-goui-upload") || !strings.Contains(html, "a.pdf") {
		t.Fatalf("%s", html)
	}
	_ = u.HandleEvent(ctx, "docs.remove", map[string]any{"value": "abc"})
	if len(u.Files) != 0 {
		t.Fatal(u.Files)
	}
}

func TestAvatarUpload_Uploaded(t *testing.T) {
	a := &AvatarUpload{CommonAttrs: CommonAttrs{Name: "av"}, EventName: "av"}
	_ = a.HandleEvent(context.Background(), "av.uploaded", map[string]any{
		"id": "x", "name": "avatar.png", "url": "/goui/files/x", "size": "100", "contentType": "image/png",
	})
	if a.Avatar.URL != "/goui/files/x" {
		t.Fatal(a.Avatar)
	}
	html, _ := a.Render()
	if !strings.Contains(html, "goui-crop-canvas") || !strings.Contains(html, "/goui/files/x") {
		t.Fatalf("%s", html)
	}
}
