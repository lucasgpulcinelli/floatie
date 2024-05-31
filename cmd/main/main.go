package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft"
	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	raftInstance *raft.Raft
	storage      = sync.Map{}
)

func applyLog(log string) error {
	action, key := "", ""
	fmt.Sscanf(log, "%s %s %s", &action, &key)

	if action == "POST" {
		value := ""
		fmt.Sscanf(log, "%s %s %s", &action, &key, &value)
		storage.Store(key, value)
	} else if action == "DELETE" {
		storage.Delete(key)
	}

	return nil
}

func getStored(key string) (string, error) {
	value, ok := storage.Load(key)
	if !ok {
		return "", errors.New("not found")
	}
	return value.(string), nil
}

func main() {
	l := &slog.LevelVar{}
	l.Set(slog.LevelDebug)
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l}),
	)

	slog.SetDefault(logger)
	slog.Debug("starting Raft")

	idS, ok := os.LookupEnv("ID")
	if !ok {
		panic("needs ID environment variable")
	}

	id, err := strconv.Atoi(idS)
	if err != nil {
		panic(err)
	}

	peersS, ok := os.LookupEnv("PEERS")
	if !ok {
		panic("needs PEERS environment variable")
	}

	peersN, err := strconv.Atoi(peersS)
	if err != nil {
		panic(err)
	}

	peers := map[int32]rpcs.RaftClient{}
	for i := 1; i <= peersN; i++ {
		if i == id {
			continue
		}
		conn, err := grpc.Dial(
			fmt.Sprintf("floatie-%d:9999", i),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)

		if err != nil {
			panic(err)
		}

		peers[int32(i)] = rpcs.NewRaftClient(conn)
	}

	raftInstance, err = raft.New(int32(id), peers, applyLog)
	if err != nil {
		panic(err)
	}

	raftInstance.StartTimerLoop(&raft.RaftTimings{
		TimeoutLow:   5 * time.Second,
		TimeoutHigh:  10 * time.Second,
		HearbeatLow:  3 * time.Second,
		HearbeatHigh: 4 * time.Second,
	})

	err = raftInstance.WithAddress(":9999")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", handler)

	slog.Debug("starting server")

	err = http.ListenAndServe(":8080", nil)
	slog.Error("%v", err)
}

func handler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Needs 'key' parameter")
		return
	}
	value := r.URL.Query().Get("value")
	if r.Method == "POST" && value == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Needs 'value' parameter")
		return
	}

	if r.Method == "GET" {
		value, err := getStored(key)

		if err != nil && err.Error() == "not found" {
			w.WriteHeader(404)
			return
		}
		if err != nil {
			w.WriteHeader(500)
			slog.Error("error getting value", "error", err)
			return
		}

		w.WriteHeader(200)
		fmt.Fprintf(w, "%s", value)
		return
	}
	if r.Method != "POST" && r.Method != "DELETE" {
		w.WriteHeader(405)
		return
	}

	for {
		log := fmt.Sprintf("%s %s", r.Method, key)
		if r.Method == "POST" {
			log += " " + value
		}

		ok := raftInstance.SendLog(log)
		if ok {
			code := 200
			if r.Method == "POST" {
				code = 201
			}
			w.WriteHeader(code)
			return
		}

		lid, ok := raftInstance.GetCurrentLeader()
		if !ok {
			w.WriteHeader(503)
			fmt.Fprintln(w, "No leader at the moment")
			return
		}

		if lid != raftInstance.GetID() {
			w.Header().Add("Location", fmt.Sprintf("http://floatie-%d:8080", lid))
			w.WriteHeader(307)
			return
		}
	}
}
