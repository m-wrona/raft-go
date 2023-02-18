package main

import (
	"flag"
	"fmt"
	"strings"

	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cluster := flag.String("cluster", "http://127.0.0.1:9021", "comma separated cluster peers")
	id := flag.Int("id", 1, "node ID")
	kvPort := flag.Int("port", 9121, "key-value server port")
	join := flag.Bool("join", false, "join an existing cluster")
	storePath := flag.String("storePath", "./", "path where raft state will be kept")
	flag.Parse()

	log, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	proposeC := make(chan string)
	defer close(proposeC)
	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)

	// raft provides a commit stream for the proposals from the http api
	var kvs *kvstore
	getSnapshot := func() ([]byte, error) { return kvs.getSnapshot() }
	commitC, errorC, snapshotterReady := newRaftNode(*id, strings.Split(*cluster, ","), *join, getSnapshot, proposeC, confChangeC, *storePath)

	kvs = newKVStore(<-snapshotterReady, proposeC, commitC, errorC)

	server := grpc.NewServer()
	newController(server, log, kvs, confChangeC)

	startGRPC(server, Config{Address: fmt.Sprintf("0.0.0.0:%d", *kvPort), Network: "tcp"}, log)
}
