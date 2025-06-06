package rulesengine

import "github.com/golang/glog"

type RulesEngineObserver interface {
	logError(msg string)
	logInfo(msg string)
}

type treeManagerLogger struct{}

func (logger *treeManagerLogger) logError(msg string) {
	// TODO: log metric
	glog.Errorf(msg)
	return
}

func (logger *treeManagerLogger) logInfo(msg string) {
	// TODO: log metric
	glog.Infoln(msg)
	return
}
