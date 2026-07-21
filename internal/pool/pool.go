// Package pool предоставляет generic-контейнер поверх sync.Pool
// для объектов с методом Reset().
package pool

import "sync"

// Resetter — ограничение для типов, у которых есть сброс состояния.
type Resetter interface {
	Reset()
}

// Pool — контейнер объектов одного типа T с методом Reset().
// Внутри использует sync.Pool для повторного использования «тяжёлых» объектов.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт пул. newFn вызывается, когда в пуле нет свободных объектов.
// Если newFn == nil, Get возвращает zero-value типа T.
func New[T Resetter](newFn func() T) *Pool[T] {
	p := &Pool[T]{}
	if newFn != nil {
		p.pool.New = func() any {
			return newFn()
		}
		return p
	}
	p.pool.New = func() any {
		var zero T
		return zero
	}
	return p
}

// Get возвращает объект из пула (или новый, если пул пуст).
func (p *Pool[T]) Get() T {
	v := p.pool.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

// Put сбрасывает состояние объекта и возвращает его в пул.
func (p *Pool[T]) Put(v T) {
	if any(v) == nil {
		return
	}
	v.Reset()
	p.pool.Put(v)
}
