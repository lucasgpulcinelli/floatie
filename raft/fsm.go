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

	switch raft.state {
	case Leader:
		raft.stopLeader()
	case Candidate:
		raft.abortElection()
	case Follower:
	}

	raft.state = state
}

func (raft *Raft) stopLeader() {
	slog.Debug("stop being leader")
}

func (raft *Raft) abortElection() {
	slog.Debug("aborting election")
}

func (raft *Raft) triggerElection() {
	slog.Debug("starting election")

	raft.mut.Lock()
	defer raft.mut.Unlock()

	if raft.state == Leader {
		return
	}

	raft.setState(Candidate)
}
