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

	apiV1 "github/m-wrona/raft-go/model/api/v1"
	raftV1 "github/m-wrona/raft-go/model/raft/v1"
)

type TestServer struct {
	Server         *grpc.Server
	Client         *grpc.ClientConn
	RaftClient     raftV1.RaftServiceClient
	KeyValueClient apiV1.KeyValueServiceClient
}

func StartTestGrpcServer(id int, clusters []string, proposeC chan string, confChangeC chan raftpb.ConfChange, dirPath string) *TestServer {
	var kvs *kvstore
	getSnapshot := func() ([]byte, error) { return kvs.getSnapshot() }
	join := id > 1
	commitC, errorC, snapshotterReady := newRaftNode(id, clusters, join, getSnapshot, proposeC, confChangeC, dirPath)
	kvs = newKVStore(<-snapshotterReady, proposeC, commitC, errorC)

	time.Sleep(500 * time.Millisecond)

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

	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 60; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		conn, err = grpc.DialContext(ctx, serverUrl, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		cancel()
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		panic(fmt.Errorf("GRPC client %d connection error - url: %s, err: %s", id, serverUrl, err))
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
