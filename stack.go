package main

import (
	"errors"
	"sync"
)

var stackEmptyError = errors.New("Stack is empty")
var stackFullError = errors.New("Stack is full")

type payload struct {
	data []byte
	next *payload
}

type stack struct {
	mu  *sync.Mutex
	top *payload
	max int
	len int
}

func newStack(maxLength int) *stack {
	return &stack{mu: &sync.Mutex{}, max: maxLength}
}

func (s *stack) push(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.len == s.max {
		return stackFullError
	}
	payload := &payload{data: data, next: s.top}
	s.top = payload
	s.len++

	return nil
}

func (s *stack) pop() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.len == 0 {
		return nil, stackEmptyError
	}

	top := s.top
	s.top = top.next
	s.len--

	return top.data, nil
}
