package raft

import "fmt"

func (raft *Raft) setState(state State) {
	switch raft.state {
	case Leader:
		raft.stopLeader()
	case Candidate:
		raft.abortElection()
	case Follower:
	}

	raft.state = state
}

func (raft *Raft) stopLeader() {

}

func (raft *Raft) abortElection() {

}

func (raft *Raft) triggerElection() {
	if raft.state == Leader {
		panic("triggered election while leader")
	}

	raft.setState(Candidate)

	fmt.Println("timeout, start election")
}
