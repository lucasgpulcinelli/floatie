package raft

import "testing"

func TestCreateRaft(t *testing.T) {
	r, err := New(0, "localhost:9999", map[peerID]string{})
	if err != nil {
		t.Fatalf("creation: %v", err)
	}
	err = r.Stop()
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestCreateWithPeers(t *testing.T) {
	r, err := New(0, "localhost:9999", map[peerID]string{
		1: "localhost:2222", 2: "localhost:3333"},
	)
	if err != nil {
		t.Fatalf("creation: %v", err)
	}

	err = r.Stop()
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
}
