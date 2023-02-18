package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apiV1 "github/m-wrona/raft-go/generated/api/v1"
)

func assertValueEquals(t *testing.T, client apiV1.KeyValueServiceClient, wantValue uint32) {
	var getValueResp *apiV1.GetValueResponse
	var err error
	for i := 0; i < 50; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		getValueResp, err = client.Get(ctx, &apiV1.GetValueRequest{})
		cancel()
		if getValueResp != nil && getValueResp.GetValue() == wantValue {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.Nilf(t, err, "value not read: %s", err)
	require.Equal(t, wantValue, getValueResp.GetValue(), "value not read")
}
