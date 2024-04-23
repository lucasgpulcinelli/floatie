package raft

import (
	"math/rand"
	"time"
)

func randDuration(a, b time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(b)-int64(a)) + int64(a))
}
