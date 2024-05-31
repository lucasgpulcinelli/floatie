package raft

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func (raft *Raft) startLeader() {
	slog.Info("starting becoming leader")

	ctx, cancel := context.WithCancel(context.Background())
	raft.lp = &LeaderProperties{
		cancelHeartbeats: cancel,
		nextIndex:        map[int32]int32{},
		matchIndex:       map[int32]int32{},
	}

	// initalize lp arrays
	for id := range raft.peers {
		raft.lp.nextIndex[id] = raft.lastAppliedIndex + 1
		raft.lp.matchIndex[id] = -1
	}

	// start heartbeat goroutines
	for id, peer := range raft.peers {
		go raft.heartbeatManager(ctx, id, peer, raft.currentTerm)
	}
}

func (raft *Raft) stopLeader() {
	if raft.lp == nil {
		return
	}

	slog.Debug("stopping being leader")

	raft.lp.cancelHeartbeats()
	raft.lp = nil
}

func (raft *Raft) heartbeatManager(ctx context.Context, id int32, peer rpcs.RaftClient, term int32) {
	t := randDuration(raft.timings.HearbeatLow, raft.timings.HearbeatHigh)
	ticker := time.NewTicker(t)
	for {
		raft.mut.Lock()
		if raft.currentTerm != term {
			raft.mut.Unlock()
			return
		}

		prevLogIndex := raft.lp.nextIndex[id] - 1
		toSend := int32(len(raft.logs)) - prevLogIndex - 1

		prevLogTerm := int32(-1)
		if prevLogIndex != -1 {
			prevLogTerm = raft.logs[prevLogIndex].Term
		}

		data := &rpcs.AppendEntryData{
			Term:         raft.currentTerm,
			LeaderID:     raft.id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			LeaderCommit: raft.commitIndex,
			Entries:      raft.logs[prevLogIndex+1:],
		}
		raft.mut.Unlock()

		res, err := peer.AppendEntries(ctx, data)

		// if we get an error, retry
		if err != nil {
			slog.Error("error during AppendEntries", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}

		raft.mut.Lock()
		if raft.currentTerm != term {
			raft.mut.Unlock()
			return
		}

		// if we are not the leader anymore, convert to follower
		if raft.currentTerm != res.Term {
			raft.currentTerm = res.Term
			raft.setState(Follower)
			raft.mut.Unlock()
			return
		}

		// if the prevLog does not match, go back one log
		if !res.Success {
			raft.lp.nextIndex[id]--
		} else {
			raft.lp.matchIndex[id] = prevLogIndex + toSend
			raft.lp.nextIndex[id] += toSend
			raft.refreshCommitIndex()
		}

		raft.mut.Unlock()

		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
		}
	}
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
}
