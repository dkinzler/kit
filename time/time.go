// Package time provides helper functionality built on top of the standard library time package.
package time

import "time"

// Set to a different function for testing, e.g. one that always returns the same fixed time.
// Defaults to TimeNow.
var TimeFunc func() time.Time = func() time.Time {
	return time.Now()
}

// Returns the time by calling TimeFunc.
func CurrTime() time.Time {
	return TimeFunc()
}

// Returns the time as Unix time in nanoseconds by calling TimeFunc.
func CurrTimeUnixNano() int64 {
	return CurrTime().UnixNano()
}

// Adds a duration to the given unix time in nanoseconds.
func AddDurationToUnixNano(t int64, add time.Duration) int64 {
	return time.Unix(0, t).Add(add).UnixNano()
}
