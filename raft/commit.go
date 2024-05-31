package raft

import (
	"log/slog"
	"slices"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func (raft *Raft) GetLogs() []string {
	raft.mut.Lock()
	defer raft.mut.Unlock()

	ls := []string{}
	for i, v := range raft.logs {
		if i > int(raft.lastAppliedIndex) {
			break
		}
		ls = append(ls, v.Data)
	}
	return ls
}

func (raft *Raft) dropOldLogs(prevLogIndex int32) {
	raft.logs = raft.logs[:int(prevLogIndex)+1]

	raft.requestCond.Broadcast()
}

func (raft *Raft) applyCommited() {
	if raft.commitGoroutineRunning || raft.commitIndex == -1 {
		return
	}

	raft.commitGoroutineRunning = true
	go func() {
		raft.mut.Lock()
		defer raft.mut.Unlock()

		t := time.Millisecond * 10

		for raft.lastAppliedIndex < raft.commitIndex {
			toApply := raft.logs[raft.lastAppliedIndex+1].Data
			raft.mut.Unlock()

			err := raft.applyLog(toApply)
			if err != nil {
				slog.Error("applyLog failed", "error", err)
				time.Sleep(t)
				t *= 2

				raft.mut.Lock()
				continue
			}

			t = time.Millisecond * 10

			raft.mut.Lock()
			raft.lastAppliedIndex++
			raft.requestCond.Broadcast()
		}

		raft.commitGoroutineRunning = false
	}()
}

func (raft *Raft) refreshCommitIndex() {

	// calculate the median
	matchIndexes := []int32{}
	for _, mi := range raft.lp.matchIndex {
		matchIndexes = append(matchIndexes, mi)
	}

	slices.Sort[[]int32](matchIndexes)

	N := matchIndexes[(len(matchIndexes)+1)/2-1]

	// and see if it is suitable as commitIndex
	if N <= raft.commitIndex || raft.logs[N].Term != raft.currentTerm {
		return
	}

	raft.commitIndex = N
	raft.applyCommited()
}

func (raft *Raft) SendLog(logData string) bool {
	raft.mut.Lock()
	defer raft.mut.Unlock()

	if raft.state != Leader {
		return false
	}

	index := int32(len(raft.logs))
	term := raft.currentTerm
	raft.logs = append(raft.logs, &rpcs.Log{Term: raft.currentTerm, Data: logData})

	for {
		if raft.lastAppliedIndex >= index {
			break
		}
		if len(raft.logs) <= int(index) || raft.logs[index].Term != term {
			return false
		}
		raft.requestCond.Wait()
	}

	return true
}
