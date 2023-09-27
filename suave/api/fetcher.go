package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"
)

type MissingStateFetcher struct {
	endpoint string
	client   *http.Client
}

func NewMissingStateFetcher(endpoint string) *MissingStateFetcher {
	return &MissingStateFetcher{
		endpoint: endpoint,
		client:   &http.Client{},
	}
}

// TODO: should synchronize concurrent calls for the same addr
func (f *MissingStateFetcher) FetchStateObject(root common.Hash, addr common.Address) (*Account, *Storage, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/state/%s/%s", f.endpoint, root.Hex(), addr.Hex()), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := f.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to fetch state object: %s", string(body))
	}

	var resp FetcherResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, nil, err
	}

	return resp.Account, resp.Storage, nil
}

type MissingStateServer struct {
	cancel context.CancelFunc

	host    string
	port    int
	backend ETHAPIBackend
}

type ETHAPIBackend interface {
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
}

func NewMissingStateServer(host string, port int, backend ETHAPIBackend) *MissingStateServer {
	return &MissingStateServer{
		host:    host,
		port:    port,
		backend: backend,
	}
}

func (mss *MissingStateServer) Stop() error {
	if mss.cancel != nil {
		mss.cancel()
	}

	return nil
}

func (mss *MissingStateServer) Start() error {
	mss.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	mss.cancel = cancel

	r := mux.NewRouter()
	r.HandleFunc("/state/{hash}/{addr}", mss.handleGetState).Methods("GET")

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", mss.host, mss.port),
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	go srv.ListenAndServe()

	return nil
}

func (mss *MissingStateServer) handleGetState(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	hashStr, ok := vars["hash"]
	if !ok {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	addrStr, ok := vars["addr"]
	if !ok {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	addr := common.HexToAddress(addrStr)

	headerHash := common.HexToHash(hashStr)
	blockNumberOrHash := rpc.BlockNumberOrHash{BlockHash: &headerHash, RequireCanonical: false}

	{
		stateDB, header, _ := mss.backend.StateAndHeaderByNumberOrHash(context.TODO(), rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
		log.Info("Most recent state", "balance", stateDB.GetBalance(addr), "hash", header.Hash(), "requested", *blockNumberOrHash.BlockHash)
	}

	stateDB, _, err := mss.backend.StateAndHeaderByNumberOrHash(context.TODO(), blockNumberOrHash)
	if err != nil {
		http.Error(w, fmt.Sprintf("state missing for hash %s: %s", hashStr, err.Error()), http.StatusBadRequest)
		return
	}

	nonce := stateDB.GetNonce(addr)
	code := stateDB.GetCode(addr)
	codeHash := stateDB.GetCodeHash(addr)
	balance := new(big.Int).Set(stateDB.GetBalance(addr))

	storageData := make(map[common.Hash]common.Hash)
	stateDB.ForEachStorage(addr, func(key, value common.Hash) bool {
		storageData[key] = value
		return true
	})

	resp := FetcherResponse{
		Account: &Account{
			Nonce:    nonce,
			Code:     code,
			CodeHash: codeHash,
			Balance:  balance,
		},
		Storage: &Storage{
			data: storageData,
		},
	}

	log.Info("State server response", "addr", addr, "acc", *resp.Account, "storage", *resp.Storage)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("could encode state response for address %x: %s", addr, err.Error()), http.StatusBadRequest)
		return
	}
}

type FetcherResponse struct {
	Account *Account
	Storage *Storage
}
