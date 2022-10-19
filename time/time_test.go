package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddDurationToUnixNano(t *testing.T) {
	a := assert.New(t)

	timeNow := CurrTimeUnixNano()
	x := AddDurationToUnixNano(timeNow, 42*time.Nanosecond)
	a.Equal(timeNow+42, x)

	y := AddDurationToUnixNano(timeNow, 42*time.Second)
	a.Equal(timeNow+42000000000, y)
}

func TestCurrTimeReturnsTimeFromTimeFunc(t *testing.T) {
	a := assert.New(t)

	ft := time.Date(2022, 4, 4, 13, 37, 0, 0, time.UTC)
	ftu := ft.UnixNano()

	TimeFunc = func() time.Time {
		return ft
	}

	a.True(ft.Equal(CurrTime()))
	a.Equal(ftu, CurrTimeUnixNano())
}
