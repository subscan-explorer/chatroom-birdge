package queue

import (
	"bytes"
	"fmt"
)

const MaxInt = int(^uint(0) >> 1)
const MinInt = -MaxInt

type Queue[T any] struct {
	array    []T
	front    int
	rear     int
	capacity int
	size     int
}

func NewQueue[T any](capacity int) *Queue[T] {
	return new(Queue[T]).init(capacity)
}

func (q *Queue[T]) init(capacity int) *Queue[T] {
	q.array = make([]T, capacity)
	q.front, q.rear, q.size, q.capacity = -1, -1, 0, capacity
	return q
}

func (q *Queue[T]) length() int {
	return q.size
}

func (q *Queue[T]) isEmpty() bool {
	return q.size == 0
}

func (q *Queue[T]) isFull() bool {
	return q.size == q.capacity
}

func (q *Queue[T]) String() string {
	var result bytes.Buffer
	result.WriteByte('[')
	j := q.front
	for i := 0; i < q.size; i++ {
		result.WriteString(fmt.Sprintf("%v", q.array[j]))
		if i < q.size-1 {
			result.WriteByte(' ')
		}
		j = (j + 1) % q.capacity
	}
	result.WriteByte(']')
	return result.String()
}

func (q *Queue[T]) Front() T {
	return q.array[q.front]
}

func (q *Queue[T]) Back() T {
	return q.array[q.rear]
}

func (q *Queue[T]) MustEnqueue(v T) {
	if q.isFull() {
		q.Dequeue()
	}
	q.Enqueue(v)
}

func (q *Queue[T]) Enqueue(v T) {
	if q.isFull() {
		return
	}

	q.rear = (q.rear + 1) % q.capacity
	q.array[q.rear] = v
	if q.front == -1 {
		q.front = q.rear
	}
	q.size++
}

func (q *Queue[T]) Dequeue() *T {
	if q.isEmpty() {
		return nil
	}

	data := q.array[q.front]
	if q.front == q.rear {
		q.front = -1
		q.rear = -1
		q.size = 0
	} else {
		q.front = (q.front + 1) % q.capacity
		q.size--
	}
	return &data
}

func (q *Queue[T]) Search(ok func(v T) bool) *T {
	if q.isEmpty() {
		return nil
	}
	j := q.front
	for i := 0; i < q.size; i++ {
		if ok(q.array[j]) {
			return &q.array[j]
		}
		j = (j + 1) % q.capacity
	}
	return nil
}
