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
	timerChan chan time.Duration
	timerStop chan struct{}
	server    *grpc.Server

	mut sync.Mutex

	timings *RaftTimings

	rpcs.UnimplementedRaftServer
}

// A RaftTimings defines the timings constants related with the raft protocol.
type RaftTimings struct {
	TimeoutLow  time.Duration
	TimeoutHigh time.Duration

	DeltaLow  time.Duration
	DeltaHigh time.Duration
}

// A LeaderProperty is a value that must only be instantiated during a term
// where the current instance is a Leader.
type LeaderProperties struct {
	nextIndex  []int32
	matchIndex []int32
}

// New creates a new raft instance with a unique id and some peers. It does not
// create the gRPC server nor the timer goroutine to trigger elections.
// To do that, use the WithAddress and StartTimerLoop methods.
func New(id int32, peers map[int32]rpcs.RaftClient) (*Raft, error) {
	if peers == nil {
		return nil, fmt.Errorf("tried to create raft with nil peers")
	}
	if _, ok := peers[id]; ok {
		return nil, fmt.Errorf("raft peers contains self connection")
	}
	return &Raft{
		id:               id,
		state:            Follower,
		lastVoted:        -1,
		commitIndex:      -1,
		lastAppliedIndex: -1,
		logs:             []*rpcs.Log{},
		peers:            peers,
	}, nil
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

// StartTimerLoop starts the timer goroutine to trigger elections if the leader
// fails to send heartbeats.
func (raft *Raft) StartTimerLoop(timings *RaftTimings) {
	raft.mut.Lock()
	defer raft.mut.Unlock()

	raft.timings = timings

	raft.timerChan = make(chan time.Duration)
	raft.timerStop = make(chan struct{}, 0)
	timeout := randDuration(timings.TimeoutLow, timings.TimeoutHigh)

	go timerLoop(timeout, raft.timerChan, raft.timerStop, func() {
		go func() {
			raft.mut.Lock()
			slog.Debug("timeout occurred")
			if raft.state != Leader {
				raft.setState(Candidate)
			}
			raft.mut.Unlock()
		}()
	})
}

func (raft *Raft) GetLogs() []string {
	raft.mut.Lock()
	defer raft.mut.Unlock()
	return raft.getLogs()
}

func (raft *Raft) getLogs() []string {
	ls := []string{}
	for _, v := range raft.logs {
		ls = append(ls, v.Data)
	}
	return ls
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
