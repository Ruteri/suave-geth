package suave

type Config struct {
	RemoteMEVMRunnerEndpoint      string
	SuaveEthRemoteBackendEndpoint string
	RedisStorePubsubUri           string
	RemoteStoreEndpoint           string
	RedisStoreUri                 string
	PebbleDbPath                  string

	MissingStateEndpoint string

	ServeMissingState      bool
	MissingStateServerHost string
	MissingStateServerPort int

	RunMEVMServer  bool
	MEVMServerHost string
	MEVMServerPort int
}

var DefaultConfig = Config{
	ServeMissingState:      false,
	MissingStateServerHost: "127.0.0.1",
	MissingStateServerPort: 8164,

	RunMEVMServer:  false,
	MEVMServerHost: "127.0.0.1",
	MEVMServerPort: 8165,
}
