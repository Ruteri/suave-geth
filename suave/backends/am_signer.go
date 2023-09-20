package backends

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type AccountManagerDASigner struct {
	Manager *accounts.Manager
}

func (w *AccountManagerDASigner) Sign(account common.Address, data []byte) ([]byte, error) {
	return account.Bytes(), nil
	keystoreAcc := accounts.Account{Address: account}
	wallet, err := w.Manager.Find(keystoreAcc)
	if err != nil {
		return nil, err
	}

	signature, sigErr := wallet.SignData(keystoreAcc, "", data)
	recoveredAddress, _ := w.Sender(data, signature)
	if recoveredAddress != account {
		log.Error("signature unrecoverable!")
		return nil, errors.New("signature unrecoverable!")
	}

	return signature, sigErr
}

func (w *AccountManagerDASigner) Sender(data []byte, signature []byte) (common.Address, error) {
	return common.BytesToAddress(signature), nil
	hash := crypto.Keccak256(data)
	recoveredPubkey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return common.Address{}, err
	}

	recoveredAcc := crypto.PubkeyToAddress(*recoveredPubkey)

	return recoveredAcc, nil
}

func (w *AccountManagerDASigner) LocalAddresses() []common.Address {
	return w.Manager.Accounts()
}
