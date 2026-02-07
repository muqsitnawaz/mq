package mq

import (
	"sort"
	"strings"
)

// Filter filters a slice based on a predicate function.
func Filter[T any](items []T, predicate func(T) bool) []T {
	var result []T
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms items from one type to another.
func Map[T, R any](items []T, transform func(T) R) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = transform(item)
	}
	return result
}

// FlatMap maps and flattens the results.
func FlatMap[T, R any](items []T, transform func(T) []R) []R {
	var result []R
	for _, item := range items {
		result = append(result, transform(item)...)
	}
	return result
}

// Reduce reduces items to a single value.
func Reduce[T, R any](items []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, item := range items {
		result = reducer(result, item)
	}
	return result
}

// Take returns the first n items.
func Take[T any](items []T, n int) []T {
	if n <= 0 {
		return []T{}
	}
	if n >= len(items) {
		return items
	}
	return items[:n]
}

// Skip skips the first n items.
func Skip[T any](items []T, n int) []T {
	if n <= 0 {
		return items
	}
	if n >= len(items) {
		return []T{}
	}
	return items[n:]
}

// Unique returns unique items.
func Unique[T comparable](items []T) []T {
	seen := make(map[T]bool)
	var result []T
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// UniqueBy returns unique items based on a key function.
func UniqueBy[T any, K comparable](items []T, keyFunc func(T) K) []T {
	seen := make(map[K]bool)
	var result []T
	for _, item := range items {
		key := keyFunc(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

// GroupBy groups items by a key function.
func GroupBy[T any, K comparable](items []T, keyFunc func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, item := range items {
		key := keyFunc(item)
		result[key] = append(result[key], item)
	}
	return result
}

// SortBy sorts items based on a comparison function.
func SortBy[T any](items []T, less func(a, b T) bool) []T {
	result := make([]T, len(items))
	copy(result, items)
	sort.Slice(result, func(i, j int) bool {
		return less(result[i], result[j])
	})
	return result
}

// Find returns the first item matching the predicate.
func Find[T any](items []T, predicate func(T) bool) (T, bool) {
	var zero T
	for _, item := range items {
		if predicate(item) {
			return item, true
		}
	}
	return zero, false
}

// Any returns true if any item matches the predicate.
func Any[T any](items []T, predicate func(T) bool) bool {
	for _, item := range items {
		if predicate(item) {
			return true
		}
	}
	return false
}

// All returns true if all items match the predicate.
func All[T any](items []T, predicate func(T) bool) bool {
	for _, item := range items {
		if !predicate(item) {
			return false
		}
	}
	return true
}

// Chain provides a fluent interface for chaining operations.
type Chain[T any] struct {
	items []T
}

// NewChain creates a new chain with the given items.
func NewChain[T any](items []T) *Chain[T] {
	return &Chain[T]{items: items}
}

// Filter applies a filter predicate.
func (c *Chain[T]) Filter(predicate func(T) bool) *Chain[T] {
	c.items = Filter(c.items, predicate)
	return c
}

// Take limits the results to n items.
func (c *Chain[T]) Take(n int) *Chain[T] {
	c.items = Take(c.items, n)
	return c
}

// Skip skips the first n items.
func (c *Chain[T]) Skip(n int) *Chain[T] {
	c.items = Skip(c.items, n)
	return c
}

// SortBy sorts items.
func (c *Chain[T]) SortBy(less func(a, b T) bool) *Chain[T] {
	c.items = SortBy(c.items, less)
	return c
}

// Result returns the final items.
func (c *Chain[T]) Result() []T {
	return c.items
}

// Count returns the number of items.
func (c *Chain[T]) Count() int {
	return len(c.items)
}

// First returns the first item.
func (c *Chain[T]) First() (T, bool) {
	var zero T
	if len(c.items) == 0 {
		return zero, false
	}
	return c.items[0], true
}

// Specific operators for document elements

// FilterHeadingsByLevel filters headings by their level.
func FilterHeadingsByLevel(headings []*Heading, levels ...int) []*Heading {
	if len(levels) == 0 {
		return headings
	}

	levelSet := make(map[int]bool)
	for _, level := range levels {
		levelSet[level] = true
	}

	return Filter(headings, func(h *Heading) bool {
		return levelSet[h.Level]
	})
}

// FilterHeadingsByText filters headings by text pattern.
func FilterHeadingsByText(headings []*Heading, pattern string) []*Heading {
	return Filter(headings, func(h *Heading) bool {
		return strings.Contains(strings.ToLower(h.Text), strings.ToLower(pattern))
	})
}

// FilterCodeBlocksByLanguage filters code blocks by language.
func FilterCodeBlocksByLanguage(blocks []*CodeBlock, languages ...string) []*CodeBlock {
	if len(languages) == 0 {
		return blocks
	}

	langSet := make(map[string]bool)
	for _, lang := range languages {
		langSet[strings.ToLower(lang)] = true
	}

	return Filter(blocks, func(cb *CodeBlock) bool {
		return langSet[strings.ToLower(cb.Language)]
	})
}

// FilterCodeBlocksByLines filters code blocks by line count.
func FilterCodeBlocksByLines(blocks []*CodeBlock, minLines int) []*CodeBlock {
	return Filter(blocks, func(cb *CodeBlock) bool {
		return cb.GetLines() >= minLines
	})
}

// MapHeadingsToText extracts text from headings.
func MapHeadingsToText(headings []*Heading) []string {
	return Map(headings, func(h *Heading) string {
		return h.Text
	})
}

// MapSectionsToText extracts text from sections.
func MapSectionsToText(sections []*Section) []string {
	return Map(sections, func(s *Section) string {
		return s.GetText()
	})
}

// MapCodeBlocksToContent extracts content from code blocks.
func MapCodeBlocksToContent(blocks []*CodeBlock) []string {
	return Map(blocks, func(cb *CodeBlock) string {
		return cb.Content
	})
}

func CountCodeByLanguage(blocks []*CodeBlock) map[string]int {
	counts := make(map[string]int)
	for _, cb := range blocks {
		lang := cb.Language
		if lang == "" {
			lang = "plain"
		}
		counts[lang]++
	}
	return counts
}

// HeadingPredicate creates a predicate for filtering headings.
type HeadingPredicate func(*Heading) bool

// CombinePredicates combines multiple predicates with AND logic.
func CombinePredicates(predicates ...HeadingPredicate) HeadingPredicate {
	return func(h *Heading) bool {
		for _, p := range predicates {
			if !p(h) {
				return false
			}
		}
		return true
	}
}

// ByLevel creates a predicate for heading level.
func ByLevel(levels ...int) HeadingPredicate {
	levelSet := make(map[int]bool)
	for _, level := range levels {
		levelSet[level] = true
	}
	return func(h *Heading) bool {
		return levelSet[h.Level]
	}
}

// ByTextContains creates a predicate for text containing a pattern.
func ByTextContains(pattern string) HeadingPredicate {
	lower := strings.ToLower(pattern)
	return func(h *Heading) bool {
		return strings.Contains(strings.ToLower(h.Text), lower)
	}
}

// ByTextPrefix creates a predicate for text starting with a prefix.
func ByTextPrefix(prefix string) HeadingPredicate {
	lower := strings.ToLower(prefix)
	return func(h *Heading) bool {
		return strings.HasPrefix(strings.ToLower(h.Text), lower)
	}
}
