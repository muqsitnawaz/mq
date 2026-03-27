package mq

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// FlattenStructuredData renders nested structured data as deterministic path:value lines.
func FlattenStructuredData(value interface{}) string {
	var lines []string
	appendStructuredLines(&lines, "", value)
	return strings.Join(lines, "\n")
}

func appendStructuredLines(lines *[]string, path string, value interface{}) {
	if value == nil {
		appendStructuredLine(lines, path, "null")
		return
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		appendStructuredLine(lines, path, "null")
		return
	}

	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			appendStructuredLine(lines, path, "null")
			return
		}
		appendStructuredLines(lines, path, rv.Elem().Interface())
	case reflect.Map:
		if rv.Len() == 0 {
			return
		}
		type mapEntry struct {
			key   string
			value reflect.Value
		}
		entries := make([]mapEntry, 0, rv.Len())
		for _, key := range rv.MapKeys() {
			entries = append(entries, mapEntry{
				key:   fmt.Sprint(key.Interface()),
				value: rv.MapIndex(key),
			})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].key < entries[j].key
		})
		for _, entry := range entries {
			appendStructuredLines(lines, joinStructuredPath(path, entry.key), entry.value.Interface())
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			appendStructuredLines(lines, joinStructuredIndex(path, i), rv.Index(i).Interface())
		}
	default:
		appendStructuredLine(lines, path, formatStructuredScalar(value))
	}
}

func appendStructuredLine(lines *[]string, path, value string) {
	if value == "" {
		return
	}
	if path == "" {
		*lines = append(*lines, value)
		return
	}
	*lines = append(*lines, path+": "+value)
}

func joinStructuredPath(path, key string) string {
	if path == "" {
		if isIdentifierLike(key) {
			return key
		}
		return fmt.Sprintf("[%s]", strconv.Quote(key))
	}
	if isIdentifierLike(key) {
		return path + "." + key
	}
	return fmt.Sprintf("%s[%s]", path, strconv.Quote(key))
}

func joinStructuredIndex(path string, index int) string {
	if path == "" {
		return fmt.Sprintf("[%d]", index)
	}
	return fmt.Sprintf("%s[%d]", path, index)
}

func isIdentifierLike(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func formatStructuredScalar(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.ReplaceAll(v, "\n", "\\n")
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		if v == float32(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return "null"
	default:
		return fmt.Sprint(v)
	}
}
