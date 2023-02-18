package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/raft/v3/raftpb"

	apiV1 "github/m-wrona/raft-go/generated/api/v1"
)

func Test_Service_SingleNode_PutAndGetValue(t *testing.T) {
	proposeC := make(chan string)
	defer close(proposeC)

	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)

	clusters := []string{"http://127.0.0.1:9021"}
	sut := StartTestGrpcServer(1, clusters, proposeC, confChangeC, t.TempDir())
	defer sut.Server.Stop()

	var wantValue uint32 = 2
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	setValueResp, err := sut.KeyValueClient.Set(ctx, &apiV1.SetValueRequest{
		Value: wantValue,
	})
	require.Nilf(t, err, "value not set: %s", err)
	require.Truef(t, setValueResp.GetOk(), "value not set")

	assertValueEquals(t, sut.KeyValueClient, wantValue)
}

func Test_Service_MultiNode_PutAndGetValue(t *testing.T) {
	suts := make([]*TestServer, 0)
	gr := sync.WaitGroup{}
	mux := sync.Mutex{}

	clusters := []string{"http://127.0.0.1:9021", "http://127.0.0.1:9022"}
	gr.Add(len(clusters))

	for i := 1; i <= len(clusters); i++ {
		proposeC := make(chan string)
		confChangeC := make(chan raftpb.ConfChange)
		id := i
		go func() {
			defer gr.Done()
			s := StartTestGrpcServer(id, clusters, proposeC, confChangeC, t.TempDir())
			mux.Lock()
			suts = append(suts, s)
			mux.Unlock()
		}()
	}
	gr.Wait()

	var wantValue uint32 = 2
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	setValueResp, err := suts[0].KeyValueClient.Set(ctx, &apiV1.SetValueRequest{
		Value: wantValue,
	})
	cancel()
	require.Nilf(t, err, "value not set: %s", err)
	require.Truef(t, setValueResp.GetOk(), "value not set")

	assertValueEquals(t, suts[0].KeyValueClient, wantValue)
	assertValueEquals(t, suts[1].KeyValueClient, wantValue)

	var wantValue2 uint32 = 3
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	setValueResp2, err2 := suts[1].KeyValueClient.Set(ctx2, &apiV1.SetValueRequest{
		Value: wantValue2,
	})
	cancel2()
	require.Nilf(t, err2, "value not set: %s", err)
	require.Truef(t, setValueResp2.GetOk(), "value not set")

	assertValueEquals(t, suts[0].KeyValueClient, wantValue2)
	assertValueEquals(t, suts[1].KeyValueClient, wantValue2)
}
