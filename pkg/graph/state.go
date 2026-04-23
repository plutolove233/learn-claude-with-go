package graph

import (
	"reflect"
	"sync"
)

// State is the shared mutable state passed between graph nodes.
type State struct {
	mu     sync.RWMutex
	values map[string]any
}

// NewState creates a new State from an optional initial snapshot.
func NewState(initial map[string]any) *State {
	state := &State{values: make(map[string]any)}
	state.Merge(initial)
	return state
}

// Get returns a value from state by key.
func (s *State) Get(key string) (any, bool) {
	if s == nil {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.values == nil {
		return nil, false
	}
	value, ok := s.values[key]
	return value, ok
}

// Set stores a value in state.
func (s *State) Set(key string, value any) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]any)
	}
	s.values[key] = deepCopyValue(value)
}

// Delete removes a value from state.
func (s *State) Delete(key string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		return
	}
	delete(s.values, key)
}

// Has reports whether the key exists in state.
func (s *State) Has(key string) bool {
	_, ok := s.Get(key)
	return ok
}

// Merge applies an update into state.
func (s *State) Merge(update map[string]any) {
	if s == nil || len(update) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]any, len(update))
	}
	for key, value := range update {
		s.values[key] = deepCopyValue(value)
	}
}

// Snapshot returns a deep copy of the current state map.
func (s *State) Snapshot() map[string]any {
	if s == nil {
		return map[string]any{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.values) == 0 {
		return map[string]any{}
	}

	snapshot := make(map[string]any, len(s.values))
	for key, value := range s.values {
		snapshot[key] = deepCopyValue(value)
	}
	return snapshot
}

// Value returns a typed value from state.
func Value[T any](s *State, key string) (T, bool) {
	var zero T

	raw, ok := s.Get(key)
	if !ok {
		return zero, false
	}

	value, ok := raw.(T)
	if !ok {
		return zero, false
	}
	return value, true
}

func deepCopyValue(value any) any {
	if value == nil {
		return nil
	}
	return deepCopy(reflect.ValueOf(value)).Interface()
}

func deepCopy(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		copied := reflect.New(v.Type().Elem())
		copied.Elem().Set(deepCopy(v.Elem()))
		return copied
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		copied := deepCopy(v.Elem())
		wrapped := reflect.New(v.Type()).Elem()
		wrapped.Set(copied)
		return wrapped
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		copied := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			copied.SetMapIndex(iter.Key(), deepCopy(iter.Value()))
		}
		return copied
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		copied := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			copied.Index(i).Set(deepCopy(v.Index(i)))
		}
		return copied
	case reflect.Array:
		copied := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			copied.Index(i).Set(deepCopy(v.Index(i)))
		}
		return copied
	case reflect.Struct:
		copied := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := copied.Field(i)
			if field.CanSet() {
				field.Set(deepCopy(v.Field(i)))
				continue
			}
			field.Set(v.Field(i))
		}
		return copied
	default:
		return v
	}
}
