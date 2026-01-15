package util

import "fmt"

// CircularQueue implements a circular queue data structure. It allows efficient
// addition and removal of elements in a FIFO manner while utilizing a fixed-size buffer. In addition,
// it supports pushing elements back into the queue for re-processing.
type CircularQueue[T any] struct {
	data     []T
	capacity int
	front    int
	rear     int
	size     int
}

// create an instance of the circular queue:
func NewCircularQueue[T any](capacity int) *CircularQueue[T] {
	return &CircularQueue[T]{
		data:     make([]T, capacity),
		capacity: capacity,
		front:    0,
		rear:     -1,
		size:     0,
	}
}

// IsFull checks if the circular queue is full.
func (c *CircularQueue[T]) IsFull() bool {
	return c.size == c.capacity
}

// IsEmpty checks if the circular queue is empty,
func (c *CircularQueue[T]) IsEmpty() bool {
	return c.size == 0
}

// Length returns the current number of elements in the circular queue.
func (c *CircularQueue[T]) Length() int {
	return c.size
}

// Enqueue adds an element to the rear of the circular queue.
func (c *CircularQueue[T]) Enqueue(data T) error {
	if c.IsFull() {
		return fmt.Errorf("Circular Queue is Full!")
	}

	c.rear = (c.rear + 1) % c.capacity // to effectively wrap around (ensuring the queue uses all available slots)
	c.data[c.rear] = data              // insert the data into the last position
	c.size++                           // increment size of queue

	return nil
}

// Dequeue removes and returns an element from the front of the circular queue.
func (c *CircularQueue[T]) Dequeue() (T, error) {
	if c.IsEmpty() {
		var zero T
		return zero, fmt.Errorf("Circular Queue is empty!")
	}

	data := c.data[c.front]              // retrieves the element in front of the queue
	c.front = (c.front + 1) % c.capacity // updates the front index to move to the next position
	c.size--                             //decrement queue size

	return data, nil
}

// Push allows adding a data object at the end of the queue that the function [Dequeue] will
// emit objects from. This enables data tha hase been de-queued to be pushed back into the queue for
// another or the same reader to consume data. This is commonly used in text / code parsing or search
// e.g. with a backtracking algorithm.
func (c *CircularQueue[T]) Push(data T) error {
	if c.IsFull() {
		return fmt.Errorf("Circular Queue is Full!")
	}

	c.front = (c.front + 1) % c.capacity // to effectively wrap around (ensuring the queue uses all available slots)
	c.data[c.front] = data               // insert the data into the last position
	c.size++                             // increment size of queue

	return nil

}

// Peek returns the element at the front of the circular queue without removing it.
func (c *CircularQueue[T]) Peek() (T, error) {
	if c.IsEmpty() {
		var zero T
		return zero, fmt.Errorf("Circular Queue is empty!")
	}

	value := c.data[c.front]
	return value, nil
}
