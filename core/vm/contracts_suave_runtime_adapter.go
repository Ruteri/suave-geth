// Code generated by suave/gen. DO NOT EDIT.
// Hash: 5f3ef0076fdd52927e99273a846eb0a3fc3523ac0ca908e8b14993c671c19071
package vm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/mitchellh/mapstructure"
)

var (
	errFailedToUnpackInput = fmt.Errorf("failed to decode input")
	errFailedToDecodeField = fmt.Errorf("failed to decode field")
	errFailedToPackOutput  = fmt.Errorf("failed to encode output")
)

type SuaveRuntime interface {
	buildEthBlock(blockArgs types.BuildBlockArgs, bidId types.BidId, namespace string) ([]byte, []byte, error)
	confidentialInputs() ([]byte, error)
	confidentialRequestData() ([]byte, error)
	confidentialStoreRetrieve(bidId types.BidId, key string) ([]byte, error)
	confidentialStoreStore(bidId types.BidId, key string, data1 []byte) error
	ethcall(contractAddr common.Address, input1 []byte) ([]byte, error)
	extractHint(bundleData []byte) ([]byte, error)
	fetchBids(cond uint64, namespace string) ([]types.Bid, error)
	fillMevShareBundle(bidId types.BidId) ([]byte, error)
	generateKeyPair() ([]byte, []byte, error)
	generatePseudoRandomBytes(numBytes uint64, domainSeparator []byte) ([]byte, error)
	newBid(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, bidType string) (types.Bid, error)
	signEthTransaction(txn []byte, chainId string, signingKey string) ([]byte, error)
	simulateBundle(bundleData []byte) (uint64, error)
	submitBundleJsonRPC(url string, method string, params []byte) ([]byte, error)
	submitEthBlockBidToRelay(relayUrl string, builderBid []byte) ([]byte, error)
}

var (
	buildEthBlockAddr             = common.HexToAddress("0x0000000000000000000000000000000042100001")
	confidentialInputsAddr        = common.HexToAddress("0x0000000000000000000000000000000042010001")
	confidentialRequestDataAddr   = common.HexToAddress("0x0000000000000000000000000000000000000007")
	confidentialStoreRetrieveAddr = common.HexToAddress("0x0000000000000000000000000000000042020001")
	confidentialStoreStoreAddr    = common.HexToAddress("0x0000000000000000000000000000000042020000")
	ethcallAddr                   = common.HexToAddress("0x0000000000000000000000000000000042100003")
	extractHintAddr               = common.HexToAddress("0x0000000000000000000000000000000042100037")
	fetchBidsAddr                 = common.HexToAddress("0x0000000000000000000000000000000042030001")
	fillMevShareBundleAddr        = common.HexToAddress("0x0000000000000000000000000000000043200001")
	generateKeyPairAddr           = common.HexToAddress("0x0000000000000000000000000000000040100003")
	generatePseudoRandomBytesAddr = common.HexToAddress("0x0000000000000000000000000000000040100000")
	newBidAddr                    = common.HexToAddress("0x0000000000000000000000000000000042030000")
	signEthTransactionAddr        = common.HexToAddress("0x0000000000000000000000000000000040100001")
	simulateBundleAddr            = common.HexToAddress("0x0000000000000000000000000000000042100000")
	submitBundleJsonRPCAddr       = common.HexToAddress("0x0000000000000000000000000000000043000001")
	submitEthBlockBidToRelayAddr  = common.HexToAddress("0x0000000000000000000000000000000042100002")
)

type SuaveRuntimeAdapter struct {
	impl SuaveRuntime
}

