package raft_test

import (
	"context"
	"testing"

	"github.com/lucasgpulcinelli/floatie/raft"
	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func TestAppendEntries(t *testing.T) {
	r, err := raft.New(0, map[int32]*rpcs.RaftClient{})
	if err != nil {
		t.Logf("error during raft creation: %v", err)
		t.FailNow()
	}

	// heartbeat
	resp, err := r.AppendEntries(context.Background(), &rpcs.AppendEntryData{
		Term:         0,
		LeaderID:     1,
		PrevLogIndex: -1,
		PrevLogTerm:  -1,
		LeaderCommit: -1,
		Entries:      []*rpcs.Log{},
	})
	if err != nil || !resp.Success || resp.Term != 0 {
		t.Logf("AppendEntry 1 failed")
		t.FailNow()
	}

	resp, err = r.AppendEntries(context.Background(), &rpcs.AppendEntryData{
		Term:         0,
		LeaderID:     1,
		PrevLogIndex: -1,
		PrevLogTerm:  -1,
		LeaderCommit: 1,
		Entries: []*rpcs.Log{{
			Term: 0,
			Data: "start",
		}, {
			Term: 0,
			Data: "next",
		}},
	})
	if err != nil || !resp.Success || resp.Term != 0 {
		t.Logf("AppendEntry 2 failed")
		t.FailNow()
	}

	resp, err = r.AppendEntries(context.Background(), &rpcs.AppendEntryData{
		Term:         1,
		LeaderID:     1,
		PrevLogIndex: 1,
		PrevLogTerm:  0,
		LeaderCommit: 3,
		Entries: []*rpcs.Log{{
			Term: 0,
			Data: "",
		}, {
			Term: 1,
			Data: "final",
		}},
	})
	if err != nil || !resp.Success || resp.Term != 1 {
		t.Logf("AppendEntry 3 failed")
		t.FailNow()
	}

	// another heartbeat
	resp, err = r.AppendEntries(context.Background(), &rpcs.AppendEntryData{
		Term:         1,
		LeaderID:     1,
		PrevLogIndex: 3,
		PrevLogTerm:  1,
		LeaderCommit: 3,
		Entries:      []*rpcs.Log{},
	})
	if err != nil || !resp.Success || resp.Term != 1 {
		t.Logf("AppendEntry 4 failed")
		t.FailNow()
	}

	logs := r.GetLogs()
	if len(logs) != 4 {
		t.Logf("AppendEntry size is %d, not 4", len(logs))
		t.FailNow()
	}

	if logs[0] != "start" ||
		logs[1] != "next" ||
		logs[2] != "" ||
		logs[3] != "final" {

		t.Logf("AppendEntry logs are incorrect: %v", logs)
	}

	r.Stop()
}

func TestRequestVote(t *testing.T) {
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
