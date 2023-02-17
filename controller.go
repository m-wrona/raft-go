package main

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	v1 "github/m-wrona/raft-go/generated/api/v1"
)

type controller struct {
	log *zap.Logger
}

func newController(server *grpc.Server, log *zap.Logger) *controller {
	c := &controller{
		log: log.With(zap.String("component", "grpcController")),
	}
	grpc_health_v1.RegisterHealthServer(server, c)
	v1.RegisterBroadcastServer(server, c)
	return c
}

func (c *controller) Broadcast(ctx context.Context, request *v1.BroadcastRequest) (*v1.BroadcastResponse, error) {
	c.log.Debug("Broadcast request received", zap.Any("request", request))
	// TODO return RAFT status instead
	return &v1.BroadcastResponse{
		Message: "ok",
	}, nil
}

func (c *controller) Check(ctx context.Context, request *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	c.log.Debug("Healthcheck request received", zap.Any("request", request))
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (c *controller) Watch(request *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	return server.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
