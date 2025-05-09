// Package extras provides extra Tengo modules.
package extras

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/d5/tengo/v2"
	"golang.org/x/net/html"
)

// ankoHtmlNode wraps an *html.Node for Tengo.
type ankoHtmlNode struct {
	tengo.ObjectImpl
	Value *html.Node
}

func (n *ankoHtmlNode) TypeName() string {
	return "html-node"
}

func (n *ankoHtmlNode) String() string {
	return htmlquery.OutputHTML(n.Value, true)
}

func (n *ankoHtmlNode) Copy() tengo.Object {
	return n
}

func (node *ankoHtmlNode) IndexGet(index tengo.Object) (tengo.Object, error) {
	k, _ := index.(*tengo.String)
	switch k.Value {
	case "remove_child":
		return &tengo.UserFunction{
			Name: "remove_child",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				targetNode, ok := args[0].(*ankoHtmlNode)
				if !ok {
					return nil, fmt.Errorf("remove_child: argument must be an html-node")
				}
				node.Value.RemoveChild(targetNode.Value)
				return &ankoHtmlNode{Value: node.Value}, nil
			},
		}, nil
	}
	return tengo.UndefinedValue, nil
}

func htmlModule(logger *slog.Logger) map[string]tengo.Object {
	return map[string]tengo.Object{
		"parse": &tengo.UserFunction{
			Name: "parse",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("html.parse: expected 1 argument")
				}
				htmlStr, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("html.parse: argument must be a string")
				}
				doc, err := htmlquery.Parse(strings.NewReader(htmlStr.Value))
				if err != nil {
					return nil, fmt.Errorf("html.parse: %w", err)
				}
				return &ankoHtmlNode{Value: doc}, nil
			},
		},
		"serialize": &tengo.UserFunction{
			Name: "serialize",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("html.serialize: expected 1 argument")
				}
				node, ok := args[0].(*ankoHtmlNode)
				if !ok {
					return nil, fmt.Errorf("html.serialize: argument must be an html-node")
				}
				return &tengo.String{Value: htmlquery.OutputHTML(node.Value, true)}, nil
			},
		},
		"query": &tengo.UserFunction{
			Name: "query",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("html.query: expected 2 arguments")
				}
				doc, ok1 := args[0].(*ankoHtmlNode)
				xpath, ok2 := args[1].(*tengo.String)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("html.query: arguments must be an html-node and a string")
				}
				if doc == nil {
					return nil, fmt.Errorf("html.query: cannot search within a nil node")
				}
				node, err := htmlquery.Query(doc.Value, xpath.Value)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("html.query: %w", err)
				}
				if node == nil {
					logger.Warn("Runtime", "func", "html.query", "message", "no element matched the provided XPath")
				}
				return &ankoHtmlNode{Value: node}, nil
			},
		},
		"query_text": &tengo.UserFunction{
			Name: "query_text",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("html.query: expected 2 arguments")
				}
				doc, ok1 := args[0].(*ankoHtmlNode)
				xpath, ok2 := args[1].(*tengo.String)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("html.query: arguments must be an html-node and a string")
				}
				if doc == nil {
					return nil, fmt.Errorf("html.query: cannot search within a nil node")
				}
				node, err := htmlquery.Query(doc.Value, xpath.Value)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("html.query: %w", err)
				}
				if node == nil {
					logger.Warn("Runtime", "func", "html.query", "message", "no element matched the provided XPath")
				}
				return &tengo.String{Value: htmlquery.InnerText(node)}, nil
			},
		},
		"query_all": &tengo.UserFunction{
			Name: "query_all",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("html.query_all: expected 2 arguments")
				}
				doc, ok := args[0].(*ankoHtmlNode)
				if !ok {
					return nil, fmt.Errorf("html.query_all: first argument must be an html-node")
				}
				xpath, ok := args[1].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("html.query_all: second argument must be a string")
				}
				if doc == nil {
					return nil, fmt.Errorf("html.query_all: cannot search within a nil node")
				}
				nodes, err := htmlquery.QueryAll(doc.Value, xpath.Value)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("html.query_all: %w", err)
				}
				arr := make([]tengo.Object, len(nodes))
				for i, node := range nodes {
					arr[i] = &ankoHtmlNode{Value: node}
				}
				return &tengo.Array{Value: arr}, nil
			},
		},
		"attr": &tengo.UserFunction{
			Name: "attr",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("html.attr: expected 2 arguments")
				}
				node, ok1 := args[0].(*ankoHtmlNode)
				name, ok2 := args[1].(*tengo.String)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("html.attr: arguments must be an html-node and a string")
				}
				if node.Value == nil {
					return nil, fmt.Errorf("html.attr: cannot extract attribute from a nil node")
				}
				attr := htmlquery.SelectAttr(node.Value, name.Value)
				if attr == "" {
					return tengo.UndefinedValue, nil
				}
				return &tengo.String{Value: attr}, nil
			},
		},
		"text": &tengo.UserFunction{
			Name: "text",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("html.text: expected 1 argument")
				}
				node, ok := args[0].(*ankoHtmlNode)
				if !ok {
					return nil, fmt.Errorf("html.text: argument must be an html-node")
				}
				if node.Value == nil {
					return nil, fmt.Errorf("html.text: cannot extract text from a nil node")
				}
				return &tengo.String{Value: htmlquery.InnerText(node.Value)}, nil
			},
		},
	}
}
