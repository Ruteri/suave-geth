package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	suave_backends "github.com/ethereum/go-ethereum/suave/backends"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type Config struct {
	Host  string
	Port  int
	Store suave.ConfidentialStoreBackend
}

func main() {
	var (
		host               = flag.String("host", "0.0.0.0", "host to listen on")
		port               = flag.Int("port", 8153, "port to listen on")
		redisStoreEndpoint = flag.String("suave.confidential.redis-store-endpoint", "", "Redis endpoint to connect to for confidential storage backend")
		pebbleDbPath       = flag.String("suave.confidential.pebble-store-db-path", "pebble-dev", "Path to pebble db to use for confidential storage backend")
		verbosity          = flag.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
	)

	flag.Parse()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	var confidentialStoreBackend suave.ConfidentialStoreBackend
	if *redisStoreEndpoint == "dev" {
		confidentialStoreBackend = suave_backends.NewMiniredisBackend()
	} else if *redisStoreEndpoint != "" {
		confidentialStoreBackend = suave_backends.NewRedisStoreBackend(*redisStoreEndpoint)
	} else if *pebbleDbPath != "" {
		confidentialStoreBackend = suave_backends.NewPebbleStoreBackend(*pebbleDbPath)
	} else {
		confidentialStoreBackend = suave_backends.NewLocalConfidentialStore()
	}

	c := Config{
		Host:  *host,
		Port:  *port,
		Store: confidentialStoreBackend,
	}

	RunRemoteStore(context.Background(), c)
}

func RunRemoteStore(ctx context.Context, c Config) {
	c.Store.Start()
	defer c.Store.Stop()

	storeAdapter := newConfStoreHttpAdapter(c.Store)

	r := mux.NewRouter()
	r.HandleFunc("/bid", storeAdapter.handleBidInitialize).Methods("POST")
	r.HandleFunc("/bid/{id}", storeAdapter.handleBidById).Methods("GET")
	r.HandleFunc("/bid/{id}/{caller}/{key}", storeAdapter.handleStore).Methods("POST")
	r.HandleFunc("/bid/{id}/{caller}/{key}", storeAdapter.handleRetrieve).Methods("GET")

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", c.Host, c.Port),
		Handler: r,
	}
	srv.ListenAndServe()
}

type confStoreHttpAdapter struct {
	store suave.ConfidentialStoreBackend
}

func newConfStoreHttpAdapter(store suave.ConfidentialStoreBackend) *confStoreHttpAdapter {
	return &confStoreHttpAdapter{
		store: store,
	}
}

func (s *confStoreHttpAdapter) handleBidInitialize(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	req.Body.Close()

	var bid suave.Bid
	err = json.Unmarshal(body, &bid)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not unmarshal bid: %s", err.Error()), http.StatusBadRequest)
		return
	}

	err = s.store.InitializeBid(bid)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not unmarshal bid: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func (s *confStoreHttpAdapter) handleBidById(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, ok := vars["id"]
	if !ok {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	bidId, err := uuid.FromBytes(common.Hex2Bytes(id))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid id: %s", err.Error()), http.StatusBadRequest)
		return
	}

	bid, err := s.store.FetchEngineBidById(types.BidId(bidId))
	if err != nil {
		http.Error(w, fmt.Sprintf("could not fetch bid by id %x: %s", bidId, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(bid); err != nil {
		http.Error(w, fmt.Sprintf("could encode bid by id %x: %s", bidId, err.Error()), http.StatusBadRequest)
		return
	}
}

func (s *confStoreHttpAdapter) handleStore(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, ok := vars["id"]
	if !ok {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	callerHex, ok := vars["caller"]
	if !ok {
		http.Error(w, "missing caller", http.StatusBadRequest)
		return
	}

	caller := common.HexToAddress(callerHex)

	key, ok := vars["key"]
	if !ok {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	bidId, err := uuid.FromBytes(common.Hex2Bytes(id))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid id: %s", err.Error()), http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	req.Body.Close()

	bid, err := s.store.FetchEngineBidById(types.BidId(bidId))
	if err != nil {
		http.Error(w, fmt.Sprintf("could not fetch bid %x: %s", bidId, err.Error()), http.StatusBadRequest)
		return
	}

	_, err = s.store.Store(bid, caller, key, body)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not store %x/%x/%s: %s", bidId, caller, key, err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *confStoreHttpAdapter) handleRetrieve(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, ok := vars["id"]
	if !ok {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	callerHex, ok := vars["caller"]
	if !ok {
		http.Error(w, "missing caller", http.StatusBadRequest)
		return
	}

	caller := common.HexToAddress(callerHex)

	key, ok := vars["key"]
	if !ok {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	bidId, err := uuid.FromBytes(common.Hex2Bytes(id))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid id: %s", err.Error()), http.StatusBadRequest)
		return
	}

	bid, err := s.store.FetchEngineBidById(types.BidId(bidId))
	if err != nil {
		http.Error(w, fmt.Sprintf("could not fetch bid %x: %s", bidId, err.Error()), http.StatusBadRequest)
		return
	}

	data, err := s.store.Retrieve(bid, caller, key)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not retrieve data %x/%x/%s: %s", bidId, caller, key, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
