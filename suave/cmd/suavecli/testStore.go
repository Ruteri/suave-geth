package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/ethereum/go-ethereum/suave/e2e"
	"github.com/ethereum/go-ethereum/suave/sdk"
)

func cmdTestStore() {
	flagset := flag.NewFlagSet("testStore", flag.ExitOnError)

	var (
		suaveRpc                = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		executionNodeAddressHex = flagset.String("ex_node_addr", "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B", "wallet address of execution node")
		privKeyHex              = flagset.String("privkey", "", "private key as hex (for testing)")
		contractAddressFlag     = flagset.String("contract", "", "contract address to use (default: deploy new one)")
		verbosity               = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		privKey                 *ecdsa.PrivateKey
	)

	flagset.Parse(os.Args[2:])

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	privKey, err := crypto.HexToECDSA(*privKeyHex)
	RequireNoErrorf(err, "-nodekeyhex: %v", err)
	/* shush linter */ privKey.Public()

	if executionNodeAddressHex == nil || *executionNodeAddressHex == "" {
		utils.Fatalf("please provide ex_node_addr")
	}
	executionNodeAddress := common.HexToAddress(*executionNodeAddressHex)

	suaveClient, err := rpc.DialContext(context.TODO(), *suaveRpc)
	RequireNoErrorf(err, "could not connect to suave rpc: %v", err)

	suaveSdkClient := sdk.NewClient(suaveClient, privKey, executionNodeAddress)

	contract := e2e.NewArtifact("../../artifacts/bids.sol/StoreTestContract.json")

	var contractAddress *common.Address
	if *contractAddressFlag != "" {
		suaveContractAddress := common.HexToAddress(*contractAddressFlag)
		contractAddress = &suaveContractAddress
	} else {
		deploymentTxRes, err := sdk.DeployContract(contract.Code, suaveSdkClient)
		RequireNoErrorf(err, "could not send deployment tx: %v", err)
		deploymentReceipt, err := deploymentTxRes.Wait()

		RequireNoErrorf(err, "error waiting for deployment tx inclusion: %v", err)
		if deploymentReceipt.Status != 1 {
			jsonEncodedReceipt, _ := deploymentReceipt.MarshalJSON()
			utils.Fatalf("deployment not successful: %s", string(jsonEncodedReceipt))
		}

		contractAddress = &deploymentReceipt.ContractAddress
		fmt.Println("contract address: ", deploymentReceipt.ContractAddress.Hex())
	}

	sdkContract := sdk.GetContract(*contractAddress, contract.Abi, suaveSdkClient)

	confidentialDataBytes, err := contract.Abi.Methods["store"].Inputs.Pack([]byte("xxxx"))
	RequireNoErrorf(err, "could not pack bundle confidential data: %v", err)

	confidentialRequestTxRes, err := sdkContract.SendTransaction("storeConfidential", nil, confidentialDataBytes)
	RequireNoErrorf(err, "could not send bundle request: %v", err)
	receipt, err := confidentialRequestTxRes.Wait()
	RequireNoError(err)
	if receipt.Status != 1 {
		utils.Fatalf("Store failed!")
	}

	unpackedBidEvent, err := contract.Abi.Events["BidEvent"].Inputs.Unpack(receipt.Logs[0].Data)
	RequireNoError(err)

	bidId := unpackedBidEvent[0].([16]byte)

	for i := 0; i < 10; i++ {
		_, err = sdkContract.SendTransaction("fetch", []interface{}{bidId}, nil)
		if err == nil {
			utils.Fatalf("fetch did not revert")
		}

		parsedErr, err := parseRpcError(err)
		RequireNoError(err)
		fmt.Println(string(parsedErr))
	}
}

func parseRpcError(rpcErr error) ([]byte, error) {
	if rpcErr == nil {
		return nil, errors.New("no error")
	}

	if len(rpcErr.Error()) < len("execution reverted: 0x") {
		return nil, errors.New("too short")
	}
	fmt.Println(rpcErr.Error())
	decodedError, err := hexutil.Decode(rpcErr.Error()[len("execution reverted: "):])
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}

	if len(decodedError) < 4 {
		return nil, errors.New("decoded too short")
	}

	unpacked, err := artifacts.SuaveAbi.Errors["PeekerReverted"].Inputs.Unpack(decodedError[4:])
	if err != nil {
		return nil, fmt.Errorf("could not unpack: %w", err)
	}
	fmt.Println(unpacked, string(decodedError))
	return unpacked[1].([]byte), nil
}
