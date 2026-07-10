package rulesengine

import "github.com/prebid/prebid-server/v3/logger"

type RulesEngineObserver interface {
	logError(msg string)
	logInfo(msg string)
}

type treeManagerLogger struct{}

func (l *treeManagerLogger) logError(msg string) {
	// TODO: log metric
	logger.Errorf(msg)
	return
}

func (l *treeManagerLogger) logInfo(msg string) {
	// TODO: log metric
	logger.Infof(msg)
	return
}
