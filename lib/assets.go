package mq

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// schemeRe matches a URL scheme at the very start (e.g. "https:", "mailto:").
var schemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// AssetSummary tallies the non-text content of a document for the tree footer.
type AssetSummary struct {
	Images        int
	Figures       int
	Tables        int
	Lists         int
	Links         int
	LinksExternal int
	LinksRelative int
	SVGCount      int
	SVGBytes      int
	Media         map[string]int
	CodeByLang    map[string]int
	CodeTotal     int
}

// BuildAssets tallies the document's non-text content.
func (d *Document) BuildAssets() *AssetSummary {
	a := &AssetSummary{
		Images:     len(d.GetImages()),
		Figures:    len(d.Figures()),
		Tables:     len(d.GetTables()),
		Lists:      len(d.GetLists(nil)),
		SVGCount:   d.SVGCount(),
		SVGBytes:   d.SVGBytes(),
		Media:      d.Media(),
		CodeByLang: map[string]int{},
	}

	for _, l := range d.GetLinks() {
		a.Links++
		if isExternalURL(l.URL) {
			a.LinksExternal++
		} else {
			a.LinksRelative++
		}
	}

	for lang, count := range CountCodeByLanguage(d.GetCodeBlocks()) {
		a.CodeByLang[lang] = count
		a.CodeTotal += count
	}

	return a
}

// isExternalURL reports whether a link points off-document. It matches a scheme
// at the START (so "/redirect?url=http://x" stays relative) or a protocol-relative
// "//host" prefix.
func isExternalURL(u string) bool {
	return strings.HasPrefix(u, "//") || schemeRe.MatchString(u)
}

// Render returns the "Assets" footer block, honoring --only/--drop kind filters.
// Returns "" when there is nothing to show.
func (a *AssetSummary) Render(opts TreeOptions) string {
	type row struct{ label, value string }
	var rows []row

	add := func(kind, label, value string) {
		if value == "" || !kindAllowed(kind, opts) {
			return
		}
		rows = append(rows, row{label, value})
	}

	if a.Images > 0 {
		add("images", "Images", fmt.Sprintf("%d", a.Images))
	}
	if a.Figures > 0 {
		add("figures", "Figures", fmt.Sprintf("%d", a.Figures))
	}
	if a.Tables > 0 {
		add("tables", "Tables", fmt.Sprintf("%d", a.Tables))
	}
	if a.Lists > 0 {
		add("lists", "Lists", fmt.Sprintf("%d", a.Lists))
	}
	if a.CodeTotal > 0 {
		add("code", "Code", fmt.Sprintf("%d%s", a.CodeTotal, codeBreakdown(a.CodeByLang)))
	}
	if a.Links > 0 {
		add("links", "Links", fmt.Sprintf("%d (%d ext, %d rel)", a.Links, a.LinksExternal, a.LinksRelative))
	}
	if a.SVGCount > 0 {
		add("svg", "SVG", fmt.Sprintf("%d inline (%s, not shown)", a.SVGCount, humanBytes(a.SVGBytes)))
	}
	if len(a.Media) > 0 {
		add("media", "Media", fmt.Sprintf("%d %s [skipped]", mediaTotal(a.Media), mediaBreakdown(a.Media)))
	}

	if len(rows) == 0 {
		return ""
	}

	width := 0
	for _, r := range rows {
		if len(r.label) > width {
			width = len(r.label)
		}
	}

	var buf strings.Builder
	buf.WriteString("\nAssets\n")
	for _, r := range rows {
		buf.WriteString(fmt.Sprintf("  %-*s  %s\n", width, r.label, r.value))
	}
	return buf.String()
}

func codeBreakdown(byLang map[string]int) string {
	if len(byLang) == 0 {
		return ""
	}
	langs := make([]string, 0, len(byLang))
	for l := range byLang {
		if l != "" {
			langs = append(langs, l)
		}
	}
	if len(langs) == 0 {
		return ""
	}
	sort.Slice(langs, func(i, j int) bool { return byLang[langs[i]] > byLang[langs[j]] })
	parts := make([]string, 0, len(langs))
	for _, l := range langs {
		parts = append(parts, fmt.Sprintf("%s %d", l, byLang[l]))
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func mediaTotal(m map[string]int) int {
	total := 0
	for _, c := range m {
		total += c
	}
	return total
}

func mediaBreakdown(m map[string]int) string {
	kinds := make([]string, 0, len(m))
	for k := range m {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		parts = append(parts, fmt.Sprintf("%d %s", m[k], k))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func humanBytes(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("~%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("~%d KB", n/(1<<10))
	default:
		return fmt.Sprintf("~%d B", n)
	}
}
