package e2e

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/suave/sdk"
	"github.com/stretchr/testify/require"
)

func WithRemoteMEVM(t *testing.T) (frameworkOpt, frameworkOpt) {
	memvPort := 18165
	return func(c *frameworkConfig) {
			c.suaveConfig.RemoteMEVMRunnerEndpoint = fmt.Sprintf("http://127.0.0.1:%d", memvPort)
		}, func(c *frameworkConfig) {
			c.suaveConfig.RunMEVMServer = true
			c.suaveConfig.MEVMServerPort = memvPort
		}
}

func WithMissingStateServer(t *testing.T) (frameworkOpt, frameworkOpt) {
	mstPort := 18164
	return func(c *frameworkConfig) {
			c.suaveConfig.MissingStateEndpoint = fmt.Sprintf("http://127.0.0.1:%d", mstPort)
		}, func(c *frameworkConfig) {
			c.suaveConfig.ServeMissingState = true
			c.suaveConfig.MissingStateServerPort = mstPort
		}
}

func WithKeystore(t *testing.T) frameworkOpt {
	keydir := t.TempDir()
	keystore := keystore.NewPlaintextKeyStore(keydir)
	acc, err := keystore.NewAccount("")
	require.NoError(t, err)
	require.NoError(t, keystore.TimedUnlock(acc, "", time.Hour))

	return func(c *frameworkConfig) {
		c.keystore = keystore
	}
}

func TestRemoteMEVM(t *testing.T) {
	remoteMevmClientOpt, remoteMevmServerOpt := WithRemoteMEVM(t)
	missingStateClientOpt, missingStateServerOpt := WithMissingStateServer(t)

	keystoreOpt := WithKeystore(t)

	fr := newFramework(t, keystoreOpt, remoteMevmClientOpt, missingStateServerOpt)
	defer fr.Close()
	rfr := newFramework(t, keystoreOpt, remoteMevmServerOpt, missingStateClientOpt)
	defer rfr.Close()

	time.Sleep(time.Second)

	clt := fr.NewSDKClient()
	// clt.SetExectionNodeAddress(testAddr)
	log.Info("xx", "ta", testAddr, "ex", fr.ExecutionNode())

	{
		targetBlock := uint64(16103213)
		allowedPeekers := []common.Address{{0x41, 0x42, 0x43}, newBundleBidAddress}

		bundle := struct {
			Txs             types.Transactions `json:"txs"`
			RevertingHashes []common.Hash      `json:"revertingHashes"`
		}{
			Txs:             types.Transactions{types.NewTx(&types.LegacyTx{})},
			RevertingHashes: []common.Hash{},
		}
		bundleBytes, err := json.Marshal(bundle)
		require.NoError(t, err)

		confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		bundleBidContractI := sdk.GetContract(newBundleBidAddress, BundleBidContract.Abi, clt)
		_, err = bundleBidContractI.SendTransaction("newBid", []interface{}{targetBlock, allowedPeekers, []common.Address{}}, confidentialDataBytes)
		requireNoRpcError(t, err)

		block := fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))

		receipts := block.Receipts
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.SuaveTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 1, len(block.Transactions()))
		unpacked, err := BundleBidContract.Abi.Methods["emitBid"].Inputs.Unpack(block.Transactions()[0].Data()[4:])
		require.NoError(t, err)
		bid := unpacked[0].(struct {
			Id                  [16]uint8        "json:\"id\""
			Salt                [16]uint8        "json:\"salt\""
			DecryptionCondition uint64           "json:\"decryptionCondition\""
			AllowedPeekers      []common.Address "json:\"allowedPeekers\""
			AllowedStores       []common.Address "json:\"allowedStores\""
			Version             string           "json:\"version\""
		})
		require.Equal(t, targetBlock, bid.DecryptionCondition)
		require.Equal(t, allowedPeekers, bid.AllowedPeekers)

		require.NotNil(t, receipts[0].Logs[0])
		require.Equal(t, newBundleBidAddress, receipts[0].Logs[0].Address)

		unpacked, err = BundleBidContract.Abi.Events["BidEvent"].Inputs.Unpack(receipts[0].Logs[0].Data)
		require.NoError(t, err)

		require.Equal(t, bid.Id, unpacked[0].([16]byte))
		require.Equal(t, bid.DecryptionCondition, unpacked[1].(uint64))
		require.Equal(t, bid.AllowedPeekers, unpacked[2].([]common.Address))

		_, err = rfr.SuaveBackend().ConfidentialStoreEngine.Retrieve(bid.Id, common.Address{0x41, 0x42, 0x43}, "default:v0:ethBundleSimResults")
		require.NoError(t, err)
	}
}
