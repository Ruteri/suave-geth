package api

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type Account struct {
	Nonce    uint64
	Code     []byte
	CodeHash common.Hash
	Balance  *big.Int
}

type Storage struct {
	data map[common.Hash]common.Hash
}

type ExternalStateDB struct {
	lock  sync.Mutex
	accs  map[common.Address]*Account
	state map[common.Address]*Storage
	logs  []*types.Log

	headerHash common.Hash

	fetcher     *MissingStateFetcher
	fetchMisses map[common.Address]struct{}
}

func NewExternalStateDB(fetcher *MissingStateFetcher, headerHash common.Hash, stateAccountOverrides map[common.Address]Account, stateStorageOverrides map[common.Address]Storage) *ExternalStateDB {
	db := &ExternalStateDB{
		accs:        make(map[common.Address]*Account),
		state:       make(map[common.Address]*Storage),
		headerHash:  headerHash,
		fetcher:     fetcher,
		fetchMisses: make(map[common.Address]struct{}),
	}

	for addr, account := range stateAccountOverrides {
		account := account
		db.accs[addr] = &account
	}

	for addr, storage := range stateStorageOverrides {
		storage := storage
		db.state[addr] = &storage
	}

	return db
}

func (s *ExternalStateDB) CreateAccount(addr common.Address) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.accs[addr] = &Account{}
}

func (s *ExternalStateDB) SubBalance(addr common.Address, amt *big.Int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if acc, ok := s.accs[addr]; ok {
		if acc.Balance == nil {
			acc.Balance = new(big.Int)
		}
		acc.Balance.Sub(acc.Balance, amt)
	}
}
func (s *ExternalStateDB) AddBalance(addr common.Address, amt *big.Int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if acc, ok := s.accs[addr]; ok {
		if acc.Balance == nil {
			acc.Balance = new(big.Int)
		}
		acc.Balance.Add(acc.Balance, amt)
	}
}

func (s *ExternalStateDB) locked_getStateObjects(addr common.Address) (*Account, *Storage) {
	// Assumes s.lock is held

	acc, accOk := s.accs[addr]
	storage, storageOk := s.state[addr]

	if accOk || storageOk {
		return acc, storage
	}

	// Fetch and insert

	if _, missedAlready := s.fetchMisses[addr]; missedAlready {
		return nil, nil
	}

	s.lock.Unlock()
	fetchedAcctPtr, fetchedStoragePtr, err := s.fetcher.FetchStateObject(s.headerHash, addr)
	s.lock.Lock()

	if err != nil {
		log.Warn("external state db: could not fetch state object", "addr", addr, "err", err)
		return nil, nil
	}

	// Fetched in the meanwhile
	if _, missedAlready := s.fetchMisses[addr]; missedAlready {
		acc, accOk = s.accs[addr]
		storage, storageOk = s.state[addr]

		var accPtr *Account

		if accOk {
			accPtr = acc
		}

		var storagePtr *Storage

		if storageOk {
			storagePtr = storage
		}
		return accPtr, storagePtr
	}

	s.fetchMisses[addr] = struct{}{}
	if fetchedAcctPtr != nil {
		s.accs[addr] = fetchedAcctPtr
	}
	if fetchedStoragePtr != nil {
		s.state[addr] = fetchedStoragePtr
	}
	return fetchedAcctPtr, fetchedStoragePtr
}

func (s *ExternalStateDB) GetBalance(addr common.Address) *big.Int {
	s.lock.Lock()
	defer s.lock.Unlock()

	balance := new(big.Int)

	acc, _ := s.locked_getStateObjects(addr)
	if acc != nil && acc.Balance != nil {
		balance.Set(acc.Balance)
	}

	return balance
}

func (s *ExternalStateDB) GetNonce(addr common.Address) uint64 {
	s.lock.Lock()
	defer s.lock.Unlock()

	acc, _ := s.locked_getStateObjects(addr)
	if acc != nil {
		return acc.Nonce
	}
	return 0
}

func (s *ExternalStateDB) SetNonce(addr common.Address, nonce uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	acc, _ := s.locked_getStateObjects(addr)
	if acc != nil {
		acc.Nonce = nonce
	}
}

func (s *ExternalStateDB) GetCodeHash(addr common.Address) common.Hash {
	s.lock.Lock()
	defer s.lock.Unlock()

	acc, _ := s.locked_getStateObjects(addr)
	if acc != nil {
		return acc.CodeHash
	}
	return common.Hash{}
}
func (s *ExternalStateDB) GetCode(addr common.Address) []byte {
	s.lock.Lock()
	defer s.lock.Unlock()

	acc, _ := s.locked_getStateObjects(addr)
	if acc != nil {
		ret := make([]byte, len(acc.Code))
		copy(ret, acc.Code)
		return ret
	}
	return nil
}
func (s *ExternalStateDB) SetCode(addr common.Address, code []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()

	acc, _ := s.locked_getStateObjects(addr)
	if acc != nil {
		copy(acc.Code, code)
	}
}
func (s *ExternalStateDB) GetCodeSize(addr common.Address) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	acc, _ := s.locked_getStateObjects(addr)
	if acc != nil {
		return len(acc.Code)
	}
	return 0
}

/* NO REFUNDS */
func (s *ExternalStateDB) AddRefund(uint64)  {}
func (s *ExternalStateDB) SubRefund(uint64)  {}
func (s *ExternalStateDB) GetRefund() uint64 { return 0 }

func (s *ExternalStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	// FIXME
	return s.GetState(addr, key)
}
func (s *ExternalStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, st := s.locked_getStateObjects(addr)
	if st != nil {
		return st.data[key]
	}
	return common.Hash{}
}
func (s *ExternalStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, st := s.locked_getStateObjects(addr)
	if st == nil {
		st = &Storage{
			data: make(map[common.Hash]common.Hash),
		}
		s.state[addr] = st
	}

	st.data[key] = value
}

/* NO TRANSIENT STORAGE (for now) */
func (s *ExternalStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash{}
}
func (s *ExternalStateDB) SetTransientState(addr common.Address, key, value common.Hash) {}

/* No suicides */
func (s *ExternalStateDB) Suicide(common.Address) bool     { return false }
func (s *ExternalStateDB) HasSuicided(common.Address) bool { return false }

func (s *ExternalStateDB) Exist(addr common.Address) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	acc, st := s.locked_getStateObjects(addr)
	return acc != nil || st != nil
}
func (s *ExternalStateDB) Empty(addr common.Address) bool {
	return !s.Exist(addr) // we don't initialize empty!
}

/* NO 2929 (for now) */
func (s *ExternalStateDB) AddressInAccessList(addr common.Address) bool { return false }
func (s *ExternalStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	addressOk = false
	slotOk = false
	return
}
func (s *ExternalStateDB) AddAddressToAccessList(addr common.Address)                {}
func (s *ExternalStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {}
func (s *ExternalStateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// TODO
	// - prepare accessList(post-berlin)
	// - reset transient storage(eip 1153)
	// - set tx context for logs etc
}

/* Not supported (yet, maybe we should) */
func (s *ExternalStateDB) RevertToSnapshot(int) {}
func (s *ExternalStateDB) Snapshot() int {
	return 0
}

func (s *ExternalStateDB) AddLog(log *types.Log) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.logs = append(s.logs, log)
}
func (s *ExternalStateDB) AddPreimage(hash common.Hash, data []byte) {
}
