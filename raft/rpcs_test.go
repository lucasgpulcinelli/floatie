package raft_test

import (
	"context"
	"testing"

	"github.com/lucasgpulcinelli/floatie/raft"
	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func TestSingleRequestVote(t *testing.T) {
	r, err := raft.New(0, map[int32]*rpcs.RaftClient{})
	if err != nil {
		t.Logf("error during raft creation: %v", err)
		t.FailNow()
	}

	resp, err := r.RequestVote(context.Background(), &rpcs.RequestVoteData{
		Term:         0,
		CandidateID:  1,
		LastLogIndex: -1,
		LastLogTerm:  -1,
	})
	if err != nil {
		t.Logf("error during requestVote: %v", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("vote did not succeed")
		t.FailNow()
	}

	if resp.Term != 0 {
		t.Logf("term is not correct")
		t.FailNow()
	}

	r.Stop()
}

func TestSingleHeartbeat(t *testing.T) {
	r, err := raft.New(0, map[int32]*rpcs.RaftClient{})
	if err != nil {
		t.Logf("error during raft creation: %v", err)
		t.FailNow()
	}

	resp, err := r.AppendEntries(context.Background(), &rpcs.AppendEntryData{
		Term:         0,
		LeaderID:     1,
		PrevLogIndex: -1,
		PrevLogTerm:  -1,
		LeaderCommit: -1,
		Entries:      []*rpcs.Log{},
	})
	if err != nil {
		t.Logf("error during requestVote: %v", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("vote did not succeed")
		t.FailNow()
	}

	if resp.Term != 0 {
		t.Logf("term is not correct")
		t.FailNow()
	}

	r.Stop()
}