func (b *SuaveRuntimeAdapter) run(addr common.Address, input []byte) ([]byte, error) {
	switch addr {
	case buildEthBlockAddr:
		return b.buildEthBlock(input)

	case confidentialInputsAddr:
		return b.confidentialInputs(input)

	case confidentialRequestDataAddr:
		return b.confidentialRequestData(input)

	case confidentialStoreRetrieveAddr:
		return b.confidentialStoreRetrieve(input)

	case confidentialStoreStoreAddr:
		return b.confidentialStoreStore(input)

	case ethcallAddr:
		return b.ethcall(input)

	case extractHintAddr:
		return b.extractHint(input)

	case fetchBidsAddr:
		return b.fetchBids(input)

	case fillMevShareBundleAddr:
		return b.fillMevShareBundle(input)

	case generateKeyPairAddr:
		return b.generateKeyPair(input)

	case generatePseudoRandomBytesAddr:
		return b.generatePseudoRandomBytes(input)

	case newBidAddr:
		return b.newBid(input)

	case signEthTransactionAddr:
		return b.signEthTransaction(input)

	case simulateBundleAddr:
		return b.simulateBundle(input)

	case submitBundleJsonRPCAddr:
		return b.submitBundleJsonRPC(input)

	case submitEthBlockBidToRelayAddr:
		return b.submitEthBlockBidToRelay(input)

	default:
		return nil, fmt.Errorf("suave precompile not found for " + addr.String())
	}
}

