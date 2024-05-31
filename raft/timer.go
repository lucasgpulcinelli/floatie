package raft

import (
	"log/slog"
	"math/rand"
	"time"
)

func randDuration(a, b time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(b)-int64(a)) + int64(a))
}

// StartTimerLoop starts the timer goroutine to trigger elections if the leader
// fails to send heartbeats.
func (raft *Raft) StartTimerLoop(timings *RaftTimings) {
	raft.mut.Lock()
	defer raft.mut.Unlock()

	raft.timings = timings

	raft.timerChan = make(chan struct{})
	raft.timerStop = make(chan struct{}, 0)

	go timerLoop(raft.timings, raft.timerChan, raft.timerStop, func() {
		go func() {
			raft.mut.Lock()
			slog.Debug("timeout occurred")
			if raft.state != Leader {
				raft.setState(Candidate)
			}
			raft.mut.Unlock()
		}()
	})
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
