package uuid

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidUUIDsCreated(t *testing.T) {
	a := assert.New(t)

	//check that UUID has correct length
	//a version 4 UUID should be 128 bit = 16 bytes = 32 characters in hex
	//the UUID as a string is split into 5 parts separated by 4 "-" characters
	//which yields a total of 36 characters
	uuid, err := NewUUID()
	a.Nil(err)
	a.Len(uuid, 36)

	//two calls should return different UUIDs
	//... unless we are astronomically unlucky
	id1, err := NewUUID()
	a.Nil(err)
	id2, err := NewUUID()
	a.Nil(err)
	a.NotEqual(id1, id2)
}

func TestPrefixAddedCorrectly(t *testing.T) {
	a := assert.New(t)

	uuid, err := NewUUIDWithPrefix("test")
	a.Nil(err)
	//the prefix "test-" has length 5
	a.Len(uuid, 36+5)
	a.True(strings.HasPrefix(uuid, "test-"))
}
