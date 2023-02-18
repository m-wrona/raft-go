package main

import (
	"context"
	"errors"
	"strconv"

	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	apiV1 "github/m-wrona/raft-go/model/api/v1"
	raftV1 "github/m-wrona/raft-go/model/raft/v1"
)

const (
	valueKey = "sample-value"
)

type controller struct {
	log         *zap.Logger
	store       *kvstore
	confChangeC chan<- raftpb.ConfChange
}

func newController(
	server *grpc.Server,
	log *zap.Logger,
	store *kvstore,
	confChangeC chan<- raftpb.ConfChange,
) *controller {
	c := &controller{
		log:         log.With(zap.String("component", "grpcController")),
		store:       store,
		confChangeC: confChangeC,
	}
	grpc_health_v1.RegisterHealthServer(server, c)
	apiV1.RegisterKeyValueServiceServer(server, c)
	raftV1.RegisterRaftServiceServer(server, c)
	return c
}

func (c *controller) Check(ctx context.Context, request *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	c.log.Debug("Healthcheck request received", zap.Any("request", request))
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (c *controller) Watch(request *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	return server.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}

func (c *controller) Set(ctx context.Context, request *apiV1.SetValueRequest) (*apiV1.SetValueResponse, error) {
	c.log.Debug("Set value request received", zap.Any("request", request))

	c.store.Propose(valueKey, strconv.Itoa(int(request.Value)))
	// Optimistic-- no waiting for ack from raft. Value is not yet
	// committed so a subsequent GET on the key may return old value
	return &apiV1.SetValueResponse{Ok: true}, nil
}

func (c *controller) Get(ctx context.Context, request *apiV1.GetValueRequest) (*apiV1.GetValueResponse, error) {
	if v, ok := c.store.Lookup(valueKey); ok {
		if i, err := strconv.Atoi(v); err != nil {
			return nil, err
		} else {
			return &apiV1.GetValueResponse{Value: uint32(i)}, nil
		}
	}
	return nil, errors.New("value not found")
}

func (c *controller) Add(ctx context.Context, request *raftV1.NodeRequest) (*raftV1.NodeResponse, error) {
	c.log.Debug("Add node request received", zap.Any("request", request))
	cc := raftpb.ConfChange{
		Type:   raftpb.ConfChangeAddNode,
		NodeID: request.Id,
	}
	c.confChangeC <- cc
	return &raftV1.NodeResponse{Ok: true}, nil
}

func (c *controller) Remove(ctx context.Context, request *raftV1.NodeRequest) (*raftV1.NodeResponse, error) {
	c.log.Debug("Remove node request received", zap.Any("request", request))
	cc := raftpb.ConfChange{
		Type:   raftpb.ConfChangeRemoveNode,
		NodeID: request.Id,
	}
	c.confChangeC <- cc
	return &raftV1.NodeResponse{Ok: true}, nil
}
