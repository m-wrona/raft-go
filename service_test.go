package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/raft/v3/raftpb"

	apiV1 "github/m-wrona/raft-go/generated/api/v1"
)

func Test_Service_PutAndGetValue(t *testing.T) {
	clusters := []string{"http://127.0.0.1:9021"}

	_ = os.RemoveAll("raftexample-1")
	_ = os.RemoveAll("raftexample-1-snap")
	defer func() {
		_ = os.RemoveAll("raftexample-1")
		_ = os.RemoveAll("raftexample-1-snap")
	}()

	proposeC := make(chan string)
	defer close(proposeC)

	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)

	sut := StartTestGrpcServer(clusters, proposeC, confChangeC)
	defer sut.Server.Stop()

	var wantValue uint32 = 2
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	setValueResp, err := sut.KeyValueClient.Set(ctx, &apiV1.SetValueRequest{
		Value: wantValue,
	})
	require.Nil(t, err, "value not set")
	require.Truef(t, setValueResp.GetOk(), "value not set")

	time.Sleep(3 * time.Second)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	getValueResp, err := sut.KeyValueClient.Get(ctx2, &apiV1.GetValueRequest{})
	require.Nilf(t, err, "value not read: %s", err)
	require.Equal(t, wantValue, getValueResp.GetValue(), "value not read")
}
