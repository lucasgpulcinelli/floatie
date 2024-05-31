package raft

import (
	"context"
	"log/slog"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func (raft *Raft) dropOldLogs(prevLogIndex int32) {
	raft.logs = raft.logs[:int(prevLogIndex)+1]

	raft.requestCond.Broadcast()
}

func (raft *Raft) AppendEntries(ctx context.Context, data *rpcs.AppendEntryData) (*rpcs.RaftResult, error) {
	slog.Debug("received AppendEntries", "data", data)

	raft.mut.Lock()
	defer raft.mut.Unlock()

	fail := &rpcs.RaftResult{Success: false, Term: raft.currentTerm}

	if data.Term < raft.currentTerm {
		return fail, nil
	}

	if data.PrevLogIndex != -1 && (len(raft.logs) < int(data.PrevLogIndex) ||
		raft.logs[data.PrevLogIndex].Term != data.PrevLogTerm) {

		return fail, nil
	}

	if raft.timerChan != nil {
		raft.timerChan <- struct{}{}
	}

	raft.currentTerm = data.Term

	if raft.state != Follower {
		raft.setState(Follower)
	}

	raft.dropOldLogs(data.PrevLogIndex)

	for _, e := range data.Entries {
		raft.logs = append(raft.logs, e)
	}

	if data.LeaderCommit > raft.commitIndex {
		raft.commitIndex = min(data.LeaderCommit, int32(len(raft.logs)))
	}

	return &rpcs.RaftResult{Success: true, Term: raft.currentTerm}, nil
}

func (raft *Raft) RequestVote(ctx context.Context, data *rpcs.RequestVoteData) (*rpcs.RaftResult, error) {
	slog.Debug("received RequestVote", "data", data)

	raft.mut.Lock()
	defer raft.mut.Unlock()

	voteFalse := &rpcs.RaftResult{Success: false, Term: raft.currentTerm}

	if data.CandidateID == raft.id {
		slog.Warn("received vote with own ID")
		return voteFalse, nil
	}

	if data.Term < raft.currentTerm || data.LastLogIndex < raft.commitIndex {
		return voteFalse, nil
	}

	if data.Term == raft.currentTerm && raft.lastVoted != -1 &&
		raft.lastVoted != data.CandidateID {

		return voteFalse, nil
	}

	if len(raft.logs) != 0 {
		lastLogTerm := raft.logs[len(raft.logs)-1].Term
		if data.LastLogTerm < lastLogTerm {
			return voteFalse, nil
		}
	}

	if raft.timerChan != nil {
		raft.timerChan <- struct{}{}
	}

	raft.lastVoted = data.CandidateID
	raft.currentTerm = data.Term

	if raft.state != Follower {
		raft.setState(Follower)
	}

	return &rpcs.RaftResult{Success: true, Term: raft.currentTerm}, nil
}
