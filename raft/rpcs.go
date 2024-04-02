package raft

import (
	"context"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func (raft *Raft) matchLog(logIndex, logTerm int32) bool {
	if len(raft.logs) < int(logIndex) {
		return false
	}

	if raft.logs[logIndex].Term != logTerm {
		raft.logs = raft.logs[:logIndex]
		return false
	}

	return true
}

func (raft *Raft) AppendEntries(ctx context.Context, data *rpcs.AppendEntryData) (*rpcs.RaftResult, error) {
	fail := &rpcs.RaftResult{Success: false, Term: raft.currentTerm}

	if data.Term < raft.currentTerm {
		return fail, nil
	}

	if !raft.matchLog(data.PrevLogIndex, data.PrevLogTerm) {
		return fail, nil
	}

	for _, e := range data.Entries {
		raft.logs = append(raft.logs, e)
	}

	if data.LeaderCommit > raft.commitIndex {
		raft.commitIndex = min(data.LeaderCommit, int32(len(raft.logs)))
	}

	return &rpcs.RaftResult{Success: true, Term: raft.currentTerm}, nil
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
