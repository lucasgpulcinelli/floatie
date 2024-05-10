// Package raft implements the raft protocol via the Raft struct, using gRPC to
// communicate between peers.
package raft

import (
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
	"google.golang.org/grpc"
)

// A Raft represents a node in the raft protocol running locally.
//
// The instance, when creted with the New() function, connects with the other
// peers, elects leaders, and replicate logs via gRPC and a timer goroutine
// automatically.
//
// An instance should be stopped via the Stop method in order to properly stop
// the timer goroutine and the gRPC server.
//
// An instance must only have leader properties if the state == Leader.
type Raft struct {
	state       State
	currentTerm int32
	lastVoted   int32
	logs        []*rpcs.Log
	peers       map[int32]*grpc.ClientConn

	commitIndex      int32
	lastAppliedIndex int32

	lp *LeaderProperties

	id        int32
	timerChan chan time.Duration
	timerStop chan struct{}
	server    *grpc.Server

	mut sync.Mutex

	options *RaftOption

	rpcs.UnimplementedRaftServer
}

// A RaftOption defines the timings and constants related with the raft
// protocol.
type RaftOption struct {
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

// New creates a new raft instance with a unique id, some peers and an address
// to expose the gRPC service. It creates the gRPC server as well as the timer
// goroutine to trigger elections.
// The function may return without a proper leader elected.
func New(id int32, grpcAddr string, peers map[int32]*grpc.ClientConn, options *RaftOption) (*Raft, error) {
	raft := &Raft{
		id:          id,
		state:       Follower,
		lastVoted:   -1,
		commitIndex: -1,
		logs:        []*rpcs.Log{},
		options:     options,
		peers:       peers,
	}

	raft.mut.Lock()
	defer raft.mut.Unlock()

	// start gRPC server
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, err
	}

	raft.server = grpc.NewServer()
	rpcs.RegisterRaftServer(raft.server, raft)
	go raft.server.Serve(listener)

	// start timer goroutine
	raft.timerChan = make(chan time.Duration)
	raft.timerStop = make(chan struct{}, 0)
	timeout := randDuration(raft.options.TimeoutLow, raft.options.TimeoutHigh)

	go timerLoop(timeout, raft.timerChan, raft.timerStop, func() {
		go raft.triggerElection()
	})

	return raft, nil
}

// Stop stops the raft instance from sending and receiving RPCs and timing out
// to elect leaders. Deletes the gRPC server and the timer goroutine.
func (raft *Raft) Stop() error {
	slog.Debug("stopping raft instance")

	raft.mut.Lock()
	defer raft.mut.Unlock()

	raft.timerStop <- struct{}{}

	raft.server.GracefulStop()

	errs := []error{}
	for _, conn := range raft.peers {
		errs = append(errs, conn.Close())
	}

	return errors.Join(errs...)
}
