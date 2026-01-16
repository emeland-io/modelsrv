package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.emeland.io/modelsrv/internal/util"
)

func TestEnqueueDequeue(t *testing.T) {
	cq := util.NewCircularQueue[string](3)
	assert.Equal(t, 0, cq.Length())

	err := cq.Enqueue("a")
	assert.NoError(t, err)
	assert.Equal(t, 1, cq.Length())

	err = cq.Enqueue("b")
	assert.NoError(t, err)
	assert.Equal(t, 2, cq.Length())

	val, err := cq.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, "a", val)
	assert.Equal(t, 1, cq.Length())
}
func TestPush(t *testing.T) {
	cq := util.NewCircularQueue[string](3)
	assert.Equal(t, 0, cq.Length())

	err := cq.Enqueue("a")
	assert.NoError(t, err)
	assert.Equal(t, 1, cq.Length())

	err = cq.Push("b")
	assert.NoError(t, err)
	assert.Equal(t, 2, cq.Length())

	val, err := cq.Dequeue()
	assert.NoError(t, err)
	assert.Equal(t, "b", val)
	assert.Equal(t, 1, cq.Length())
}

func TestPeek(t *testing.T) {
	cq := util.NewCircularQueue[string](3)
	assert.Equal(t, 0, cq.Length())

	err := cq.Enqueue("a")
	assert.NoError(t, err)
	assert.Equal(t, 1, cq.Length())

	err = cq.Push("b")
	assert.NoError(t, err)
	assert.Equal(t, 2, cq.Length())

	val, err := cq.Peek()
	assert.NoError(t, err)
	assert.Equal(t, "b", val)
	assert.Equal(t, 2, cq.Length())
}
