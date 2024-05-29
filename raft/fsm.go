package raft

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

// State defines the moment in the raft FSM that the instance is in.
type State byte

// The possible states for the raft FSM.
const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Leader:
		return "Leader"
	case Candidate:
		return "Candidate"
	default:
		return "Invalid"
	}
}

func (raft *Raft) setState(state State) {
	slog.Debug("setting state", "from", raft.state, "to", state)

	if state != Leader && state != Candidate && state != Follower {
		panic("tried setting state to invalid")
	}

	switch raft.state {
	case Leader:
		raft.stopLeader()
	case Candidate:
		raft.abortElection()
	case Follower:
		break
	default:
		panic("invalid value in raft state")
	}

	raft.state = state

	switch raft.state {
	case Leader:
		raft.startLeader()
	case Candidate:
		raft.triggerElection()
	case Follower:
	default:
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

func (raft *Raft) abortElection() {
	if raft.electionCancel == nil {
		return
	}

	slog.Debug("aborting election")
	raft.electionCancel()
	raft.electionCancel = nil
}

func (raft *Raft) heartbeatManager(ctx context.Context, id int32, peer rpcs.RaftClient) {
	t := randDuration(raft.timings.HearbeatLow, raft.timings.HearbeatHigh)
	ticker := time.NewTicker(t)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
		}
		raft.mut.Lock()

		prevLogTerm := int32(-1)
		if raft.lastAppliedIndex != -1 {
			prevLogTerm = raft.logs[raft.lastAppliedIndex].Term
		}

		data := &rpcs.AppendEntryData{
			Term:         raft.currentTerm,
			LeaderID:     raft.id,
			PrevLogIndex: raft.lastAppliedIndex,
			PrevLogTerm:  int32(prevLogTerm),
			LeaderCommit: raft.commitIndex,
			Entries:      []*rpcs.Log{},
		}
		raft.mut.Unlock()
		res, err := peer.AppendEntries(ctx, data)
		if err != nil {
			slog.Error("error during AppendEntries", "error", err)
			continue
		}

		if !res.Success {
			ticker.Stop()
			raft.mut.Lock()
			raft.setState(Follower)
			raft.mut.Unlock()
			return
		}
	}
}

func (raft *Raft) startLeader() {
	slog.Info("starting becoming leader")

	ctx, cancel := context.WithCancel(context.Background())
	raft.lp = &LeaderProperties{cancelHeartbeats: cancel}

	// start heartbeat goroutines
	for id, peer := range raft.peers {
		go raft.heartbeatManager(ctx, id, peer)
	}
}

func (raft *Raft) triggerElection() {
	slog.Info("starting election")

	raft.currentTerm++
	newTerm := raft.currentTerm

	ctx, cancel := context.WithCancel(context.Background())
	raft.electionCancel = cancel
	raft.lastVoted = raft.id

	lastLogTerm := int32(0)
	if raft.commitIndex != -1 {
		lastLogTerm = raft.logs[raft.commitIndex].Term
	}

	voteRequest := &rpcs.RequestVoteData{
		Term:         raft.currentTerm,
		CandidateID:  raft.id,
		LastLogIndex: raft.commitIndex,
		LastLogTerm:  lastLogTerm,
	}

	favorable := atomic.Int32{}
	unfavorable := atomic.Int32{}

	wg := sync.WaitGroup{}
	wg.Add(len(raft.peers))

	raft.mut.Unlock()

	for id, peer := range raft.peers {
		go func(id int32, peer rpcs.RaftClient) {
			result, err := peer.RequestVote(ctx, voteRequest)

			// if the election has already ended or has been cancelled
			if ctx.Err() == context.Canceled {
				wg.Done()
				return
			}
			if err != nil {
				slog.Warn(
					"Error during vote request",
					"id", id,
					"error", err,
				)
				wg.Done()
				return
			}

			if result.GetSuccess() {
				favorable.Add(1)
			} else {
				unfavorable.Add(1)
			}

			// if a result is already defined
			if int(favorable.Load()) >= (len(raft.peers)+1)/2 ||
				int(unfavorable.Load()) >= (len(raft.peers)+1)/2 {
				// make all other goroutines stop waiting for a result from the peers
				cancel()
			}

			wg.Done()
		}(id, peer)
	}

	wg.Wait()

	raft.mut.Lock()

	// if something happened (such as an election cancel or other leader with
	// higher term sent a message), our election does not matter anymore
	if raft.currentTerm != newTerm {
		slog.Info("election result aborted", "term", newTerm)
		raft.electionCancel = nil
		return
	}

	cancel()
	raft.electionCancel = nil

	// if elected
	if int(favorable.Load()) >= (len(raft.peers)+1)/2 {
		slog.Info("elected as leader")
		raft.setState(Leader)
	}
}
