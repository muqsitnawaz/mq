package mql

import mq "github.com/muqsitnawaz/mq/lib"

// BuildDirTree creates a directory tree across all formats supported by mql.Engine.
func BuildDirTree(dirPath string, mode mq.TreeMode) (*mq.DirTreeResult, error) {
	engine := New()
	return mq.BuildDirTreeWithLoader(dirPath, mq.DefaultTreeOptions(mode), engine.LoadDocument)
}

// BuildDirTreeWithOptions creates a directory tree with depth/limit bounds.
func BuildDirTreeWithOptions(dirPath string, opts mq.TreeOptions) (*mq.DirTreeResult, error) {
	engine := New()
	return mq.BuildDirTreeWithLoader(dirPath, opts, engine.LoadDocument)
}

// SearchDir searches a directory across all formats supported by mql.Engine.
func SearchDir(dirPath string, query string) (*mq.SearchResults, error) {
	engine := New()
	return mq.SearchDirWithLoader(dirPath, query, engine.LoadDocument)
}
