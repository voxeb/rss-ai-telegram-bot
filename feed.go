package main

import (
	"encoding/xml"
	"fmt"
	"strings"

	nethtml "golang.org/x/net/html"
)

// Normalize titles once, after XML decoding and before any downstream use.
// GUIDs and URLs are unchanged, so publication identities remain stable.
func parseFeed(data []byte) (feed, error) {
	var f feed
	if err := xml.Unmarshal(data, &f); err != nil {
		return feed{}, err
	}
	for i := range f.Entries {
		title, err := headlineText(f.Entries[i].Title)
		if err != nil {
			return feed{}, fmt.Errorf("RSS headline: %w", err)
		}
		f.Entries[i].Title = title
	}
	return f, nil
}

func headlineText(value string) (string, error) {
	// The HTML parser decodes entities in text nodes. Do not unescape first or
	// parse again: an encoded literal such as &lt;tag&gt; must remain text.
	root, err := nethtml.Parse(strings.NewReader(value))
	if err != nil {
		return "", err
	}
	var text strings.Builder
	var collect func(*nethtml.Node)
	collect = func(n *nethtml.Node) {
		if n.Type == nethtml.TextNode {
			text.WriteString(n.Data)
		}
		separator := false
		if n.Type == nethtml.ElementNode {
			switch n.Data {
			case "script", "style", "template":
				return
			case "br", "p", "div", "li", "h1", "h2", "h3", "h4", "h5", "h6":
				separator = true
				text.WriteByte(' ')
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
		if separator {
			text.WriteByte(' ')
		}
	}
	collect(root)
	return strings.Join(strings.Fields(text.String()), " "), nil
}
