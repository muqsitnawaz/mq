package mq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlattenStructuredData(t *testing.T) {
	value := map[string]interface{}{
		"meta": map[string]interface{}{
			"user": "muqsit",
		},
		"ok": true,
		"items": []interface{}{
			map[string]interface{}{"name": "first"},
			42.0,
		},
		`key.with.dot`: "escaped",
	}

	text := FlattenStructuredData(value)

	assert.Equal(t, stringsJoin(
		`items[0].name: first`,
		`items[1]: 42`,
		`["key.with.dot"]: escaped`,
		`meta.user: muqsit`,
		`ok: true`,
	), text)
}

func stringsJoin(lines ...string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for _, line := range lines[1:] {
		result += "\n" + line
	}
	return result
}
