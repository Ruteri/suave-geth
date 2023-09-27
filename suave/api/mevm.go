package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/gorilla/mux"
)

type ConfidentialRequestContext struct {
	RequestTx             hexutil.Bytes
	ConfidentialInputs    []byte
	Header                *types.Header
	StateAccountOverrides map[common.Address]Account
	StateStorageOverrides map[common.Address]Storage
}

type ConfidentialResult struct {
	Tx     hexutil.Bytes
	Result *core.ExecutionResult
}

type MEVMClient struct {
	endpoint string
	client   *http.Client
}

func NewMEVMClient(endpoint string) *MEVMClient {
	return &MEVMClient{
		endpoint: endpoint,
		client:   &http.Client{},
	}
}

func (c *MEVMClient) RunMEVM(ctx context.Context, _ *state.StateDB, header *types.Header, tx *types.Transaction, confInputs []byte) (*types.Transaction, *core.ExecutionResult, error) {
	requestTxBytes, err := tx.MarshalJSON()
	if err != nil {
		return nil, nil, err
	}
	reqContextBytes, err := json.Marshal(ConfidentialRequestContext{
		RequestTx:             requestTxBytes,
		ConfidentialInputs:    confInputs,
		Header:                header,
		StateAccountOverrides: nil,
		StateStorageOverrides: nil,
	})
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/confidential_request", c.endpoint), bytes.NewBuffer(reqContextBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("mevm client failed to execute: %s", string(body))
	}

	var resp ConfidentialResult
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, nil, err
	}

	var resTx types.Transaction
	err = resTx.UnmarshalJSON(resp.Tx)
	if err != nil {
		return nil, nil, err
	}

	return &resTx, resp.Result, nil
}

type MEVMApi struct {
	fetcher        *MissingStateFetcher
	accountManager *accounts.Manager
	suaveBackend   *vm.SuaveExecutionBackend
}

func NewMEVMApi(suaveBackend *vm.SuaveExecutionBackend, accountManager *accounts.Manager, missingStateEndpoint string) *MEVMApi {
	return &MEVMApi{
		fetcher:        NewMissingStateFetcher(missingStateEndpoint),
		accountManager: accountManager,
		suaveBackend:   suaveBackend,
	}
}

func (m *MEVMApi) ExecuteConfidentialRequest(reqContext ConfidentialRequestContext) (*types.Transaction, *core.ExecutionResult, error) {
	var requestTx types.Transaction
	err := requestTx.UnmarshalJSON(reqContext.RequestTx)
	if err != nil {
		return nil, nil, fmt.Errorf("mevm api: could not unmarshal request tx from req context: %w", err)
	}

	executionNodeAddress, err := suave.ExecutionNodeFromTransaction(&requestTx)
	if err != nil {
		return nil, nil, fmt.Errorf("mevm api: could not parse execution node address: %w", err)
	}

	wallet, err := m.accountManager.Find(accounts.Account{Address: executionNodeAddress})
	if err != nil {
		return nil, nil, fmt.Errorf("mevm api: execution node not available: %w", err)
	}

	signer := types.NewSuaveSigner(requestTx.ChainId())
	baseFee := big.NewInt(0)
	msg, err := core.TransactionToMessage(&requestTx, signer, baseFee)
	if err != nil {
		return nil, nil, fmt.Errorf("mevm api: could not format transction as message: %w", err)
	}

	statedb := NewExternalStateDB(m.fetcher, reqContext.Header.Hash(), reqContext.StateAccountOverrides, reqContext.StateStorageOverrides)

	vmConfig := &vm.Config{
		IsConfidential: true,
		NoBaseFee:      true,
	}
	blockCtx := newBlockContext(reqContext.Header, m.GetHashFn)
	suaveCtx := &vm.SuaveContext{
		Backend:                      m.suaveBackend,
		ConfidentialComputeRequestTx: &requestTx,
		ConfidentialInputs:           reqContext.ConfidentialInputs,
	}

	evm, vmError := GetEVM(msg, statedb, reqContext.Header, vmConfig, blockCtx, suaveCtx)
	defer evm.Cancel()

	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, nil, fmt.Errorf("mevm api: message apply failed: %w", err)
	}
	if err := vmError(); err != nil {
		return nil, nil, fmt.Errorf("mevm api: vm error: %w", err)
	}

	// Check for call in return
	var computeResult []byte

	args := abi.Arguments{abi.Argument{Type: abi.Type{T: abi.BytesTy}}}
	unpacked, err := args.Unpack(result.ReturnData)
	if err == nil && len(unpacked[0].([]byte))%32 == 4 {
		// This is supposed to be the case for all confidential compute!
		computeResult = unpacked[0].([]byte)
	} else {
		computeResult = result.ReturnData // Or should it be nil maybe in this case?
	}

	suaveResultTxData := &types.SuaveTransaction{ExecutionNode: executionNodeAddress, ConfidentialComputeRequest: requestTx, ConfidentialComputeResult: computeResult, ChainID: requestTx.ChainId()}

	signedResultTx, err := wallet.SignTx(accounts.Account{Address: executionNodeAddress}, types.NewTx(suaveResultTxData), requestTx.ChainId())
	if err != nil {
		return nil, result, fmt.Errorf("mevm api: could not sign result tx: %w", err)
	}

	// TODO: sign the tx with local account
	return signedResultTx, result, nil
}

