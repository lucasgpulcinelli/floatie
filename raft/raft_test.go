package raft

import "testing"

func TestCreateRaft(t *testing.T) {
	r, err := NewRaft("localhost:9999", []string{})
	if err != nil {
		t.Fatalf("creation: %v", err)
	}
	err = r.Stop()
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestCreateWithPeers(t *testing.T) {
	r, err := NewRaft("localhost:9999", []string{
		"localhost:2222", "localhost:3333"},
	)
	if err != nil {
		t.Fatalf("creation: %v", err)
	}

	err = r.Stop()
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
}
