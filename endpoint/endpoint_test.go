package endpoint

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseImplementsResponderCorrectly(t *testing.T) {
	a := assert.New(t)
	var x Responder = Response{
		R:   "sometestvalue",
		Err: errors.New("someerror"),
	}
	a.Equal("sometestvalue", x.Response())
	a.Equal(errors.New("someerror"), x.Error())
}
