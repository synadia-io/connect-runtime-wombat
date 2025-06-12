package main

import (
	"regexp"
	"strings"
)

var (
	asciidocLinkRegex          = regexp.MustCompile(`(https?://[^\[]+)\[([^\]]+)\]`)
	xrefLinkRegex              = regexp.MustCompile(`xref:([^\[]+)\[([^\]]+)\]`)
	angleBracketLinkRegex      = regexp.MustCompile(`<<([^,]+),([^>]+)>>`)
	singlAngleBracketLinkRegex = regexp.MustCompile(`<<([^>]+)>>`)
)

func replaceAsciidocLinksWithMarkdown(input string) string {
	input = asciidocLinkRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := asciidocLinkRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			url := parts[1]
			text := strings.TrimSuffix(parts[2], "^")
			return "[" + text + "](" + url + ")"
		}
		return match
	})

	input = xrefLinkRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := xrefLinkRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			//url := parts[1]
			text := strings.TrimSuffix(parts[2], "^")
			return text
		}
		return match
	})

	input = angleBracketLinkRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := angleBracketLinkRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			//url := parts[1]
			text := strings.TrimSuffix(parts[2], "^")
			return text
		}
		return match
	})

	input = singlAngleBracketLinkRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := singlAngleBracketLinkRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			//url := parts[1]
			text := strings.TrimSuffix(parts[1], "^")
			return text
		}
		return match
	})

	return input
}
