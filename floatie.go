package floatie

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/lucasgpulcinelli/floatie/raft"
	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type KVStore struct {
	raftInstance *raft.Raft
	storage      sync.Map
}

func (kv *KVStore) applyLog(log string) (any, error) {
	logSplit := strings.Split(log, " ")

	if len(logSplit) == 3 && logSplit[0] == "STORE" {
		kv.storage.Store(logSplit[1], logSplit[2])
	} else if len(logSplit) == 2 && logSplit[0] == "DELETE" {
		kv.storage.Delete(logSplit[1])
	} else if logSplit[0] == "GET" {
		value, _ := kv.storage.Load(logSplit[1])
		return value, nil
	} else {
		slog.Warn("Malformed action received", "log", log)
	}

	return nil, nil
}

func (kv *KVStore) sendLog(log string) (any, bool, int32) {
	for {
		result, ok := kv.raftInstance.SendLog(log)
		if ok {
			return result, true, -1
		}

		// it is not a problem that these state read operations are not 100%
		// consistent, because it only triggers another request in the worst case
		lid, exists := kv.raftInstance.GetCurrentLeader()
		if !exists {
			return nil, false, -1
		}

		if lid != kv.raftInstance.GetID() {
			return nil, false, lid
		}
	}
}

func (kv *KVStore) Get(key string) (string, bool, int32) {
	value, ok, lid := kv.sendLog(fmt.Sprintf("GET %s", key))
	if value == nil {
		return "", ok, lid
	}
	return value.(string), ok, lid
}

func (kv *KVStore) Store(key, value string) (bool, int32) {
	_, ok, lid := kv.sendLog(fmt.Sprintf("STORE %s %s", key, value))
	return ok, lid
}

func (kv *KVStore) Delete(key string) (bool, int32) {
	_, ok, lid := kv.sendLog(fmt.Sprintf("DELETE %s", key))
	return ok, lid
}

func NewKVStore() *KVStore {
	return &KVStore{
		storage: sync.Map{},
	}
}

func (kv *KVStore) WithAddress(address string) error {
	return kv.raftInstance.WithAddress(address)
}

func (kv *KVStore) WithRawCluster(id int32, peers map[int32]rpcs.RaftClient) error {
	var err error

	kv.raftInstance, err = raft.New(id, peers, kv.applyLog)

	return err
}

func (kv *KVStore) WithCluster(id int32, peers map[int32]string) error {
	peerM := map[int32]rpcs.RaftClient{}
	for i, host := range peers {
		if i == id {
			return errors.New("peers should not contain self")
		}
		conn, err := grpc.Dial(
			host,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)

		if err != nil {
			return err
		}

		peerM[i] = rpcs.NewRaftClient(conn)
	}

	return kv.WithRawCluster(id, peerM)
}

func (kv *KVStore) WithTimer(timings *raft.RaftTimings) error {
	if kv.raftInstance == nil {
		return errors.New("Must initialize cluster beforehand")
	}

	kv.raftInstance.StartTimerLoop(timings)
	return nil
}
