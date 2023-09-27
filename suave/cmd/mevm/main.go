package main

import (
	"flag"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/suave/api"
	suave_backends "github.com/ethereum/go-ethereum/suave/backends"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

func main() {
	var (
		host                   = flag.String("host", "0.0.0.0", "host to listen on")
		port                   = flag.Int("port", 8153, "port to listen on")
		stateEndpoint          = flag.String("suave.state.endpoint", "", "Endpoint to connect to for missing chain state")
		redisStoreEndpoint     = flag.String("suave.confidential.redis-store-endpoint", "", "Redis endpoint to connect to for confidential storage backend (default: pebble)")
		remoteStoreEndpoint    = flag.String("suave.confidential.remote-store-endpoint", "", "Remote store backend to connect to (default: pebble)")
		pebbleDbPath           = flag.String("suave.confidential.pebble-store-db-path", "pebble-dev", "Path to pebble db to use for confidential storage backend")
		redisTransportEndpoint = flag.String("suave.confidential.redis-transport-endpoint", "", "Redis endpoint to connect to for confidential store transport (default: mock)")
		remoteEthEndpoint      = flag.String("suave.eth.endpoint", "", "Eth endpoint to connect to (default: mock)")
		verbosity              = flag.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
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
	} else if *remoteStoreEndpoint != "" {
		confidentialStoreBackend = suave_backends.NewRemoteStoreBackend(*remoteStoreEndpoint)
	} else if *pebbleDbPath != "" {
		confidentialStoreBackend = suave_backends.NewPebbleStoreBackend(*pebbleDbPath)
	} else {
		confidentialStoreBackend = suave_backends.NewLocalConfidentialStore()
	}

	var transportTopic suave.StoreTransportTopic
	if *redisTransportEndpoint != "" {
		transportTopic = suave_backends.NewRedisPubSubTransport(*redisTransportEndpoint)
	} else {
		transportTopic = suave.MockTransport{}
	}

	var mempool suave.MempoolBackend = suave_backends.NewMempoolOnConfidentialStore(suave_backends.NewLocalConfidentialStore())

	confidentialStoreEngine, err := suave.NewConfidentialStoreEngine(confidentialStoreBackend, transportTopic, mempool, suave.MockSigner{}, suave.MockChainSigner{})
	if err != nil {
		log.Error("could not instantiate engine", "err", err)
		return
	}

	confidentialStoreEngine.Start()
	defer confidentialStoreEngine.Stop()

	var confidentialEthBackend suave.ConfidentialEthBackend
	if *remoteEthEndpoint != "" {
		confidentialEthBackend = suave_backends.NewRemoteEthBackend(*remoteEthEndpoint)
	} else {
		confidentialEthBackend = &suave_backends.EthMock{}
	}

	var accountManager *accounts.Manager
	c := api.MEVMConfig{
		Host: *host,
		Port: *port,
		Api: api.NewMEVMApi(&vm.SuaveExecutionBackend{
			ConfidentialStoreEngine: confidentialStoreEngine,
			MempoolBackend:          mempool,
			ConfidentialEthBackend:  confidentialEthBackend,
		}, accountManager, *stateEndpoint),
	}

	mevmServer := api.NewMEVMServer(c)
	mevmServer.Start()
	// what do?
}
