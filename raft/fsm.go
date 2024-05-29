package raft

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

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
	slog.Debug("stopping being leader")

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

func (raft *Raft) startLeader() {
	slog.Info("starting becoming leader")
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
			if err == context.Canceled {
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
