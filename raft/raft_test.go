package raft

import (
	"testing"
	"time"
)

func TestCreateRaft(t *testing.T) {
	r, err := New(0, "localhost:9999", map[int32]string{}, &RaftOption{
		TimeoutLow:  1 * time.Millisecond,
		TimeoutHigh: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("creation: %v", err)
	}
	err = r.Stop()
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestCreateWithPeers(t *testing.T) {
	r, err := New(0, "localhost:9999", map[int32]string{
		1: "localhost:2222", 2: "localhost:3333"},
		&RaftOption{
			TimeoutLow:  1 * time.Millisecond,
			TimeoutHigh: 10 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatalf("creation: %v", err)
	}

	err = r.Stop()
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
}
