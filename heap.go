package main

import (
	goheap "container/heap"
	"iter"
	"slices"
)

type Heap[T any] struct {
	heap heap[T]
}

func MakeHeap[T any](less func(T, T) bool) Heap[T] {
	return Heap[T]{
		heap: heap[T]{
			less: less,
		},
	}
}

func (h *Heap[T]) Len() int {
	return len(h.heap.values)
}

func (h *Heap[T]) Pop() T {
	return goheap.Pop(&h.heap).(T)
}

func (h *Heap[T]) Push(value T) {
	goheap.Push(&h.heap, value)
}

func (h *Heap[T]) IsEmpty() bool {
	return len(h.heap.values) == 0
}

func (h *Heap[T]) Values() iter.Seq[T] {
	return slices.Values(h.heap.values)
}

type heap[T any] struct {
	values []T
	less   func(T, T) bool
}

func (h *heap[T]) Len() int {
	return len(h.values)
}

func (h *heap[T]) Less(i, j int) bool {
	return h.less(h.values[i], h.values[j])
}

func (h *heap[T]) Swap(i, j int) {
	h.values[i], h.values[j] = h.values[j], h.values[i]
}

func (h *heap[T]) Push(x any) {
	h.values = append(h.values, x.(T))
}

func (h *heap[T]) Pop() any {
	n := len(h.values)
	value := h.values[n-1]
	h.values = h.values[0 : n-1]
	return value
}
