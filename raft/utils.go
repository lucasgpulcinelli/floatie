package raft

import (
	"math/rand"
	"time"
)

func randDuration(a, b time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(b)-int64(a)) + int64(a))
}

func timerLoop(t *RaftTimings, timerChan chan struct{}, timerStop chan struct{}, callback func()) {
	ticker := time.NewTicker(randDuration(t.TimeoutLow, t.TimeoutHigh))
	for {
		select {
		case <-ticker.C:
			callback()
		case <-timerChan:
			d := randDuration(t.TimeoutLow, t.TimeoutHigh)
			ticker.Reset(d)
		case <-timerStop:
			return
		}
	}
}
