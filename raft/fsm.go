package raft

import (
	"log/slog"
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

func (raft *Raft) setTerm(newTerm int32) {
	if raft.currentTerm > newTerm {
		slog.Warn("tried to set term to smaller value")
		return
	}

	if raft.currentTerm != newTerm {
		raft.lastVoted = -1
	}

	raft.currentTerm = newTerm
}
