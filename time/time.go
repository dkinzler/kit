package time

import "time"

// can be set to a different function for testing, e.g. one that always returns the same fixed time
// defaults to TimeNow
var TimeFunc func() time.Time = func() time.Time {
	return time.Now()
}

func CurrTime() time.Time {
	return TimeFunc()
}

func CurrTimeUnixNano() int64 {
	return CurrTime().UnixNano()
}

func AddDurationToUnixNano(t int64, add time.Duration) int64 {
	return time.Unix(0, t).Add(add).UnixNano()
}
