package raft

import (
	"math/rand"
	"time"
)

func randDuration(a, b time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(b)-int64(a)) + int64(a))
}

func timerLoop(timeout time.Duration, timerChan chan time.Duration, timerStop chan struct{}, callback func()) {
	ticker := time.NewTicker(timeout)
	started := time.Now()
	for {
		select {
		case <-ticker.C:
			callback()
		case d := <-timerChan:
			ticker.Reset(max(time.Now().Sub(started)+d, timeout))
		case <-timerStop:
			return
		}
		started = time.Now()
	}
}