func GetEVM(msg *core.Message, statedb vm.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext, suaveCtx *vm.SuaveContext) (*vm.EVM, func() error) {
	txContext := core.NewEVMTxContext(msg)
	return vm.NewConfidentialEVM(*suaveCtx, *blockCtx, txContext, statedb, params.SuaveChainConfig, *vmConfig), func() error { return nil }
}

func (m *MEVMApi) GetHashFn(n uint64) common.Hash {
	// h, err := m.fetcher.FetchHeader(n)
	//if err != nil {
	return common.Hash{}
	//}
}

func newBlockContext(header *types.Header, getHashFn func(uint64) common.Hash) *vm.BlockContext {
	return &vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     getHashFn,

		Coinbase:    common.Address{},
		GasLimit:    30000000,
		BlockNumber: new(big.Int).Add(big.NewInt(1), header.Number),
		Time:        header.Time + 1,
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(0),
		Random:      nil,
	}
}

type MEVMConfig struct {
	Host string
	Port int
	Api  *MEVMApi
}

type MEVMServer struct {
	cancel context.CancelFunc
	api    *MEVMApi
	srv    *http.Server
}

func NewMEVMServer(config MEVMConfig) *MEVMServer {
	mevmServer := &MEVMServer{
		api: config.Api,
	}

	r := mux.NewRouter()
	r.HandleFunc("/confidential_request", mevmServer.handleConfidentialRequest).Methods("POST")

	mevmServer.srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler: r,
	}

	return mevmServer
}

func (s *MEVMServer) handleConfidentialRequest(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	req.Body.Close()

	var confidentialRequestCtx ConfidentialRequestContext
	err = json.Unmarshal(body, &confidentialRequestCtx)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not unmarshal request: %s", err.Error()), http.StatusBadRequest)
		return
	}

	tx, result, err := s.api.ExecuteConfidentialRequest(confidentialRequestCtx)
	if err != nil {
		http.Error(w, fmt.Sprintf("execution failed: %s", err.Error()), http.StatusBadRequest)
		return
	}

	resTxBytes, err := tx.MarshalJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not marshal response tx: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := ConfidentialResult{
		Tx:     resTxBytes,
		Result: result,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("could encode result: %s", err.Error()), http.StatusBadRequest)
		return
	}
}

func (s *MEVMServer) Start() error {
	ctx, cancel := context.WithCancel(context.TODO())
	s.cancel = cancel

	log.Info("Starting MEVM server on", "addr", s.srv.Addr)

	go func() {
		<-ctx.Done()
		s.srv.Close()
	}()

	go s.srv.ListenAndServe()

	return nil
}

func (s *MEVMServer) Stop() error {
	log.Info("Stopping MEVM server on", "addr", s.srv.Addr)

	s.cancel()
	return nil
}
