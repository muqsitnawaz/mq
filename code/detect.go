package code

import (
	"path/filepath"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/odvcencio/gotreesitter/grammars"
)

func init() {
	mq.ExtraDetectors = append(mq.ExtraDetectors, detectCode)
}

// Extensions already handled by dedicated mq parsers (markdown, html, pdf, data, office).
// Tree-sitter has grammars for some of these (markdown, json, yaml, html, css) but
// mq's dedicated parsers are better suited, so we skip them.
var skipExtensions = map[string]struct{}{
	".md": {}, ".markdown": {}, ".mdown": {}, ".mkd": {},
	".html": {}, ".htm": {}, ".xhtml": {},
	".pdf":  {},
	".json": {}, ".jsonl": {}, ".ndjson": {},
	".yaml": {}, ".yml": {},
	".docx": {}, ".xlsx": {}, ".csv": {}, ".tsv": {}, ".pptx": {},
	".txt": {}, ".text": {}, ".log": {},
}

func detectCode(path string, _ []byte) (mq.Format, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, skip := skipExtensions[ext]; skip {
		return mq.FormatUnknown, false
	}
	if grammars.DetectLanguage(path) != nil {
		return mq.FormatCode, true
	}
	return mq.FormatUnknown, false
}
