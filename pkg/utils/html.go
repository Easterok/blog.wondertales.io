package utils

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func extractText(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}

	var result strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		result.WriteString(extractText(child))
	}
	return result.String()
}

type Anchor struct {
	Id   string
	Text string
}

func ExtractAnchors(s ...string) []Anchor {
	result := []Anchor{}

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			isJumpAnchor := false
			id := ""

			for _, attr := range node.Attr {
				if attr.Key == "data-type" && attr.Val == "jump-anchor" {
					isJumpAnchor = true
				}

				if attr.Key == "id" {
					id = attr.Val
				}
			}

			if isJumpAnchor && id != "" {
				txt := extractText(node)

				result = append(result, Anchor{
					Id:   id,
					Text: txt,
				})
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	for _, item := range s {
		doc, err := html.Parse(strings.NewReader(item))

		if err != nil {
			continue
		}

		traverse(doc)
	}

	return result
}

type AltLink struct {
	XMLName  xml.Name `xml:"xhtml:link"`
	Rel      string   `xml:"rel,attr"`
	Hreflang string   `xml:"hreflang,attr"`
	Href     string   `xml:"href,attr"`
}

type URL struct {
	Loc      string    `xml:"loc"`
	LastMod  string    `xml:"lastmod"`
	Priority string    `xml:"priority"`
	AltLinks []AltLink `xml:"xhtml:link"`
}

type Sitemap struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	Xhtml   string   `xml:"xmlns:xhtml,attr"`
	URLs    []URL    `xml:"url"`
}

var XmlHeader = []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")

func prepUrl(s string) string {
	if strings.HasPrefix(s, "https://www.") {
		return strings.TrimSuffix(s, "/")
	}

	return strings.TrimSuffix(strings.Replace(s, "https://", "https://www.", 1), "/")
}

func NewSitemap(base string, urls []URL, alternate []string) Sitemap {
	res := []URL{}

	dict := map[string]bool{}

	for _, url := range urls {
		alt := []AltLink{}

		for _, lang := range alternate {
			alt = append(alt, AltLink{
				Rel:      "alternate",
				Hreflang: lang,
				Href:     prepUrl(fmt.Sprintf("%s/%s/%s", base, lang, url.Loc)),
			})
		}

		alt = append(alt, AltLink{
			Rel:      "alternate",
			Hreflang: "x-default",
			Href:     prepUrl(fmt.Sprintf("%s/%s", base, url.Loc)),
		})

		loc := prepUrl(fmt.Sprintf("%s/%s", base, url.Loc))

		_, ok := dict[loc]

		if !ok {
			res = append(res, URL{
				Loc:      loc,
				LastMod:  url.LastMod,
				Priority: url.Priority,
				AltLinks: alt,
			})

			dict[loc] = true
		}
	}

	return Sitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Xhtml: "http://www.w3.org/1999/xhtml",
		URLs:  res,
	}
}

var linkre = regexp.MustCompile(`(?i)<a[^>]*>(.*?)<\/a>`)

func RemoveLinks(s string) string {
	return linkre.ReplaceAllString(s, "$1")
}
