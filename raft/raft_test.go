package raft_test

import (
	"context"
	"testing"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft"
	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
	"google.golang.org/grpc"
)

type raftMockClient struct {
	appendEntries func(*rpcs.AppendEntryData) *rpcs.RaftResult
	requestVote   func(*rpcs.RequestVoteData) *rpcs.RaftResult
}

func (rm *raftMockClient) AppendEntries(ctx context.Context, in *rpcs.AppendEntryData, opts ...grpc.CallOption) (*rpcs.RaftResult, error) {
	out := rm.appendEntries(in)
	return out, nil
}

func (rm *raftMockClient) RequestVote(ctx context.Context, in *rpcs.RequestVoteData, opts ...grpc.CallOption) (*rpcs.RaftResult, error) {
	out := rm.requestVote(in)
	return out, nil
}

func TestCreateRaft(t *testing.T) {
	peer := &raftMockClient{
		func(_ *rpcs.AppendEntryData) *rpcs.RaftResult { return nil },
		func(_ *rpcs.RequestVoteData) *rpcs.RaftResult { return nil },
	}

	r, err := raft.New(
		0,
		map[int32]rpcs.RaftClient{1: peer, 2: peer},
		func(s string) error { return nil },
	)
	if err != nil {
		t.Logf("error during creation: %v\n", err)
		t.FailNow()
	}

	r.Stop()
}

func TestCreateNilRaft(t *testing.T) {
	_, err := raft.New(0, nil, func(string) error { return nil })
	if err == nil {
		t.Log("creation passing nil did not error")
		t.FailNow()
	}
}

func TestCreateRaftSelfPeer(t *testing.T) {
	peer := &raftMockClient{
		func(_ *rpcs.AppendEntryData) *rpcs.RaftResult { return nil },
		func(_ *rpcs.RequestVoteData) *rpcs.RaftResult { return nil },
	}

	_, err := raft.New(
		0,
		map[int32]rpcs.RaftClient{1: peer, 0: peer},
		func(string) error { return nil },
	)
	if err == nil {
		t.Logf("creation passing self peer did not error")
		t.FailNow()
	}
}

func TestCreateRaftTimer(t *testing.T) {
	r, err := raft.New(
		0,
		map[int32]rpcs.RaftClient{},
		func(s string) error { return nil },
	)
	if err != nil {
		t.Logf("error during creation: %v\n", err)
		t.FailNow()
	}

	r.StartTimerLoop(&raft.RaftTimings{
		TimeoutLow:   20 * time.Millisecond,
		TimeoutHigh:  40 * time.Millisecond,
		HearbeatLow:  6 * time.Millisecond,
		HearbeatHigh: 7 * time.Millisecond,
	})

	r.Stop()
}
