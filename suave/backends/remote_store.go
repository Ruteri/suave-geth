package backends

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type RemoteStoreBackend struct {
	endpoint string
	client   *http.Client
}

func NewRemoteStoreBackend(endpoint string) *RemoteStoreBackend {
	return &RemoteStoreBackend{
		endpoint: endpoint,
		client:   http.DefaultClient,
	}
}

func (b *RemoteStoreBackend) Start() error {
	return nil
}

func (b *RemoteStoreBackend) Stop() error {
	return nil
}

func (b *RemoteStoreBackend) InitializeBid(bid suave.Bid) error {
	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(bid)
	req, err := http.NewRequest("POST", b.endpoint+"/bid", payloadBuf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(req.Body)
		return fmt.Errorf("failed to initialize bid: %s", string(body))
	}

	return nil
}

func (b *RemoteStoreBackend) FetchEngineBidById(bidId suave.BidId) (suave.Bid, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/bid/%s", b.endpoint, common.Bytes2Hex(bidId[:])), nil)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := b.client.Do(req)
	if err != nil {
		return suave.Bid{}, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return suave.Bid{}, err
	}
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return suave.Bid{}, fmt.Errorf("failed to initialize bid: %s", string(body))
	}

	var bid suave.Bid
	err = json.Unmarshal(body, &bid)
	if err != nil {
		return suave.Bid{}, err
	}

	return bid, nil
}

func (b *RemoteStoreBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	payloadBuf := new(bytes.Buffer)
	_, err := payloadBuf.Write(value)
	if err != nil {
		return suave.Bid{}, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/bid/%s/%x/%s", b.endpoint, common.Bytes2Hex(bid.Id[:]), caller.Bytes(), key), payloadBuf)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := b.client.Do(req)
	if err != nil {
		return suave.Bid{}, err
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return suave.Bid{}, err
	}
	req.Body.Close()

	if res.StatusCode != http.StatusOK {
		return suave.Bid{}, fmt.Errorf("failed to initialize bid: %s", string(body))
	}

	return bid, nil
}

func (b *RemoteStoreBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/bid/%s/%x/%s", b.endpoint, common.Bytes2Hex(bid.Id[:]), caller.Bytes(), key), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to initialize bid: %s", string(body))
	}
	return body, nil
}
