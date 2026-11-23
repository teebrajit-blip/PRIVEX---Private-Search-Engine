package extractor

import (
	"bytes"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Document struct {
	URL         string
	Title       string
	Description string
	Headings    []string
	Body        string
	Language    string
	Links       []Link
	Canonical   string
}

type Link struct {
	URL        string
	AnchorText string
}

func Extract(baseURL string, r io.Reader) (Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return Document{}, err
	}
	base, _ := url.Parse(baseURL)
	doc := Document{URL: baseURL}
	walk(root, &doc, base)
	doc.Body = strings.Join(strings.Fields(doc.Body), " ")
	doc.Headings = unique(doc.Headings)
	if doc.Language == "" {
		doc.Language = "en"
	}
	return doc, nil
}

func walk(n *html.Node, doc *Document, base *url.URL) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "noscript", "template", "nav", "footer", "aside":
			return
		case "title":
			doc.Title = strings.TrimSpace(textContent(n))
		case "main", "article":
			if text := strings.TrimSpace(textContent(n)); len(doc.Body) < len(text) {
				doc.Body = text
			}
		case "h1", "h2", "h3":
			if text := strings.TrimSpace(textContent(n)); text != "" {
				doc.Headings = append(doc.Headings, text)
			}
		case "meta":
			name := attr(n, "name")
			content := strings.TrimSpace(attr(n, "content"))
			if strings.EqualFold(name, "description") {
				doc.Description = content
			}
			if strings.EqualFold(name, "robots") && strings.Contains(strings.ToLower(content), "noindex") {
				doc.Body = ""
			}
		case "html":
			if lang := attr(n, "lang"); lang != "" {
				doc.Language = strings.Split(lang, "-")[0]
			}
		case "a":
			href := attr(n, "href")
			if href != "" && base != nil {
				if u, err := url.Parse(href); err == nil {
					absolute := base.ResolveReference(u).String()
					doc.Links = append(doc.Links, Link{URL: absolute, AnchorText: strings.TrimSpace(textContent(n))})
				}
			}
		case "link":
			if strings.EqualFold(attr(n, "rel"), "canonical") {
				if href := attr(n, "href"); href != "" && base != nil {
					if u, err := url.Parse(href); err == nil {
						doc.Canonical = base.ResolveReference(u).String()
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, doc, base)
	}
	if n.Type == html.ElementNode && n.Data == "body" && doc.Body == "" {
		doc.Body = strings.TrimSpace(textContent(n))
	}
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attribs {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var b bytes.Buffer
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			b.WriteByte(' ')
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return b.String()
}

func unique(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}
