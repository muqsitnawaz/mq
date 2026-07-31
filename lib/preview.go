package mq

import (
	"fmt"
	"regexp"
	"strings"
)

// bodyContent is the cleaned, previewable content of a section body (heading
// line removed, code fences and pure link/image lines dropped, emphasis stripped).
type bodyContent struct {
	lines []string // cleaned non-empty content lines, original breaks preserved
	paras []string // paragraphs (blank-line separated), each collapsed to one line
	flat  string   // all content joined with single spaces
}

var sentenceRe = regexp.MustCompile(`[^.!?]+[.!?]+`)

// sectionPreview renders a section's snippet and a "how much more" marker,
// honoring the --trim/--full/--more settings in opts. The marker is empty when
// nothing is hidden.
func sectionPreview(text string, opts TreeOptions) (preview, remainder string) {
	b := cleanBody(text)
	if len(b.lines) == 0 {
		return "", ""
	}
	if opts.TrimFull {
		return strings.Join(b.lines, "\n"), ""
	}
	unit := strings.ToUpper(opts.TrimUnit)
	if unit == "" {
		unit = "L"
	}
	n := opts.TrimN
	if n <= 0 {
		// --trim 0 / --bare: headings only, no preview.
		return "", ""
	}

	switch unit {
	case "P":
		return headTail(b.paras, n, opts.TrimTail, "P", opts)
	case "C":
		return previewByChars(b.flat, n, opts.TrimTail, opts)
	case "S":
		return headTail(splitSentences(b.flat), n, opts.TrimTail, "S", opts)
	case "W":
		return previewByWords(b.flat, n, opts.TrimTail, opts)
	default: // "L"
		return headTail(b.lines, n, opts.TrimTail, "L", opts)
	}
}

// headTail selects the first or last n items and builds the remainder marker.
func headTail(items []string, n int, tail bool, unit string, opts TreeOptions) (string, string) {
	total := len(items)
	if total == 0 {
		return "", ""
	}
	if n >= total {
		return strings.Join(items, "\n"), ""
	}
	var shown []string
	if tail {
		shown = items[total-n:]
	} else {
		shown = items[:n]
	}
	return strings.Join(shown, "\n"), remainderMark(total-n, unit, opts)
}

func previewByChars(flat string, n int, tail bool, opts TreeOptions) (string, string) {
	r := []rune(flat)
	if n >= len(r) {
		return flat, ""
	}
	var s string
	if tail {
		s = string(r[len(r)-n:])
		if i := strings.IndexByte(s, ' '); i > 0 && i < n/2 {
			s = s[i+1:]
		}
	} else {
		s = string(r[:n])
		if i := strings.LastIndexByte(s, ' '); i > n/2 {
			s = s[:i]
		}
	}
	hidden := len(r) - len([]rune(s))
	return s, remainderMark(hidden, "C", opts)
}

func previewByWords(flat string, n int, tail bool, opts TreeOptions) (string, string) {
	words := strings.Fields(flat)
	if n >= len(words) {
		return flat, ""
	}
	var shown []string
	if tail {
		shown = words[len(words)-n:]
	} else {
		shown = words[:n]
	}
	return strings.Join(shown, " "), remainderMark(len(words)-n, "W", opts)
}

func remainderMark(hidden int, unit string, opts TreeOptions) string {
	if hidden <= 0 || strings.EqualFold(opts.More, "off") {
		return ""
	}
	return fmt.Sprintf("+%d%s", hidden, unit)
}

func splitSentences(flat string) []string {
	matches := sentenceRe.FindAllString(flat, -1)
	var out []string
	for _, m := range matches {
		if s := strings.TrimSpace(m); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 && strings.TrimSpace(flat) != "" {
		out = []string{strings.TrimSpace(flat)}
	}
	return out
}

// cleanBody strips the heading line, code fences, and pure link/image lines,
// removes inline emphasis markers, and returns the content as lines/paragraphs/flat.
func cleanBody(text string) bodyContent {
	all := strings.Split(text, "\n")
	if len(all) < 2 {
		return bodyContent{}
	}

	var lines, paras, cur []string
	inCode := false
	flush := func() {
		if len(cur) > 0 {
			paras = append(paras, strings.Join(cur, " "))
			cur = nil
		}
	}

	for _, raw := range all[1:] {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "---") {
			continue
		}
		// Skip pure link/image lines.
		if strings.HasPrefix(line, "![") || (strings.HasPrefix(line, "[") && strings.Contains(line, "](")) {
			continue
		}
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "`", "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
		cur = append(cur, line)
	}
	flush()

	flat := strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	return bodyContent{lines: lines, paras: paras, flat: flat}
}
