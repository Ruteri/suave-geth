package main

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave_backends "github.com/ethereum/go-ethereum/suave/backends"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRemoteStore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	pebbleDir := t.TempDir()
	remoteStoreBackend := suave_backends.NewPebbleStoreBackend(pebbleDir)
	require.NoError(t, remoteStoreBackend.Start())
	defer remoteStoreBackend.Stop()

	go RunRemoteStore(ctx, Config{
		Host:  "127.0.0.1",
		Port:  18153,
		Store: remoteStoreBackend,
	})

	remoteStore := suave_backends.NewRemoteStoreBackend("http://127.0.0.1:18153")

	var err error

	testBid := suave.Bid{
		Id:                  types.BidId(uuid.New()),
		Salt:                types.BidId(uuid.New()),
		DecryptionCondition: 5,
	}

	time.Sleep(time.Second)
	err = remoteStore.InitializeBid(testBid)
	require.NoError(t, err)

	bid, err := remoteStore.FetchEngineBidById(testBid.Id)
	require.NoError(t, err)
	require.Equal(t, testBid, bid)

	_, err = remoteStore.Store(testBid, common.Address{}, "xx", []byte{0x10})
	require.NoError(t, err)

	fetchedBytes, err := remoteStore.Retrieve(testBid, common.Address{}, "xx")
	require.NoError(t, err)
	require.Equal(t, fetchedBytes, []byte{0x10})
}
