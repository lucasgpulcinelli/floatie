package raft

import (
	"context"
	"fmt"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func (raft *Raft) AppendEntries(ctx context.Context, data *rpcs.AppendEntryData) (*rpcs.RaftResult, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (raft *Raft) RequestVote(ctx context.Context, data *rpcs.RequestVoteData) (*rpcs.RaftResult, error) {
	voteFalse := &rpcs.RaftResult{Success: false, Term: raft.currentTerm}

	if raft.lastVoted != -1 && raft.lastVoted != data.CandidateID {
		return voteFalse, nil
	}

	if data.Term < raft.currentTerm || data.LastLogIndex < raft.lastAppliedIndex {
		return voteFalse, nil
	}

	if len(raft.logs) != 0 {
		lastLogTerm := raft.logs[len(raft.logs)-1].Term
		if data.LastLogTerm < lastLogTerm {
			return voteFalse, nil
		}
	}

	raft.lastVoted = data.CandidateID
	raft.currentTerm = data.Term

	// if leader, stop sending heartbeats
	raft.state = Follower

	return &rpcs.RaftResult{Success: true, Term: raft.currentTerm}, nil
}
