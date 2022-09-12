package stack_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/pkg/job/stack"
)

type idstruct struct {
	id    string
	value string
}

func (id idstruct) ID() string {
	return id.id
}

func TestKeyedStack(t *testing.T) {
	s := stack.NewKeyedStack(stack.New())

	s.Push(idstruct{id: "key1", value: "value1"})
	s.Push(idstruct{id: "key2", value: "value2"})
	s.Push(idstruct{id: "key3", value: "value3"})
	s.Push(idstruct{id: "key2", value: "value4"})
	s.Push(idstruct{id: "key3", value: "value5"})
	s.Push(idstruct{id: "key4", value: "value6"})

	assert.Equal(t, 6, s.Len())

	v, ok := s.Pop()
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "value6", v.(idstruct).value)

	v, ok = s.Pop()
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "value5", v.(idstruct).value)

	v, ok = s.Pop()
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "value4", v.(idstruct).value)

	v, ok = s.Pop()
	assert.True(t, ok, "Expected popped value, got none")
	assert.Equal(t, "value1", v.(idstruct).value)

	v, ok = s.Pop()
	assert.False(t, ok, "Expected no popped value, but got one")
	assert.Equal(t, nil, v)

	assert.Equal(t, 2, s.SkipCount())
}
