package vm

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/ethereum/go-ethereum/suave/backends"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

type mockSuaveBackend struct {
}

func (m *mockSuaveBackend) Start() error { return nil }
func (m *mockSuaveBackend) Stop() error  { return nil }

func (m *mockSuaveBackend) InitializeBid(bid suave.Bid) error {
	return nil
}

func (m *mockSuaveBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockSuaveBackend) SubmitBid(types.Bid) error {
	return nil
}

func (m *mockSuaveBackend) FetchEngineBidById(suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) FetchBidById(suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	return nil
}

func (m *mockSuaveBackend) BuildEthBlock(ctx context.Context, args *suave.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (m *mockSuaveBackend) BuildEthBlockFromBundles(ctx context.Context, args *suave.BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (m *mockSuaveBackend) Subscribe() (<-chan suave.DAMessage, context.CancelFunc) {
	return nil, func() {}
}

func (m *mockSuaveBackend) Publish(suave.DAMessage) {}

var dummyBlockContext = BlockContext{
	CanTransfer: func(StateDB, common.Address, *big.Int) bool { return true },
	Transfer:    func(StateDB, common.Address, common.Address, *big.Int) {},
	BlockNumber: big.NewInt(0),
}

func TestSuavePrecompileStub(t *testing.T) {
	// This test ensures that the Suave precompile stubs work as expected
	// for encoding/decoding.
	mockSuaveBackend := &mockSuaveBackend{}
	stubEngine, err := suave.NewConfidentialStoreEngine(mockSuaveBackend, mockSuaveBackend, suave.MockSigner{}, suave.MockChainSigner{})
	require.NoError(t, err)

	suaveContext := SuaveContext{
		Backend: &SuaveExecutionBackend{
			ConfidentialStoreEngine: stubEngine,
			ConfidentialEthBackend:  mockSuaveBackend,
		},
		ConfidentialComputeRequestTx: types.NewTx(&types.ConfidentialComputeRequest{
			ExecutionNode: common.Address{},
			Wrapped:       *types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
		}),
	}

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	vmenv := NewConfidentialEVM(suaveContext, dummyBlockContext, TxContext{}, statedb, params.AllEthashProtocolChanges, Config{IsConfidential: true})

	// The objective of the unit test is to make sure that the encoding of the precompile
	// inputs works as expected from the ABI specification. Thus, we will skip any errors
	// that are generated by the logic of the precompile.
	// Note: Once code generated is in place, we can remove this and only test the
	// encodings in isolation outside the logic.
	expectedErrors := []string{
		// json error when the precompile expects to decode a json object encoded as []byte
		// in the precompile input.
		"invalid character",
		"not allowed to store",
		"not allowed to retrieve",
		"unknown bid version",
		// error from a precompile that expects to make an http request from an input value.
		"could not send request to relay",
		// error in 'buildEthBlock' when it expects to retrieve bids in abi format from the
		// confidential store.
		"could not unpack merged bid ids",
	}

	for name, addr := range artifacts.SuaveMethods {
		abiMethod, ok := artifacts.SuaveAbi.Methods[name]
		if !ok {
			t.Fatalf("abi method '%s' not found", name)
		}

		inputVals := abi.GenerateRandomTypeForMethod(abiMethod)

		packedInput, err := abiMethod.Inputs.Pack(inputVals...)
		require.NoError(t, err)

		_, _, err = vmenv.Call(AccountRef(common.Address{}), addr, packedInput, 100000000, big.NewInt(0))
		if err != nil {
			found := false
			for _, expectedError := range expectedErrors {
				if strings.Contains(err.Error(), expectedError) {
					found = true
				}
			}
			if !found {
				t.Fatal(err)
			}
		}
	}
}

func newTestBackend(t *testing.T) *suaveRuntime {
	confStore := backends.NewLocalConfidentialStore()
	confEngine, err := suave.NewConfidentialStoreEngine(confStore, &suave.MockTransport{}, suave.MockSigner{}, suave.MockChainSigner{})
	require.NoError(t, err)

	require.NoError(t, confEngine.Start())
	t.Cleanup(func() { confEngine.Stop() })

	b := &suaveRuntime{
		suaveContext: &SuaveContext{
			Backend: &SuaveExecutionBackend{
				ConfidentialStoreEngine: confEngine,
				ConfidentialEthBackend:  &mockSuaveBackend{},
			},
			ConfidentialComputeRequestTx: types.NewTx(&types.ConfidentialComputeRequest{
				ExecutionNode: common.Address{},
				Wrapped:       *types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
			}),
		},
	}
	return b
}

func TestSuave_BidWorkflow(t *testing.T) {
	b := newTestBackend(t)

	bid5, err := b.newBid(5, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	bid10, err := b.newBid(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	bid10b, err := b.newBid(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	cases := []struct {
		cond      uint64
		namespace string
		bids      []types.Bid
	}{
		{0, "a", []types.Bid{}},
		{5, "a", []types.Bid{bid5}},
		{10, "a", []types.Bid{bid10, bid10b}},
		{11, "a", []types.Bid{}},
	}

	for _, c := range cases {
		bids, err := b.fetchBids(c.cond, c.namespace)
		require.NoError(t, err)
		require.Equal(t, c.bids, bids)
	}
}

func TestSuave_ConfStoreWorkflow(t *testing.T) {
	b := newTestBackend(t)

	callerAddr := common.Address{0x1}
	data := []byte{0x1}

	// cannot store a value for a bid that does not exist
	err := b.confidentialStoreStore(types.BidId{}, "key", data)
	require.Error(t, err)

	bid, err := b.newBid(5, []common.Address{callerAddr}, nil, "a")
	require.NoError(t, err)

	// cannot store the bid if the caller is not allowed to
	err = b.confidentialStoreStore(bid.Id, "key", data)
	require.Error(t, err)

	// now, the caller is allowed to store the bid
	b.suaveContext.CallerStack = append(b.suaveContext.CallerStack, &callerAddr)
	err = b.confidentialStoreStore(bid.Id, "key", data)
	require.NoError(t, err)

	val, err := b.confidentialStoreRetrieve(bid.Id, "key")
	require.NoError(t, err)
	require.Equal(t, data, val)

	// cannot retrieve the value if the caller is not allowed to
	b.suaveContext.CallerStack = []*common.Address{}
	_, err = b.confidentialStoreRetrieve(bid.Id, "key")
	require.Error(t, err)
}
