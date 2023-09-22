package suave

type Config struct {
	SuaveEthRemoteBackendEndpoint string
	RedisStorePubsubUri           string
	RemoteStoreEndpoint           string
	RedisStoreUri                 string
	PebbleDbPath                  string
}

var DefaultConfig = Config{}
