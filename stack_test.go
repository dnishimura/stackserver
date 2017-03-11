package main

import (
	"bytes"
	"testing"
)

func TestSimpleStack(t *testing.T) {
	stack := newStack(100)
	Assert(t, stack.len == 0, "Stack should be empty")

	err := stack.push([]byte{0})
	Assert(t, err == nil && stack.len == 1, "Simple push should work")

	err = stack.push([]byte{1})
	err = stack.push([]byte{2})
	err = stack.push([]byte{3})
	Assert(t, err == nil && stack.len == 4, "More pushes should work")

	data, err := stack.pop()
	Assert(t, bytes.Equal(data, []byte{3}), "Top of stack should be 3")
	data, err = stack.pop()
	Assert(t, bytes.Equal(data, []byte{2}), "Top of stack should be 2")
	data, err = stack.pop()
	Assert(t, bytes.Equal(data, []byte{1}), "Top of stack should be 1")
	data, err = stack.pop()
	Assert(t, bytes.Equal(data, []byte{0}), "Top of stack should be 0")
	data, err = stack.pop()
	Assert(t, err == stackEmptyError, "Empty stack should return error on pop")
}

func TestFullStack(t *testing.T) {
	stack := newStack(100)

	for i := 0; i < 100; i++ {
		err := stack.push([]byte{byte(i)})
		Assert(t, err == nil, "Expected to push until max, stopped at", i+1)
	}

	err := stack.push([]byte{byte(100)})
	Assert(t, err == stackFullError, "Expected stack full error")
	Assert(t, stack.len == 100, "Expected stack to be at max, but only has", stack.len)
}
