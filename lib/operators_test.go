package mq_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mq "github.com/muqsitnawaz/mq/lib"
)

func TestReduce(t *testing.T) {
	sum := mq.Reduce([]int{1, 2, 3, 4}, 0, func(acc, item int) int {
		return acc + item
	})
	assert.Equal(t, 10, sum)

	concat := mq.Reduce([]string{"a", "b", "c"}, "", func(acc, item string) string {
		return acc + item
	})
	assert.Equal(t, "abc", concat)

	empty := mq.Reduce([]int{}, 42, func(acc, item int) int {
		return acc + item
	})
	assert.Equal(t, 42, empty, "Reduce on an empty slice should return the initial value")
}
