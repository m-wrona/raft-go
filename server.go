package main

import (
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
)

type Config struct {
	Network string `envconfig:"GRPC_NETWORK" default:"tcp"`
	Address string `envconfig:"GRPC_ADDR" default:"0.0.0.0:8080"`
}

func Start(server *grpc.Server, config Config, log *zap.Logger) {
	go func() {
		listen, err := net.Listen(config.Network, config.Address)
		if err != nil {
			log.Panic("Failed to listen on GRPC address", zap.Error(err))
		}
		log.Info("Server started - waiting for incoming connections...")
		if err := server.Serve(listen); err != nil {
			log.Panic("Failed to serve GRPC", zap.Error(err))
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	<-c
	log.Info("Stopping server...")
	server.GracefulStop()
}
