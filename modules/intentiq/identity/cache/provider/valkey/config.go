package valkey

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	maxPort                 = 65535
	defaultConnectTimeoutMs = 5_000
)

type Config struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Password         string `json:"password"`
	ConnectTimeoutMs int    `json:"connect_timeout_ms"`
}

func (c Config) ConnectTimeout() time.Duration {
	if c.ConnectTimeoutMs <= 0 {
		return defaultConnectTimeoutMs * time.Millisecond
	}
	return time.Duration(c.ConnectTimeoutMs) * time.Millisecond
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Host, validation.Required, is.Host),
		validation.Field(&c.Port, validation.Required, validation.Min(1), validation.Max(maxPort)),
		validation.Field(&c.ConnectTimeoutMs, validation.Min(0)),
	)
}
