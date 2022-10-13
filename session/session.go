package session

import (
	"context"
	"sync"
)

//TODO fix all these and comment some stuff, although we probably wont even need this package

//TODO need to be careful that that a value is not modified while we use it
//e.g. if the value is a slice we can get it, then the lock is released and another goroutine
//can modifiy the slice while we e.g. iterate over it

type Session struct {
	mutex  sync.Mutex
	values map[interface{}]interface{}
}

func NewSession() *Session {
	return &Session{
		mutex:  sync.Mutex{},
		values: make(map[interface{}]interface{}),
	}
}

func (s *Session) GetValue(key interface{}) (interface{}, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v, ok := s.values[key]
	return v, ok
}

func (s *Session) UpdateValue(key interface{}, updateFn func(interface{}, bool) (interface{}, bool)) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	oldValue, ok := s.values[key]
	newValue, ok := updateFn(oldValue, ok)
	if !ok {
		return false
	}
	if newValue != nil {
		s.values[key] = newValue
	}
	return true
}

func (s *Session) DeleteValue(key interface{}) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.values, key)
	return true
}

type contextKey string

const sessionContextKey contextKey = "session"

func ContextWithSession(ctx context.Context) context.Context {
	session := NewSession()
	return context.WithValue(ctx, sessionContextKey, session)
}

func SessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	if !ok {
		return nil, false
	}
	return session, true
}

func ValueFromContext[V any](ctx context.Context, key interface{}) (V, bool) {
	var result V
	session, ok := ctx.Value(sessionContextKey).(*Session)
	if !ok {
		return result, false
	}
	v, ok := session.GetValue(key)
	if !ok {
		return result, false
	}
	result, ok = v.(V)
	if !ok {
		return result, false

	}
	return result, true
}

// TODO what if there is no old value maybe we should pass old value and bool to updateFn
// can we make this easier?
func UpdateContextValue[V any](ctx context.Context, key interface{}, updateFn func(oldValue V, hasOldValue bool) V) bool {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	if !ok {
		return false
	}
	return session.UpdateValue(key, func(oldValue interface{}, hasOldValue bool) (interface{}, bool) {
		var v V
		var ok bool
		if hasOldValue {
			v, ok = oldValue.(V)
			if !ok {
				return nil, false
			}
		}
		newValue := updateFn(v, hasOldValue)
		return newValue, true
	})
}
