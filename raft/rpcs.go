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
	return nil, fmt.Errorf("Not implemented")
}
