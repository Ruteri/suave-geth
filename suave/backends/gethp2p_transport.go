package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/google/uuid"
)

type P2PTransport struct {
	ctx    context.Context
	cancel context.CancelFunc

	lock        sync.Mutex
	subscribers map[uuid.UUID]func(suave.DAMessage) error

	msgCh chan transportMessage

	peers *Peers
}

type transportMessage struct {
	msg  suave.DAMessage
	peer *Peer
}

func NewP2PTransport() *P2PTransport {
	return &P2PTransport{
		subscribers: make(map[uuid.UUID]func(suave.DAMessage) error),
		msgCh:       make(chan transportMessage, 16),

		peers: NewPeers(),
	}
}

func (t *P2PTransport) MakeProtocols(computorAddresses []common.Address, network uint64, dnsdisc enode.Iterator) []p2p.Protocol {
	protocol := p2p.Protocol{
		Name:    "suave",
		Version: SUAVE1,
		Length:  17, // FIXME
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			log.Info("p2p transport: new peer", "peer", p.Info())
			peer := NewPeer(SUAVE1, p, rw, network, forkid.ID{})
			defer peer.Close()
			return t.RunProtocol(peer)
		},
		NodeInfo: func() interface{} {
			return NodeInfo{network, computorAddresses}
		},
		/* PeerInfo: func(id enode.ID) interface{} {
			return backend.PeerInfo(id)
		}, */
		Attributes:     []enr.Entry{},
		DialCandidates: dnsdisc,
	}
	return []p2p.Protocol{protocol}
}

func (t *P2PTransport) Start() error {
	if t.cancel != nil {
		return errors.New("p2p transport already started!")
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.ctx = ctx
	t.cancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("p2p transport: context closed", "err", ctx.Err())
				return
			case msg := <-t.msgCh:
				if err := t.handleStoreMsg(msg.msg, msg.peer); err != nil {
					log.Info("p2p transport: handleStoreMsg", "err", err)
				}
			}
		}
	}()

	return nil
}

func (t *P2PTransport) Stop() error {
	t.cancel()
	return nil
}

func (t *P2PTransport) Subscribe(cb func(suave.DAMessage) error) context.CancelFunc {
	ctx, cancel := context.WithCancel(t.ctx)

	t.lock.Lock()
	defer t.lock.Unlock()

	id := uuid.New()
	t.subscribers[id] = cb

	go func() {
		<-ctx.Done()
		cancel()

		t.lock.Lock()
		defer t.lock.Unlock()

		delete(t.subscribers, id)
	}()

	return cancel
}

func (t *P2PTransport) Publish(msg suave.DAMessage) {
	log.Info("p2p transport: Publish", "msg", msg)
	// Peers without a key
	t.peers.Broadcast(msg)
}

func (t *P2PTransport) handleStoreMsg(msg suave.DAMessage, peer *Peer) error {
	log.Info("p2p transport: handleStoreMsg", "msg", msg)

	// We don't care much about this lock, since this function is called synchronously
	// TODO: don't lock here at all ideally
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, cb := range t.subscribers {
		if err := cb(msg); err != nil {
			log.Error("p2p transport: disconnecting peer due to failure during message processing", "err", err, "peer", peer.Info())
			peer.Disconnect(p2p.DiscProtocolError)

			// Do not process further
			return err
		}
	}

	return nil
}

const (
	SUAVE1 = 1
)

func (p *Peer) Close() {
}

func (t *P2PTransport) RunProtocol(peer *Peer) error {
	log.Info("p2p transport: handshaking")
	if err := peer.Handshake(); err != nil {
		log.Error("p2p transport: Ethereum handshake failed", "err", err)
		return err
	}

	if err := t.peers.registerPeer(peer); err != nil {
		return err
	}

	defer t.peers.unregisterPeer(peer.id)

	// Initial sync goes here

	for {
		if err := handleMessage(t, peer); err != nil {
			log.Error("p2p transport: Message handling failed in `eth`", "err", err)
			return err
		}
	}
}

type NodeInfo struct {
	Network           uint64           `json:"network"` // Suave network ID
	ComputorAddresses []common.Address `json:"computorAddresses"`
}

var handlers = map[uint64]func(t *P2PTransport, msg p2p.Msg, peer *Peer) error{
	StoreMsg: handleStoreMsg,
}

func handleMessage(t *P2PTransport, peer *Peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := peer.rw.ReadMsg()
	if err != nil {
		return err
	}
	log.Info("p2p transport: handleMessage", "msg", msg, "peer", *peer)
	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	defer msg.Discard()

	if handler := handlers[msg.Code]; handler != nil {
		return handler(t, msg, peer)
	}
	return fmt.Errorf("%w: %v", errInvalidMsgCode, msg.Code)
}

