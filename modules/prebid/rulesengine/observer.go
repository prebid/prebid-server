package rulesengine

import "github.com/golang/glog"

type RulesEngineObserver interface {
	logError(msg string)
	logInfo(msg string)
}

type treeManagerLogger struct{}

// func (logger *treeManagerLogger) logError(format string, a ...any) {
func (logger *treeManagerLogger) logError(msg string) {
	// TODO: log metric
	//glog.Errorf(format, a...)
	glog.Errorf(msg)
	return
}

// func (logger *treeManagerLogger) logInfo(format string, a ...any) {
func (logger *treeManagerLogger) logInfo(msg string) {
	// TODO: log metric
	glog.Infoln(msg)
	//glog.Infof(format, a...)
	return
}
