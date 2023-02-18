package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	apiV1 "github/m-wrona/raft-go/generated/api/v1"
	raftV1 "github/m-wrona/raft-go/generated/raft/v1"
)

type TestServer struct {
	Server         *grpc.Server
	Client         *grpc.ClientConn
	RaftClient     raftV1.RaftServiceClient
	KeyValueClient apiV1.KeyValueServiceClient
}

func StartTestGrpcServer(clusters []string, proposeC chan string, confChangeC chan raftpb.ConfChange) *TestServer {
	var kvs *kvstore
	getSnapshot := func() ([]byte, error) { return kvs.getSnapshot() }
	commitC, errorC, snapshotterReady := newRaftNode(1, clusters, false, getSnapshot, proposeC, confChangeC)
	kvs = newKVStore(<-snapshotterReady, proposeC, commitC, errorC)

	time.Sleep(500 * time.Millisecond) // time for leader election

	serverUrl := RandomServerUrl()
	log, _ := zap.NewDevelopment()
	server := grpc.NewServer()
	newController(server, log, kvs, confChangeC)

	go func() {
		log.Debug("Starting test GRPC server...", zap.String("url", serverUrl))
		listen, err := net.Listen("tcp", serverUrl)
		if err != nil {
			panic(fmt.Errorf("GRPC listen error: %s", err))
		}
		log.Debug("Serving test GRPC server...", zap.String("url", serverUrl))
		if err := server.Serve(listen); err != nil {
			panic(fmt.Errorf("GRPC serve error: %s", err))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	conn, err := grpc.DialContext(ctx, serverUrl, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		panic(fmt.Errorf("GRPC client connection error: %s", err))
	}

	return &TestServer{
		Server:         server,
		Client:         conn,
		RaftClient:     raftV1.NewRaftServiceClient(conn),
		KeyValueClient: apiV1.NewKeyValueServiceClient(conn),
	}
}

func RandomPort() int {
	listen := RandomListener("tcp")
	idx := strings.LastIndex(listen.Addr().String(), ":")
	p := listen.Addr().String()[idx+1:]
	if port, err := strconv.Atoi(p); err != nil {
		panic(err)
	} else {
		return port
	}
}

func RandomServerUrl() string {
	return fmt.Sprintf("localhost:%d", RandomPort())
}

func RandomListener(network string) net.Listener {
	listen, err := net.Listen(network, ":0")
	if err == nil {
		return listen
	}
	panic(errors.New("couldn't create random listener"))
}
