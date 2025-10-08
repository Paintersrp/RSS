package htmlclean

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"

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

	nodes, err := xhtml.ParseFragment(strings.NewReader(trimmed), nil)
	if err != nil {
		fallback := cleanupPunctuation(collapseWhitespace(html.UnescapeString(trimmed)))
		return truncate(fallback, max)
	}

	var builder strings.Builder
	var lastEndedWithSpace bool
	for _, n := range nodes {
		walk(n, &builder, &lastEndedWithSpace)
	}

	cleaned := cleanupPunctuation(collapseWhitespace(builder.String()))
	return truncate(cleaned, max)
}

func walk(n *xhtml.Node, builder *strings.Builder, lastEndedWithSpace *bool) {
	if n == nil {
		return
	}

	switch n.Type {
	case xhtml.ElementNode:
		name := strings.ToLower(n.Data)
		switch name {
		case "script", "style":
			return
		}
		defer func() {
			if lastEndedWithSpace != nil && isBlockElement(name) && builder.Len() > 0 {
				*lastEndedWithSpace = true
			}
		}()
	case xhtml.TextNode:
		appendText(builder, html.UnescapeString(n.Data), lastEndedWithSpace)
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		walk(child, builder, lastEndedWithSpace)
	}
}

func appendText(builder *strings.Builder, text string, lastEndedWithSpace *bool) {
	cleaned := collapseWhitespace(text)

	leadingWhitespace := hasLeadingWhitespace(text)
	trailingWhitespace := hasTrailingWhitespace(text)

	if cleaned == "" {
		if lastEndedWithSpace != nil {
			*lastEndedWithSpace = trailingWhitespace
		}
		return
	}

	needsSeparator := false
	if builder.Len() > 0 {
		if lastEndedWithSpace != nil && *lastEndedWithSpace {
			needsSeparator = true
		}
		if leadingWhitespace {
			needsSeparator = true
		}
	}

	if needsSeparator {
		builder.WriteByte(' ')
	}
	builder.WriteString(cleaned)

	if lastEndedWithSpace != nil {
		*lastEndedWithSpace = trailingWhitespace
	}
}

func hasLeadingWhitespace(s string) bool {
	for _, r := range s {
		return unicode.IsSpace(r)
	}
	return false
}

func hasTrailingWhitespace(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return unicode.IsSpace(r)
}

func isBlockElement(name string) bool {
	switch name {
	case "address", "article", "aside", "blockquote", "br", "div", "dl", "dt", "dd",
		"fieldset", "figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6",
		"header", "hr", "li", "main", "nav", "ol", "p", "pre", "section", "table", "tbody",
		"td", "tfoot", "th", "thead", "tr", "ul":
		return true
	default:
		return false
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
