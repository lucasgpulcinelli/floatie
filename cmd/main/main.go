package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lucasgpulcinelli/floatie/raft"
	"google.golang.org/grpc"
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

	slog.Debug("starting Raft...")

	var err error
	raftInstance, err = raft.New(0, ":8081", map[int32]*grpc.ClientConn{}, &raft.RaftOption{
		TimeoutLow:  5 * time.Second,
		TimeoutHigh: 10 * time.Second,
		DeltaLow:    1 * time.Second,
		DeltaHigh:   2 * time.Second,
	})
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
