package mql

import mq "github.com/muqsitnawaz/mq/lib"

// BuildDirTree creates a directory tree across all formats supported by mql.Engine.
func BuildDirTree(dirPath string, mode mq.TreeMode) (*mq.DirTreeResult, error) {
	engine := New()
	return mq.BuildDirTreeWithLoader(dirPath, mode, engine.LoadDocument)
}

// SearchDir searches a directory across all formats supported by mql.Engine.
func SearchDir(dirPath string, query string) (*mq.SearchResults, error) {
	engine := New()
	return mq.SearchDirWithLoader(dirPath, query, engine.LoadDocument)
}
