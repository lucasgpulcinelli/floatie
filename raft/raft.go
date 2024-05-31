// Package raft implements the raft protocol via the Raft struct, using gRPC to
// communicate between peers.
package raft

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
	"google.golang.org/grpc"
)

// A Raft represents a node in the raft protocol running locally.
//
// An instance should be stopped via the Stop method in order to properly stop
// the timer goroutine and the gRPC server if they exist.
//
// An instance must only have leader properties if the state == Leader.
type Raft struct {
	state       State
	currentTerm int32
	lastVoted   int32
	logs        []*rpcs.Log
	peers       map[int32]rpcs.RaftClient

	commitIndex      int32
	lastAppliedIndex int32

	lp             *LeaderProperties
	electionCancel context.CancelFunc

	id        int32
	leaderID  int32
	timerChan chan struct{}
	timerStop chan struct{}
	server    *grpc.Server

	requestCond            *sync.Cond
	commitGoroutineRunning bool
	applyLog               func(string) error

	mut sync.Mutex

	timings *RaftTimings

	rpcs.UnimplementedRaftServer
}

// A RaftTimings defines the timings constants related with the raft protocol.
type RaftTimings struct {
	TimeoutLow  time.Duration
	TimeoutHigh time.Duration

	HearbeatLow  time.Duration
	HearbeatHigh time.Duration
}

// A LeaderProperty is a value that must only be instantiated during a term
// where the current instance is a Leader.
type LeaderProperties struct {
	cancelHeartbeats context.CancelFunc
	nextIndex        map[int32]int32
	matchIndex       map[int32]int32
}

// New creates a new raft instance with a unique id and some peers. It does not
// create the gRPC server nor the timer goroutine to trigger elections.
// To do that, use the WithAddress and StartTimerLoop methods.
func New(id int32, peers map[int32]rpcs.RaftClient, applyLog func(string) error) (*Raft, error) {
	if peers == nil {
		return nil, fmt.Errorf("tried to create raft with nil peers")
	}
	if _, ok := peers[id]; ok {
		return nil, fmt.Errorf("raft peers contains self connection")
	}

	raft := &Raft{
		id:               id,
		state:            Follower,
		leaderID:         -1,
		lastVoted:        -1,
		commitIndex:      -1,
		lastAppliedIndex: -1,
		logs:             []*rpcs.Log{},
		peers:            peers,
		applyLog:         applyLog,
	}

	raft.requestCond = sync.NewCond(&raft.mut)

	return raft, nil
}

// WithAddress serves the raft server at an address specified.
func (raft *Raft) WithAddress(grpcAddr string) error {
	raft.mut.Lock()
	defer raft.mut.Unlock()

	// start gRPC server
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}

	raft.server = grpc.NewServer()
	rpcs.RegisterRaftServer(raft.server, raft)
	go raft.server.Serve(listener)

	return nil
}

// Stop stops the raft instance from sending and receiving RPCs and timing out
// to elect leaders. Deletes the gRPC server and the timer goroutine.
func (raft *Raft) Stop() {
	raft.mut.Lock()
	defer raft.mut.Unlock()

	slog.Debug("stopping raft instance")

	if raft.timerStop != nil {
		raft.timerStop <- struct{}{}
	}

	if raft.server != nil {
		raft.server.GracefulStop()
	}
}

func (raft *Raft) GetCurrentLeader() (int32, bool) {
	raft.mut.Lock()
	defer raft.mut.Unlock()

	return raft.leaderID, raft.leaderID > 0
}

func (raft *Raft) GetID() int32 {
	return raft.id
}
