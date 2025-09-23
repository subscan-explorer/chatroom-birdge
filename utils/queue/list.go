package queue

import (
	"bytes"
	"container/list"
	"fmt"
)

type List[T any] struct {
	inner *list.List
	max   int
}

func NewMessageList[T any](m int) *List[T] {
	l := new(List[T])
	l.inner = list.New()
	l.max = m
	return l
}

func (l *List[T]) Push(data T) {
	if l.inner.Len() >= l.max {
		l.inner.Remove(l.inner.Front())
	}
	l.inner.PushBack(data)
}

func (l *List[T]) SearchFunc(ok func(v T) bool) *T {
	for e := l.inner.Back(); e != nil; e = e.Prev() {
		v := e.Value.(T)
		if ok(v) {
			return &v
		}
	}
	return nil
}

func (l *List[T]) DeleteFunc(del func(v T) bool) *T {
	for e := l.inner.Back(); e != nil; e = e.Prev() {
		v := e.Value.(T)
		if del(v) {
			l.inner.Remove(e)
			return &v
		}
	}
	return nil
}

func (l *List[T]) String() string {
	var result bytes.Buffer
	result.WriteByte('[')
	for e := l.inner.Front(); e != nil; {
		result.WriteString(fmt.Sprintf("%v", e.Value))
		e = e.Next()
		if e != nil {
			result.WriteByte(' ')
		}
	}
	result.WriteByte(']')
	return result.String()
}
