package aerospike

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const maxPort = 65535

type Config struct {
	Host      string       `json:"host"`
	Port      int          `json:"port"`
	Namespace string       `json:"namespace"`
	Set       string       `json:"set"`
	Policy    ClientPolicy `json:"client_policy"`
}

type ClientPolicy struct {
	ConnectionQueueSize   int `json:"connection_queue_size"`
	MinConnectionsPerNode int `json:"min_connections_per_node"`
	ConnectTimeoutMs      int `json:"connect_timeout_ms"`
	IdleTimeoutMs         int `json:"idle_timeout_ms"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Host, validation.Required, is.Host),
		validation.Field(&c.Port, validation.Required, validation.Min(1), validation.Max(maxPort)),
		validation.Field(&c.Namespace, validation.Required),
		validation.Field(&c.Set, validation.Required),
		validation.Field(&c.Policy),
	)
}

func (p ClientPolicy) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.ConnectionQueueSize, validation.Min(0)),
		validation.Field(&p.MinConnectionsPerNode, validation.Min(0)),
		validation.Field(&p.ConnectTimeoutMs, validation.Min(0)),
		validation.Field(&p.IdleTimeoutMs, validation.Min(0)),
	)
}
