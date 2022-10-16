// Package uuid implements convenience functions on top of github.com/google/uuid to create UUIDs.
package uuid

import (
	"github.com/google/uuid"
)

// Returns a new random UUID (version 4)
func NewUUID() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return uuid.String(), nil
}

// Returns a new random UUID prefixed with the given string.
// The prefix and UUID are separated by a "-" character.
func NewUUIDWithPrefix(prefix string) (string, error) {
	u, err := NewUUID()
	if err != nil {
		return "", err
	}
	return prefix + "-" + u, nil
}
