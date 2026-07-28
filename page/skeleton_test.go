package page_test

import (
	"strings"
	"testing"

	"github.com/zatrano/goui/v2/page"
)

func TestSkeletonBlock(t *testing.T) {
	html := page.SkeletonBlock("12rem")
	if !strings.Contains(html, `min-height:12rem`) || !strings.Contains(html, `goui-skeleton`) {
		t.Fatalf("unexpected skeleton: %s", html)
	}
}

func TestSkeletonRows(t *testing.T) {
	html := page.SkeletonRows("20rem", 4)
	if strings.Count(html, "goui-skeleton-row") != 4 {
		t.Fatalf("want 4 rows: %s", html)
	}
}
