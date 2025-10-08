package htmlclean

import (
	"html"
	"strings"

	xhtml "golang.org/x/net/html"
)

const defaultMaxLength = 2048

// CleanHTML converts an HTML fragment into a normalized text representation.
// It strips tags (excluding their textual content), decodes HTML entities,
// collapses repeated whitespace, and ensures the returned string does not
// exceed the supplied maximum length (or a sensible default when max <= 0).
func CleanHTML(input string, max int) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	if max <= 0 {
		max = defaultMaxLength
	}

	node, err := xhtml.Parse(strings.NewReader(trimmed))
	if err != nil {
		fallback := cleanupPunctuation(collapseWhitespace(html.UnescapeString(trimmed)))
		return truncate(fallback, max)
	}

	var builder strings.Builder
	walk(node, &builder)

	cleaned := cleanupPunctuation(collapseWhitespace(builder.String()))
	return truncate(cleaned, max)
}

func walk(n *xhtml.Node, builder *strings.Builder) {
	if n == nil {
		return
	}

	if n.Type == xhtml.ElementNode {
		switch strings.ToLower(n.Data) {
		case "script", "style":
			return
		}
	}

	if n.Type == xhtml.TextNode {
		text := collapseWhitespace(html.UnescapeString(n.Data))
		if text != "" {
			if builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(text)
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		walk(child, builder)
	}
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

var punctuationReplacer = strings.NewReplacer(
	" !", "!",
	" ?", "?",
	" ,", ",",
	" .", ".",
	" ;", ";",
	" :", ":",
)

func cleanupPunctuation(s string) string {
	return punctuationReplacer.Replace(s)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return s
	}

	runes := []rune(s)
	if len(runes) <= max {
		return s
	}

	truncated := strings.TrimSpace(string(runes[:max]))
	if truncated == "" {
		// If trimming removed everything, fall back to raw slice to avoid
		// returning empty string when there is still content.
		truncated = string(runes[:max])
	}
	return truncated
}
