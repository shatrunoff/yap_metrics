package pool

import (
	"sync"
)

// Pool представляет собой пул объектов с типом T.
type Pool[T any] struct {
	pool     sync.Pool
	mu       sync.Mutex
	allItems []T
	newFunc  func() T
}

// New создает и возвращает новый пул объектов типа T.
func New[T any](newFunc func() T) *Pool[T] {
	p := &Pool[T]{
		newFunc:  newFunc,
		allItems: make([]T, 0),
	}

	p.pool.New = func() any {
		item := newFunc()
		p.mu.Lock()
		p.allItems = append(p.allItems, item)
		p.mu.Unlock()
		return item
	}

	return p
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	item := p.pool.Get().(T)

	if resetter, ok := any(&item).(interface{ Reset() }); ok {
		resetter.Reset()
	}

	return item
}

// Put помещает объект обратно в пул.
func (p *Pool[T]) Put(item T) {
	// Сбрасываем объект перед возвращением в пул
	if resetter, ok := any(&item).(interface{ Reset() }); ok {
		resetter.Reset()
	}
	p.pool.Put(item)
}

// GetAllItems возвращает слайс всех объектов, когда-либо созданных пулом.
func (p *Pool[T]) GetAllItems() []T {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]T, len(p.allItems))
	copy(result, p.allItems)
	return result
}

// Clear очищает внутренние структуры пула.
func (p *Pool[T]) Clear() {
	p.mu.Lock()
	p.allItems = make([]T, 0)
	p.mu.Unlock()

	// Создаем новый sync.Pool
	p.pool = sync.Pool{
		New: func() any {
			item := p.newFunc()
			p.mu.Lock()
			p.allItems = append(p.allItems, item)
			p.mu.Unlock()
			return item
		},
	}
}
