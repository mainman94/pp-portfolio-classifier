package ppxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

type Node struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Children []*Node
	Text     string
}

func ParseFile(path string) (*Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*Node, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var stack []*Node
	var root *Node
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			node := &Node{Name: tok.Name, Attrs: append([]xml.Attr(nil), tok.Attr...)}
			if len(stack) == 0 {
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("unexpected end element %s", tok.Name.Local)
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			text := string(tok)
			if strings.TrimSpace(text) == "" {
				continue
			}
			current := stack[len(stack)-1]
			current.Text += text
		}
	}
	return root, nil
}

func (n *Node) WriteFile(path string) error {
	data, err := n.Bytes()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (n *Node) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := n.encode(enc); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (n *Node) encode(enc *xml.Encoder) error {
	start := xml.StartElement{Name: n.Name, Attr: n.Attrs}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if n.Text != "" {
		if err := enc.EncodeToken(xml.CharData([]byte(n.Text))); err != nil {
			return err
		}
	}
	for _, child := range n.Children {
		if err := child.encode(enc); err != nil {
			return err
		}
	}
	return enc.EncodeToken(start.End())
}

func (n *Node) Attr(name string) string {
	for _, attr := range n.Attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func (n *Node) SetAttr(name, value string) {
	for i := range n.Attrs {
		if n.Attrs[i].Name.Local == name {
			n.Attrs[i].Value = value
			return
		}
	}
	n.Attrs = append(n.Attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

func (n *Node) Child(name string) *Node {
	for _, child := range n.Children {
		if child.Name.Local == name {
			return child
		}
	}
	return nil
}

func (n *Node) ChildrenNamed(name string) []*Node {
	var out []*Node
	for _, child := range n.Children {
		if child.Name.Local == name {
			out = append(out, child)
		}
	}
	return out
}

func (n *Node) EnsureChild(name string) *Node {
	if child := n.Child(name); child != nil {
		return child
	}
	child := &Node{Name: xml.Name{Local: name}}
	n.Children = append(n.Children, child)
	return child
}

func (n *Node) Append(child *Node) {
	n.Children = append(n.Children, child)
}

func (n *Node) RemoveChild(target *Node) {
	for i, child := range n.Children {
		if child == target {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			return
		}
	}
}

func TextNode(name, text string) *Node {
	return &Node{Name: xml.Name{Local: name}, Text: text}
}