func handleStoreMsg(t *P2PTransport, msg p2p.Msg, peer *Peer) error {
	// TODO: RLP-encode the DAMessage, doesn't work straight away

	var daMsgBytes []byte
	if err := msg.Decode(&daMsgBytes); err != nil {
		log.Error("p2p transport: handleStoreMsg", "err", err)
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}

	var daMsg suave.DAMessage
	if err := json.Unmarshal(daMsgBytes, &daMsg); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}

	select {
	case t.msgCh <- transportMessage{daMsg, peer}:
		return nil
	default:
		log.Warn("p2p transport: dropping messages due to channel being full")
		return nil
	}
}

type Peer struct {
	id        string
	*p2p.Peer                   // The embedded P2P package peer
	rw        p2p.MsgReadWriter // Input/output streams for snap

	fork    forkid.ID
	version uint32
	network uint64

	handler *Peers

	knownKeys map[string]struct{}

	bidAnnounce      chan []suave.Bid
	storeKeyAnnounce chan []storeKey
}

func NewPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, network uint64, fork forkid.ID) *Peer {
	return &Peer{
		id:        p.ID().String(),
		Peer:      p,
		rw:        rw,
		fork:      fork,
		version:   uint32(version),
		network:   network,
		knownKeys: make(map[string]struct{}),
	}
}

type storeKey struct {
	BidId suave.BidId
	key   string
}

func (p *Peer) Handshake() error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)

	var status StatusPacket // safe to read after two values have been received from errc

	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, &StatusPacket{
			ProtocolVersion: uint32(SUAVE1),
			NetworkID:       p.network,
			ForkID:          p.fork,
		})
	}()
	go func() {
		errc <- p.readStatus(&status)
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				log.Error("p2p transport peer: error while status checking", "err", err)
				return err
			}
		case <-timeout.C:
			log.Error("p2p transport peer: error while status checking: timeout")
			return p2p.DiscReadTimeout
		}
	}

	return nil
}

type StatusPacket struct {
	ProtocolVersion uint32
	NetworkID       uint64
	ForkID          forkid.ID
}

const (
	StatusMsg = 0x00
	StoreMsg  = 0x01
)

func (p *Peer) readStatus(status *StatusPacket) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return fmt.Errorf("%w: first msg has code %x (!= %x)", errNoStatusMsg, msg.Code, StatusMsg)
	}
	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	if status.NetworkID != p.network {
		return fmt.Errorf("%w: %d (!= %d)", errNetworkIDMismatch, status.NetworkID, p.network)
	}
	if status.ProtocolVersion != p.version {
		return fmt.Errorf("%w: %d (!= %d)", errProtocolVersionMismatch, status.ProtocolVersion, p.version)
	}
	return nil
}

type Peers struct {
	lock  sync.Mutex
	peers map[string]*Peer
}

func NewPeers() *Peers {
	return &Peers{
		peers: make(map[string]*Peer),
	}
}

func (h *Peers) registerPeer(p *Peer) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	_, found := h.peers[p.id]
	if found {
		return errors.New("peer already known")
	}

	h.peers[p.id] = p
	return nil
}

func (h *Peers) unregisterPeer(id string) {
	h.lock.Lock()
	defer h.lock.Unlock()
	delete(h.peers, id)
}

func (h *Peers) Broadcast(msg suave.DAMessage) {
	h.lock.Lock()
	defer h.lock.Unlock()

	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Error("p2p transport: could not marshal DA message", "err", err)
		return
	}

	for _, peer := range h.peers {
		go func(peer *Peer) {
			log.Info("p2p transport peers: broadcasting msg to", "peer", *peer)

			if err := p2p.Send(peer.rw, StoreMsg, msgData); err != nil {
				log.Warn("p2p transport peers: could not send message", "err", err)
			}
		}(peer)
	}
}

var (
	errNoStatusMsg             = errors.New("no status message")
	errMsgTooLarge             = errors.New("message too long")
	errDecode                  = errors.New("invalid message")
	errInvalidMsgCode          = errors.New("invalid message code")
	errProtocolVersionMismatch = errors.New("protocol version mismatch")
	errNetworkIDMismatch       = errors.New("network ID mismatch")
	errGenesisMismatch         = errors.New("genesis mismatch")
	errForkIDRejected          = errors.New("fork ID rejected")
)

// maxMessageSize is the maximum cap on the size of a protocol message.
const maxMessageSize = 10 * 1024 * 1024

const (
	// handshakeTimeout is the maximum allowed time for the `eth` handshake to
	// complete before dropping the connection.= as malicious.
	handshakeTimeout = 5 * time.Second
)
