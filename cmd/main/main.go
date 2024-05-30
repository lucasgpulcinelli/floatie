package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft"
	"github.com/lucasgpulcinelli/floatie/raft/rpcs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	raftInstance *raft.Raft
)

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

	raftInstance, err = raft.New(int32(id), peers)
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

func handler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "Hello, Raft!")
}
