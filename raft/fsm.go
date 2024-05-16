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
	slog.Debug("starting becoming leader")
}

func (raft *Raft) triggerElection() {
	slog.Debug("starting election")

	ctx, cancel := context.WithCancel(context.Background())
	raft.electionCancel = cancel

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
				if int(favorable.Add(1))+1 >= (len(raft.peers)+1)/2 {
					cancel()
				}
			} else {
				if int(unfavorable.Add(1))+1 >= (len(raft.peers)+1)/2 {
					cancel()
				}
			}

			wg.Done()
		}(id, peer)
	}

	wg.Wait()

	raft.mut.Lock()

	cancel()
	raft.electionCancel = nil

	// if elected
	if int(favorable.Load()) >= (len(raft.peers)+1)/2 {
		raft.setState(Leader)
	}
}
