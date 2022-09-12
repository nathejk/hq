package stack_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/pkg/job/stack"
)

func TestStack(t *testing.T) {
	s := stack.New()

	v, ok := s.Pop()
	assert.False(t, ok, "Expected no popped value, but got one")
	assert.Equal(t, nil, v)
	assert.Equal(t, 0, s.Len())

	s.Push("test1")
	s.Push("test2")
	assert.Equal(t, 2, s.Len())

	v, ok = s.Pop()
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "test2", v)
	assert.Equal(t, 1, s.Len())

	v, ok = s.Pop()
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "test1", v)
	assert.Equal(t, 0, s.Len())

	v, ok = s.Pop()
	assert.False(t, ok, "Expected no popped value, but got one")
	assert.Equal(t, nil, v)
	assert.Equal(t, 0, s.Len())
}

func TestIter(t *testing.T) {
	s := stack.New()

	c := stack.Iter(s)
	v, ok := <-c
	assert.False(t, ok, "Expected no popped value, but got one")
	assert.Equal(t, nil, v)
	assert.Equal(t, 0, s.Len())

	s.Push("test1")
	s.Push("test2")
	assert.Equal(t, 2, s.Len())

	c = stack.Iter(s)

	v, ok = <-c
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "test2", v)

	v, ok = <-c
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "test1", v)

	v, ok = <-c
	assert.False(t, ok, "Expected no popped value, but got one")
	assert.Equal(t, nil, v)
	assert.Equal(t, 0, s.Len())
}
