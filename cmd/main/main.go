package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lucasgpulcinelli/floatie"
	"github.com/lucasgpulcinelli/floatie/raft"
)

var (
	kv          *floatie.KVStore
	id          = flag.Int("id", 1, "the id this instance should have")
	config      = flag.String("config", "floatie.json", "floatie cluster definition file")
	raftAddress = flag.String("raft-addr", ":9999", "address to use for raft")
	httpAddress = flag.String("http-addr", ":8080", "address to use for http")
	cluster     map[int32][2]string
)

func setupLogging() {
	l := &slog.LevelVar{}
	l.Set(slog.LevelInfo)
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l}),
	)
	slog.SetDefault(logger)
}

func setupKV() error {
	file, err := os.Open(*config)
	if err != nil {
		return err
	}

	err = json.NewDecoder(file).Decode(&cluster)
	if err != nil {
		return err
	}

	peers := map[int32]string{}
	for i, addresses := range cluster {
		if int32(*id) == i {
			continue
		}

		peers[i] = addresses[1]
	}

	kv = floatie.NewKVStore()
	err = kv.WithCluster(int32(*id), peers)
	if err != nil {
		return err
	}

	err = kv.WithAddress(*raftAddress)
	if err != nil {
		return err
	}

	err = kv.WithTimer(&raft.RaftTimings{
		HearbeatLow:  time.Millisecond * 50,
		HearbeatHigh: time.Millisecond * 80,
		TimeoutLow:   time.Millisecond * 150,
		TimeoutHigh:  time.Millisecond * 300,
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	setupLogging()

	slog.Debug("starting Raft")

	err := setupKV()
	if err != nil {
		panic(err)
	}

	http.HandleFunc("GET /api/v1/floatieDB", getHandler)
	http.HandleFunc("POST /api/v1/floatieDB", postHandler)
	http.HandleFunc("DELETE /api/v1/floatieDB", deleteHandler)

	slog.Debug("starting server")

	err = http.ListenAndServe(*httpAddress, nil)
	slog.Error("%v", err)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Needs 'key' parameter")
		return
	}

	value, ok, leader := kv.Get(key)

	if ok && value == "" {
		w.WriteHeader(404)
	} else if ok {
		w.WriteHeader(200)
		fmt.Fprint(w, value)
	} else if leader > 0 {
		w.Header().Add("Location", "http://"+cluster[leader][0])
		w.WriteHeader(307)
	} else {
		w.WriteHeader(503)
		fmt.Fprintln(w, "No leader at the moment")
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Needs 'key' parameter")
		return
	}

	value := r.URL.Query().Get("value")
	if value == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Needs 'value' parameter")
		return
	}

	ok, leader := kv.Store(key, value)
	if ok {
		w.WriteHeader(201)
	} else if leader > 0 {
		w.Header().Add("Location", "http://"+cluster[leader][0])
		w.WriteHeader(307)
	} else {
		w.WriteHeader(503)
		fmt.Fprintln(w, "No leader at the moment")
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Needs 'key' parameter")
		return
	}

	ok, leader := kv.Delete(key)
	if ok {
		w.WriteHeader(200)
	} else if leader > 0 {
		w.Header().Add("Location", "http://"+cluster[leader][0])
		w.WriteHeader(307)
	} else {
		w.WriteHeader(503)
		fmt.Fprintln(w, "No leader at the moment")
	}
}
