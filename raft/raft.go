package raft

import (
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
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
	peers       []*grpc.ClientConn

	commitIndex      int
	lastAppliedIndex int

	lp *LeaderProperties

	timerChan chan time.Duration
	mut       sync.Mutex
}

type Log struct {
	Term int
	Data any
}

type LeaderProperties struct {
	nextIndex  []int
	matchIndex []int
}

func NewRaft(grpcAddr string, peerAddresses []string) (*Raft, error) {
	raft := &Raft{state: Follower, lastVoted: -1, logs: []*Log{}}

	raft.peers = []*grpc.ClientConn{}
	for _, peer := range peerAddresses {
		conn, err := grpc.Dial(peer)
		if err != nil {
			return nil, err
		}
		raft.peers = append(raft.peers, conn)
	}

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer()
	go server.Serve(listener)

	raft.createTimerGoroutine()
	return raft, nil
}

func (raft *Raft) createTimerGoroutine() {
	raft.timerChan = make(chan time.Duration)
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
		}
		started = time.Now()
	}
}
