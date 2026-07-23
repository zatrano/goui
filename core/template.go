package core

import (
	"bytes"
	"hash/fnv"
	"html/template"
	"sync"
	"sync/atomic"
)

var (
	templateCache sync.Map
	parseCount    atomic.Int64
)

// RenderTemplate parses and executes an HTML template with the given data.
// Parsed templates are cached by the hash of the template string.
// Pass T via data for i18n: {{call .T "key"}} or {{call .T "key" .}} with placeholders.
func RenderTemplate(tmplStr string, data any) (string, error) {
	key := templateKey(tmplStr)

	cached, ok := templateCache.Load(key)
	if !ok {
		tmpl, err := template.New("").Parse(tmplStr)
		if err != nil {
			return "", err
		}
		parseCount.Add(1)

		actual, loaded := templateCache.LoadOrStore(key, tmpl)
		if loaded {
			cached = actual
		} else {
			cached = tmpl
		}
	}

	var buf bytes.Buffer
	if err := cached.(*template.Template).Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func templateKey(tmplStr string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(tmplStr))
	return h.Sum64()
}
