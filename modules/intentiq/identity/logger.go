package identity

import "github.com/prebid/prebid-server/v4/logger"

type prebidLogger struct{}

func (prebidLogger) Debug(message string) { logger.Debugf("%s", message) }
func (prebidLogger) Warn(message string)  { logger.Warnf("%s", message) }
func (prebidLogger) Error(message string) { logger.Errorf("%s", message) }
