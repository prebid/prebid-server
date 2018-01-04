package config

type InMemoryCache struct {
	// TTL of the cache
	TTL int64
	// Size of the cache before we evict objects
	Size int64
}

type RedisCache struct {
	// TTL of the cache
	TTL int64
	// The network type, either tcp or unix.
	// Default is tcp.
	Network string
	// host:port address.
	Addr string

	// Optional password. Must match the password specified in the
	// requirepass server configuration option.
	Password string
	// Database to be selected after connecting to the server.
	DB int
}
