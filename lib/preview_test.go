package mq

import (
	"strings"
	"testing"
)

func TestSectionPreviewLinesWithRemainder(t *testing.T) {
	text := "# H\nAlpha line one.\nBeta line two.\nGamma line three."
	preview, rem := sectionPreview(text, TreeOptions{TrimN: 2, TrimUnit: "L"})
	if preview != "Alpha line one.\nBeta line two." {
		t.Fatalf("preview = %q", preview)
	}
	if rem != "+1L" {
		t.Fatalf("remainder = %q, want +1L", rem)
	}
}

func TestSectionPreviewFitsNoMarker(t *testing.T) {
	text := "# H\nOnly one line here."
	preview, rem := sectionPreview(text, TreeOptions{TrimN: 2, TrimUnit: "L"})
	if preview != "Only one line here." || rem != "" {
		t.Fatalf("preview=%q rem=%q; want whole line, no marker", preview, rem)
	}
}

func TestSectionPreviewFull(t *testing.T) {
	text := "# H\nl1\nl2\nl3"
	preview, rem := sectionPreview(text, TreeOptions{TrimFull: true})
	if preview != "l1\nl2\nl3" || rem != "" {
		t.Fatalf("preview=%q rem=%q; want all lines no marker", preview, rem)
	}
}

func TestSectionPreviewTail(t *testing.T) {
	text := "# H\nl1\nl2\nl3\nl4"
	preview, rem := sectionPreview(text, TreeOptions{TrimN: 1, TrimUnit: "L", TrimTail: true})
	if preview != "l4" {
		t.Fatalf("tail preview = %q, want l4", preview)
	}
	if rem != "+3L" {
		t.Fatalf("tail remainder = %q, want +3L", rem)
	}
}

func TestSectionPreviewParagraphs(t *testing.T) {
	text := "# H\npara one line a\npara one line b\n\npara two\n\npara three"
	preview, rem := sectionPreview(text, TreeOptions{TrimN: 1, TrimUnit: "P"})
	if preview != "para one line a para one line b" {
		t.Fatalf("para preview = %q", preview)
	}
	if rem != "+2P" {
		t.Fatalf("para remainder = %q, want +2P", rem)
	}
}

func TestSectionPreviewMoreOff(t *testing.T) {
	text := "# H\nl1\nl2\nl3"
	_, rem := sectionPreview(text, TreeOptions{TrimN: 1, TrimUnit: "L", More: "off"})
	if rem != "" {
		t.Fatalf("remainder = %q, want empty with --more off", rem)
	}
}

func TestSectionPreviewZeroTrim(t *testing.T) {
	text := "# H\nl1\nl2"
	preview, rem := sectionPreview(text, TreeOptions{TrimN: 0, TrimUnit: "L"})
	if preview != "" || rem != "" {
		t.Fatalf("preview=%q rem=%q; want empty for --trim 0", preview, rem)
	}
}

func TestSectionPreviewChars(t *testing.T) {
	text := "# H\n" + strings.Repeat("word ", 20)
	preview, rem := sectionPreview(text, TreeOptions{TrimN: 30, TrimUnit: "C"})
	if len([]rune(preview)) > 30 {
		t.Fatalf("char preview too long: %d", len([]rune(preview)))
	}
	if !strings.HasPrefix(rem, "+") || !strings.HasSuffix(rem, "C") {
		t.Fatalf("char remainder = %q, want +NC", rem)
	}
}

func TestFilterTreeLevelsDepth(t *testing.T) {
	// h1 > h2 > h3; --depth 2 drops h3, keeping the spine connected.
	h3 := &TreeNode{Type: "section", Level: 3, Text: "h3"}
	h2 := &TreeNode{Type: "section", Level: 2, Text: "h2", Children: []*TreeNode{h3}}
	h1 := &TreeNode{Type: "section", Level: 1, Text: "h1", Children: []*TreeNode{h2}}

	out := filterTreeLevels([]*TreeNode{h1}, TreeOptions{MaxLevel: 2})
	if len(out) != 1 || len(out[0].Children) != 1 {
		t.Fatalf("unexpected shape: %+v", out)
	}
	if got := out[0].Children[0]; got.Text != "h2" || len(got.Children) != 0 {
		t.Fatalf("h2 should be kept with no children, got %+v", got)
	}
}

func TestFilterTreeLevelsReparent(t *testing.T) {
	// Drop the middle h2; its h3 child reparents to h1.
	h3 := &TreeNode{Type: "section", Level: 3, Text: "h3"}
	h2 := &TreeNode{Type: "section", Level: 2, Text: "h2", Children: []*TreeNode{h3}}
	h1 := &TreeNode{Type: "section", Level: 1, Text: "h1", Children: []*TreeNode{h2}}

	out := filterTreeLevels([]*TreeNode{h1}, TreeOptions{Drop: map[string]bool{"h2": true}})
	if len(out) != 1 || len(out[0].Children) != 1 || out[0].Children[0].Text != "h3" {
		t.Fatalf("h3 should reparent to h1, got %+v", out[0].Children)
	}
}

func TestBuildAssetsLinkSplitAndFilter(t *testing.T) {
	doc := NewDocument(
		[]byte(""), "x.html", FormatHTML, "t",
		nil, nil, nil,
		[]*Link{{URL: "https://a.com"}, {URL: "/local"}, {URL: "mailto:x@y.z"}},
		[]*Image{{URL: "a.png"}, {URL: "b.png"}},
		[]*Table{{}},
		nil, "",
	)
	doc.SetAssets([]*Figure{{ImageURL: "f.svg"}}, 3, 4096, map[string]int{"video": 1})

	a := doc.BuildAssets()
	if a.Links != 3 || a.LinksExternal != 2 || a.LinksRelative != 1 {
		t.Fatalf("link split wrong: total=%d ext=%d rel=%d", a.Links, a.LinksExternal, a.LinksRelative)
	}
	if a.Figures != 1 || a.SVGCount != 3 || a.Images != 2 {
		t.Fatalf("counts wrong: fig=%d svg=%d img=%d", a.Figures, a.SVGCount, a.Images)
	}

	// --only images,tables hides everything else.
	out := a.Render(TreeOptions{Assets: true, Only: map[string]bool{"images": true, "tables": true}})
	if !strings.Contains(out, "Images") || !strings.Contains(out, "Tables") {
		t.Fatalf("filtered render missing kept rows:\n%s", out)
	}
	if strings.Contains(out, "SVG") || strings.Contains(out, "Links") || strings.Contains(out, "Figures") {
		t.Fatalf("filtered render leaked dropped rows:\n%s", out)
	}
}
