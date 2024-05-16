package raft_test

import (
	"testing"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft"
	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
)

func TestCreateEmptyRaft(t *testing.T) {
	r, err := raft.New(0, map[int32]rpcs.RaftClient{})
	if err != nil {
		t.Logf("error during creation: %v\n", err)
		t.FailNow()
	}

	r.Stop()
}

func TestCreateNilRaft(t *testing.T) {
	_, err := raft.New(0, nil)
	if err == nil {
		t.Log("creation passing nil did not error")
		t.FailNow()
	}
}

func TestCreateRaftSelfPeer(t *testing.T) {
	_, err := raft.New(0, map[int32]rpcs.RaftClient{0: nil})
	if err == nil {
		t.Logf("creation passing self peer did not error")
		t.FailNow()
	}
}

func TestCreateRaftTimer(t *testing.T) {
	r, err := raft.New(0, map[int32]rpcs.RaftClient{})
	if err != nil {
		t.Logf("error during creation: %v\n", err)
		t.FailNow()
	}

	r.StartTimerLoop(&raft.RaftTimings{
		TimeoutLow:  20 * time.Millisecond,
		TimeoutHigh: 40 * time.Millisecond,
		DeltaLow:    8 * time.Millisecond,
		DeltaHigh:   12 * time.Millisecond,
	})

	r.Stop()
}
