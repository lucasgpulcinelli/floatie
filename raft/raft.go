// Package raft implements the raft protocol via the Raft struct, using gRPC to
// communicate between peers.
package raft

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// State defines the moment in the raft FSM that the instance is in.
type State byte

// peerID defines the identifier for peers.
type peerID int

// The possible states for the raft FSM.
const (
	Follower State = iota
	Candidate
	Leader
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
	currentTerm int
	lastVoted   peerID
	logs        []*Log
	peers       map[peerID]*grpc.ClientConn

	commitIndex      int
	lastAppliedIndex int

	lp *LeaderProperties

	id        peerID
	timerChan chan time.Duration
	timerStop chan struct{}
	server    *grpc.Server

	rpcs.UnimplementedRaftServer
}

// A Log represents a log in the raft protocol, created at a certain term and
// containing some data.
type Log struct {
	Term int
	Data any
}

// A LeaderProperty is a value that must only be instantiated during a term
// where the current instance is a Leader.
type LeaderProperties struct {
	nextIndex  []int
	matchIndex []int
}

// New creates a new raft instance with a unique id, some peers and an address
// to expose the gRPC service. It creates the gRPC server as well as the timer
// goroutine to trigger elections.
// The function may return without a proper leader elected.
func New(id peerID, grpcAddr string, peerAddresses map[peerID]string) (*Raft, error) {
	raft := &Raft{id: id, state: Follower, lastVoted: -1, logs: []*Log{}}

	raft.peers = map[peerID]*grpc.ClientConn{}
	for id, address := range peerAddresses {
		conn, err := grpc.Dial(address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		raft.peers[id] = conn
	}

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, err
	}

	raft.server = grpc.NewServer()
	rpcs.RegisterRaftServer(raft.server, raft)
	go raft.server.Serve(listener)

	raft.createTimerGoroutine()
	return raft, nil
}

// Stop stops the raft instance from sending and receiving RPCs and timing out
// to elect leaders. Deletes the gRPC server and the timer goroutine.
func (raft *Raft) Stop() error {
	raft.timerStop <- struct{}{}
	raft.server.GracefulStop()

	errs := []error{}
	for _, conn := range raft.peers {
		errs = append(errs, conn.Close())
	}

	return errors.Join(errs...)
}

func (raft *Raft) createTimerGoroutine() {
	raft.timerChan = make(chan time.Duration)
	raft.timerStop = make(chan struct{}, 0)
	go raft.timerLoop()
}

// timerLoop runs in a separate goroutine to trigger leader election if a
// certain time passes as per the protocol. It can only exit if a value is
// received in the raft.timerStop channel.
func (raft *Raft) timerLoop() {
	ticker := time.NewTicker(time.Millisecond * 10)
	started := time.Now()
	for {
		select {
		case <-ticker.C:
			fmt.Println("timeout, start election")
			ticker.Reset(time.Millisecond * 10)
		case d := <-raft.timerChan:
			ticker.Reset(time.Now().Sub(started) + d)
		case <-raft.timerStop:
			return
		}
		started = time.Now()
	}
}
