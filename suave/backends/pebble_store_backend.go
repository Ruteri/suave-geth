package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type PebbleStoreBackend struct {
	ctx    context.Context
	cancel context.CancelFunc
	dbPath string
	db     *pebble.DB
}

func NewPebbleStoreBackend(dbPath string) *PebbleStoreBackend {
	// TODO: should we check sanity in the constructor?
	return &PebbleStoreBackend{
		dbPath: dbPath,
	}
}

func (b *PebbleStoreBackend) Start() error {
	if b.cancel != nil {
		b.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.ctx = ctx

	db, err := pebble.Open(b.dbPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("could not open pebble database at %s: %w", b.dbPath, err)
	}

	go func() {
		<-ctx.Done()
		db.Close()
	}()

	b.db = db

	return nil
}

func (b *PebbleStoreBackend) Stop() error {
	b.cancel()
	return nil
}

func (b *PebbleStoreBackend) InitializeBid(bid suave.Bid) error {
	key := []byte(formatBidKey(bid.Id))

	_, closer, err := b.db.Get(key)
	if !errors.Is(err, pebble.ErrNotFound) {
		if err == nil {
			closer.Close()
		}
		return suave.ErrBidAlreadyPresent
	}

	data, err := json.Marshal(bid)
	if err != nil {
		return err
	}

	return b.db.Set(key, data, nil)
}

func (b *PebbleStoreBackend) FetchEngineBidById(bidId suave.BidId) (suave.Bid, error) {
	key := []byte(formatBidKey(bidId))

	bidData, closer, err := b.db.Get(key)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("bid %x not found: %w", bidId, err)
	}

	var bid suave.Bid
	err = json.Unmarshal(bidData, &bid)
	closer.Close()
	if err != nil {
		return suave.Bid{}, fmt.Errorf("could not unmarshal stored bid: %w", err)
	}

	return bid, nil
}

func (b *PebbleStoreBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	storeKey := []byte(formatBidValueKey(bid.Id, key))
	return bid, b.db.Set(storeKey, value, nil)
}

func (b *PebbleStoreBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	log.Info("pebble pebble pebble!")
	storeKey := []byte(formatBidValueKey(bid.Id, key))
	data, closer, err := b.db.Get(storeKey)
	if err != nil {
		return nil, fmt.Errorf("could not fetch data for bid %x and key %s: %w", bid.Id, key, err)
	}
	ret := make([]byte, len(data))
	copy(ret, data)
	closer.Close()
	return ret, nil
}
