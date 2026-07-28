package page

import (
	"fmt"
	"html"
	"strings"
)

// SkeletonBlock returns minimal reserved-layout HTML for ModeLive/SEO timeout
// shells. Prefer matching the real page's min-height / grid to reduce CLS.
func SkeletonBlock(minHeightCSS string) string {
	if minHeightCSS == "" {
		minHeightCSS = "240px"
	}
	h := html.EscapeString(minHeightCSS)
	return fmt.Sprintf(
		`<div class="goui-skeleton" style="min-height:%s" aria-hidden="true"></div>`,
		h,
	)
}

// SkeletonRows is a simple stacked placeholder (header + n rows).
func SkeletonRows(minHeightCSS string, rows int) string {
	if rows < 1 {
		rows = 3
	}
	if minHeightCSS == "" {
		minHeightCSS = "320px"
	}
	h := html.EscapeString(minHeightCSS)
	var b strings.Builder
	b.WriteString(`<div class="goui-skeleton" style="min-height:`)
	b.WriteString(h)
	b.WriteString(`" aria-hidden="true">`)
	b.WriteString(`<div class="goui-skeleton-header" style="height:2rem;margin-bottom:1rem"></div>`)
	for i := 0; i < rows; i++ {
		b.WriteString(`<div class="goui-skeleton-row" style="height:1rem;margin-bottom:0.5rem"></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}