func (b *SuaveRuntimeAdapter) buildEthBlock(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["buildEthBlock"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		blockArgs types.BuildBlockArgs
		bidId     types.BidId
		namespace string
	)

	if err = mapstructure.Decode(unpacked[0], &blockArgs); err != nil {
		err = errFailedToDecodeField
		return
	}

	if err = mapstructure.Decode(unpacked[1], &bidId); err != nil {
		err = errFailedToDecodeField
		return
	}

	namespace = unpacked[2].(string)

	var (
		output1 []byte
		output2 []byte
	)

	if output1, output2, err = b.impl.buildEthBlock(blockArgs, bidId, namespace); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["buildEthBlock"].Outputs.Pack(output1, output2)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) confidentialInputs(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["confidentialInputs"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var ()

	var (
		output1 []byte
	)

	if output1, err = b.impl.confidentialInputs(); err != nil {
		return
	}

	result = output1
	return result, nil

}

func (b *SuaveRuntimeAdapter) confidentialRequestData(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["confidentialRequestData"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var ()

	var (
		output1 []byte
	)

	if output1, err = b.impl.confidentialRequestData(); err != nil {
		return
	}

	result = output1
	return result, nil

}

func (b *SuaveRuntimeAdapter) confidentialStoreRetrieve(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["confidentialStoreRetrieve"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		bidId types.BidId
		key   string
	)

	if err = mapstructure.Decode(unpacked[0], &bidId); err != nil {
		err = errFailedToDecodeField
		return
	}

	key = unpacked[1].(string)

	var (
		output1 []byte
	)

	if output1, err = b.impl.confidentialStoreRetrieve(bidId, key); err != nil {
		return
	}

	result = output1
	return result, nil

}

func (b *SuaveRuntimeAdapter) confidentialStoreStore(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["confidentialStoreStore"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		bidId types.BidId
		key   string
		data1 []byte
	)

	if err = mapstructure.Decode(unpacked[0], &bidId); err != nil {
		err = errFailedToDecodeField
		return
	}

	key = unpacked[1].(string)
	data1 = unpacked[2].([]byte)

	var ()

	if err = b.impl.confidentialStoreStore(bidId, key, data1); err != nil {
		return
	}

	return nil, nil

}

func (b *SuaveRuntimeAdapter) ethcall(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["ethcall"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		contractAddr common.Address
		input1       []byte
	)

	contractAddr = unpacked[0].(common.Address)
	input1 = unpacked[1].([]byte)

	var (
		output1 []byte
	)

	if output1, err = b.impl.ethcall(contractAddr, input1); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["ethcall"].Outputs.Pack(output1)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) extractHint(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["extractHint"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		bundleData []byte
	)

	bundleData = unpacked[0].([]byte)

	var (
		output1 []byte
	)

	if output1, err = b.impl.extractHint(bundleData); err != nil {
		return
	}

	result = output1
	return result, nil

}

func (b *SuaveRuntimeAdapter) fetchBids(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["fetchBids"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		cond      uint64
		namespace string
	)

	cond = unpacked[0].(uint64)
	namespace = unpacked[1].(string)

	var (
		bid []types.Bid
	)

	if bid, err = b.impl.fetchBids(cond, namespace); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["fetchBids"].Outputs.Pack(bid)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) fillMevShareBundle(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["fillMevShareBundle"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		bidId types.BidId
	)

	if err = mapstructure.Decode(unpacked[0], &bidId); err != nil {
		err = errFailedToDecodeField
		return
	}

	var (
		encodedBundle []byte
	)

	if encodedBundle, err = b.impl.fillMevShareBundle(bidId); err != nil {
		return
	}

	result = encodedBundle
	return result, nil

}

func (b *SuaveRuntimeAdapter) generateKeyPair(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["generateKeyPair"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var ()

	var (
		privkey []byte
		pubkey  []byte
	)

	if privkey, pubkey, err = b.impl.generateKeyPair(); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["generateKeyPair"].Outputs.Pack(privkey, pubkey)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) generatePseudoRandomBytes(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["generatePseudoRandomBytes"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		numBytes        uint64
		domainSeparator []byte
	)

	numBytes = unpacked[0].(uint64)
	domainSeparator = unpacked[1].([]byte)

	var (
		output1 []byte
	)

	if output1, err = b.impl.generatePseudoRandomBytes(numBytes, domainSeparator); err != nil {
		return
	}

	result = output1
	return result, nil

}

func (b *SuaveRuntimeAdapter) newBid(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["newBid"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		decryptionCondition uint64
		allowedPeekers      []common.Address
		allowedStores       []common.Address
		bidType             string
	)

	decryptionCondition = unpacked[0].(uint64)
	allowedPeekers = unpacked[1].([]common.Address)
	allowedStores = unpacked[2].([]common.Address)
	bidType = unpacked[3].(string)

	var (
		bid types.Bid
	)

	if bid, err = b.impl.newBid(decryptionCondition, allowedPeekers, allowedStores, bidType); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["newBid"].Outputs.Pack(bid)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) signEthTransaction(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["signEthTransaction"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		txn        []byte
		chainId    string
		signingKey string
	)

	txn = unpacked[0].([]byte)
	chainId = unpacked[1].(string)
	signingKey = unpacked[2].(string)

	var (
		output1 []byte
	)

	if output1, err = b.impl.signEthTransaction(txn, chainId, signingKey); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["signEthTransaction"].Outputs.Pack(output1)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) simulateBundle(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["simulateBundle"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		bundleData []byte
	)

	bundleData = unpacked[0].([]byte)

	var (
		output1 uint64
	)

	if output1, err = b.impl.simulateBundle(bundleData); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["simulateBundle"].Outputs.Pack(output1)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) submitBundleJsonRPC(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["submitBundleJsonRPC"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		url    string
		method string
		params []byte
	)

	url = unpacked[0].(string)
	method = unpacked[1].(string)
	params = unpacked[2].([]byte)

	var (
		output1 []byte
	)

	if output1, err = b.impl.submitBundleJsonRPC(url, method, params); err != nil {
		return
	}

	result = output1
	return result, nil

}

func (b *SuaveRuntimeAdapter) submitEthBlockBidToRelay(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["submitEthBlockBidToRelay"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		relayUrl   string
		builderBid []byte
	)

	relayUrl = unpacked[0].(string)
	builderBid = unpacked[1].([]byte)

	var (
		output1 []byte
	)

	if output1, err = b.impl.submitEthBlockBidToRelay(relayUrl, builderBid); err != nil {
		return
	}

	result = output1
	return result, nil

}
