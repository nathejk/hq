package stack_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"nathejk.dk/pkg/job/stack"
)

func TestSyncedStack(t *testing.T) {
	s := stack.NewSyncedStack(stack.New(), false)

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

func TestSyncedIter(t *testing.T) {
	s := stack.NewSyncedStack(stack.New(), false)

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
func TestSyncedStackBlocking(t *testing.T) {
	s := stack.NewSyncedStack(stack.New(), true)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		v, ok := s.Pop()
		assert.True(t, ok, "Expected popped value, got none")
		assert.Equal(t, "test1", v)
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	s.Push("test1")
	wg.Wait()

	s.Unblock()
	v, ok := s.Pop()
	assert.False(t, ok, "Expected no popped value, but got one")
	assert.Equal(t, nil, v)
	assert.Equal(t, 0, s.Len())

	s.Block()
	wg.Add(1)
	go func() {
		v, ok := s.Pop()
		assert.True(t, ok, "Expected popped value, got none")
		assert.Equal(t, "test2", v)
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	s.Push("test2")
	wg.Wait()

	wg.Add(1)
	go func() {
		v, ok := s.Pop()
		assert.False(t, ok, "Expected no popped value, but got one")
		assert.Equal(t, nil, v)
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	s.Unblock()
	wg.Wait()
}
