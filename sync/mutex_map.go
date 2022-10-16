// Package sync provides advanced synchronization tools.
package sync

import (
	"sync"
)

// MutexMap allows to obtain a mutex lock that is scoped to a given string key.
// I.e. only one goroutine at a time can hold the lock for the same key, while locks for different keys are independent.
//
// An example use case might be a HTTP service that processes requests concurrently, but we want to make sure that at most one request for the same user is handled at a time.
// To achieve this, one can share a single MutexMap across all http handler goroutines and the http handler obtains the lock for the user (by e.g. using a userId as the key for the MutexMap)
// before doing any work.
//
// Example usage:
//
//	mm := NewMutexMap()
//	l := mm.Lock("exampleKey")
//	// prefer to call Unlock() with defer right after locking, to make sure the lock gets unlocked eventually
//	defer l.Unlock()
//
// Careful: nested locks, i.e. trying to obtain a lock for key x while already holding the lock for key y can lead to deadlocks.
//
// Implementation copied from answer https://stackoverflow.com/a/62562831 to https://stackoverflow.com/questions/40931373/how-to-gc-a-map-of-mutexes-in-go .
type MutexMap struct {
	lock      sync.Mutex
	keyToLock map[string]*keyMutex
}

func NewMutexMap() *MutexMap {
	return &MutexMap{keyToLock: make(map[string]*keyMutex)}
}

type keyMutex struct {
	key string
	// the MutexMap this mutex belongs to
	mm *MutexMap
	// number of goroutines having/waiting for this lock
	// when count reaches 0 we can delete this keyMutex from MutexMap to avoid MutexMap growing endlessly
	count int
	inner sync.Mutex
}

func (km *keyMutex) Unlock() {
	km.mm.unlock(km.key)
	km.inner.Unlock()
}

type Unlocker interface {
	Unlock()
}

// Obtain the lock for the given key.
// If the lock is already held by another goroutine, this function blocks until the lock is released.
// The calling goroutine can use the returned Unlocker to unlock the given key when done.
func (mm *MutexMap) Lock(key string) Unlocker {
	// obtain the global lock of this MutexMap, only one goroutine should read/modify the map that contains the mutexes
	mm.lock.Lock()
	km, ok := mm.keyToLock[key]
	if !ok {
		km = &keyMutex{key: key, mm: mm, count: 0}
		mm.keyToLock[key] = km
	}
	km.count++
	// we need to unlock the global lock of this MutexMap before we can attempt to obtain the lock for the key
	// otherwise another goroutine could not try to get the lock for another key
	// that's why we can't unlock the global lock using defer, although this should be safe, because the code between the lock and unlock should not produce any errors
	mm.lock.Unlock()
	km.inner.Lock()
	return km
}

func (mm *MutexMap) unlock(key string) {
	mm.lock.Lock()
	defer mm.lock.Unlock()
	km, ok := mm.keyToLock[key]
	if !ok {
		// this shouldn't happen, since this function is only called from within keyMutex.Unlock()
		panic("no lock for the given key")
	}
	km.count--
	if km.count == 0 {
		delete(mm.keyToLock, key)
	}
}
