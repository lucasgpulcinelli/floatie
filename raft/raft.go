package raft

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type State byte
type peerID int

const (
	Follower State = iota
	Candidate
	Leader
)

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
}

type Log struct {
	Term int
	Data any
}

type LeaderProperties struct {
	nextIndex  []int
	matchIndex []int
}

func NewRaft(id peerID, grpcAddr string, peerAddresses map[peerID]string) (*Raft, error) {
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
	go raft.server.Serve(listener)

	raft.createTimerGoroutine()
	return raft, nil
}

func (raft *Raft) Stop() error {
	raft.timerStop <- struct{}{}
	raft.server.GracefulStop()

	return nil
}

func (raft *Raft) createTimerGoroutine() {
	raft.timerChan = make(chan time.Duration)
	raft.timerStop = make(chan struct{}, 0)
	go raft.timerLoop()
}

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
