package suave

type Config struct {
	SuaveEthRemoteBackendEndpoint string
	RedisStorePubsubUri           string
	RedisStoreUri                 string
	PebbleDbPath                  string
}

var DefaultConfig = Config{}
