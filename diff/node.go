package diff

import (
	"io"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Node struct {
	Tag      string
	Text     string
	Attrs    map[string]string
	Key      string
	Children []*Node
}

func ParseHTML(htmlStr string) (*Node, error) {
	root := &Node{Tag: "root", Attrs: map[string]string{}}
	context := &xhtml.Node{
		Type:     xhtml.ElementNode,
		Data:     atom.Div.String(),
		DataAtom: atom.Div,
	}

	nodes, err := xhtml.ParseFragment(strings.NewReader(htmlStr), context)
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		converted := convertHTMLNode(n)
		if converted != nil {
			root.Children = append(root.Children, converted)
		}
	}

	return root, nil
}

func convertHTMLNode(n *xhtml.Node) *Node {
	switch n.Type {
	case xhtml.ElementNode:
		node := &Node{
			Tag:   n.Data,
			Attrs: make(map[string]string, len(n.Attr)),
		}
		for _, attr := range n.Attr {
			node.Attrs[attr.Key] = attr.Val
			if attr.Key == "data-key" {
				node.Key = attr.Val
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			converted := convertHTMLNode(child)
			if converted != nil {
				node.Children = append(node.Children, converted)
			}
		}
		return node
	case xhtml.TextNode:
		if strings.TrimSpace(n.Data) == "" {
			return nil
		}
		return &Node{
			Text:  n.Data,
			Attrs: map[string]string{},
		}
	default:
		return nil
	}
}

func clonePath(path []int) []int {
	out := make([]int, len(path))
	copy(out, path)
	return out
}

func isTextNode(n *Node) bool {
	return n != nil && n.Tag == ""
}

func hasAnyKey(nodes []*Node) bool {
	for _, n := range nodes {
		if n != nil && n.Key != "" {
			return true
		}
	}
	return false
}

func writeEscapedText(w io.Writer, text string) {
	_, _ = io.WriteString(w, xhtml.EscapeString(text))
}
