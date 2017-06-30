package util

import (
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

var unixStart = time.Date(
	1970, 1, 1, 0, 0, 0, 0, time.UTC, // 1970-01-01T00:00:00Z
)

// Timestamp2Time convert *timestamp.Timestamp to time.Time
func Timestamp2Time(t *timestamp.Timestamp) time.Time {
	if t.Seconds == 0 && t.Nanos == 0 {
		return time.Time{}
	}
	return time.Unix(t.Seconds, int64(t.Nanos))
}

// Time2Timestamp convert time.Time to *timestamp.Timestamp
func Time2Timestamp(t time.Time) *timestamp.Timestamp {
	if t.IsZero() {
		return &timestamp.Timestamp{}
	}
	sub := t.Sub(unixStart)
	return &timestamp.Timestamp{Seconds: int64(sub.Seconds()), Nanos: int32(t.Nanosecond())}
}
