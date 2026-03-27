package singleton

import "sync"

type Singleton[T any] struct {
	ctor   func() T
	value  T
	once   sync.Once
	mutex  sync.Mutex
	loaded bool
}

func NewSingleton[T any](ctor func() T) *Singleton[T] {
	return &Singleton[T]{ctor: ctor}
}

func (s *Singleton[T]) Get() T {

	// 加锁以便允许做条件判断
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.loaded {
		return s.value
	}

	// 使用 once 确保 ctor 只调用一次（线程安全）
	s.once.Do(func() {
		s.value = s.ctor()
		s.loaded = true
	})

	return s.value
}

func (s *Singleton[T]) IsLoaded() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.loaded
}

// Reset 重置用于测试
func (s *Singleton[T]) Reset() {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.once = sync.Once{}
	var zero T
	s.value = zero
	s.loaded = false
}

// GetSingleton 静态工具函数（不存 ctor）
func GetSingleton[T any](once *sync.Once, value *T, ctor func() T) T {
	once.Do(func() {
		*value = ctor()
	})
	return *value
}
