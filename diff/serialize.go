package diff

import (
	"bytes"
	"sort"
)

func Serialize(node *Node) string {
	if node == nil {
		return ""
	}

	var buf bytes.Buffer
	writeNode(&buf, node)
	return buf.String()
}

func writeNode(buf *bytes.Buffer, node *Node) {
	if node == nil {
		return
	}

	if node.Tag == "root" {
		for _, child := range node.Children {
			writeNode(buf, child)
		}
		return
	}

	if isTextNode(node) {
		writeEscapedText(buf, node.Text)
		return
	}

	buf.WriteByte('<')
	buf.WriteString(node.Tag)

	keys := make([]string, 0, len(node.Attrs))
	for key := range node.Attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		buf.WriteByte(' ')
		buf.WriteString(key)
		buf.WriteString(`="`)
		writeEscapedText(buf, node.Attrs[key])
		buf.WriteByte('"')
	}

	buf.WriteByte('>')
	for _, child := range node.Children {
		writeNode(buf, child)
	}
	buf.WriteString("</")
	buf.WriteString(node.Tag)
	buf.WriteByte('>')
}
