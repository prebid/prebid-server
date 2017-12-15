package analytics

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	EVENT_TYPE = "/request"
)

type Event interface {
}

type Request struct {
	Type   string
	Start  time.Time
	End    time.Time
	Events []Event
}

func (r *Request) Log() {
	b, _ := json.Marshal(r)
	fmt.Println(string(b))
}

type LogObject interface {
	Log()
}

type AnalyticsLogger interface {
	LogTransaction(LogObject)
}

type FileLogger struct {
	fileName string
}

func (f *FileLogger) Setup() {

}

func (f *FileLogger) LogTransaction(lo LogObject) {
	//TODO: Write to file
}

type GraphiteLogger struct {
	fileName string
}

func (f *GraphiteLogger) Setup() {

}

func (g *GraphiteLogger) LogTransaction(lo LogObject) {
	//TODO: Write to graphite
}
