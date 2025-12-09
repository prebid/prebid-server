package eventchannel

import (
	"bytes"
	"compress/gzip"
	"io"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

var largeBufferSize = int64(math.MaxInt64)
var largeEventCount = int64(math.MaxInt64)
var maxTime = 2 * time.Hour

func readGz(encoded []byte) string {
	gr, _ := gzip.NewReader(bytes.NewReader(encoded))
	defer gr.Close()

	decoded, _ := io.ReadAll(gr)
	return string(decoded)
}

func newSender(dataSent chan []byte) Sender {
	mux := &sync.Mutex{}
	return func(payload []byte) error {
		mux.Lock()
		defer mux.Unlock()
		dataSent <- payload
		return nil
	}
}

func readChanOrTimeout(t *testing.T, c <-chan []byte, msgAndArgs ...interface{}) ([]byte, bool) {
	t.Helper()
	select {
	case actual := <-c:
		return actual, false
	case <-time.After(200 * time.Millisecond):
		return nil, assert.Fail(t, "Should receive an event, but did NOT", msgAndArgs...)
	}
}

func TestEventChannelIsBufferFull(t *testing.T) {
	send := func([]byte) error { return nil }
	clockMock := clock.NewMock()

	maxBufferSize := int64(15)
	maxEventCount := int64(3)

	eventChannel := NewEventChannel(send, clockMock, maxBufferSize, maxEventCount, maxTime)
	defer eventChannel.Close()

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))

	assert.False(t, eventChannel.isBufferFull()) // not yet full by either max buffer size or max event count

	eventChannel.buffer([]byte("three"))

	assert.True(t, eventChannel.isBufferFull()) // full by event count (3)

	eventChannel.reset()

	assert.False(t, eventChannel.isBufferFull()) // was just reset, should not be full

	eventChannel.buffer([]byte("larger-than-15-characters"))

	assert.True(t, eventChannel.isBufferFull()) // full by max buffer size
}

func TestEventChannelReset(t *testing.T) {
	send := func([]byte) error { return nil }
	clockMock := clock.NewMock()

	eventChannel := NewEventChannel(send, clockMock, largeBufferSize, largeEventCount, maxTime)
	defer eventChannel.Close()

	assert.Zero(t, eventChannel.metrics.eventCount)
	assert.Zero(t, eventChannel.metrics.bufferSize)

	eventChannel.buffer([]byte("one"))

	assert.NotZero(t, eventChannel.metrics.eventCount)
	assert.NotZero(t, eventChannel.metrics.bufferSize)

	eventChannel.reset()

	assert.Zero(t, eventChannel.buff.Len())
	assert.Zero(t, eventChannel.metrics.eventCount)
	assert.Zero(t, eventChannel.metrics.bufferSize)
}

func TestEventChannelFlush(t *testing.T) {
	dataSent := make(chan []byte)
	send := newSender(dataSent)
	clockMock := clock.NewMock()

	eventChannel := NewEventChannel(send, clockMock, largeBufferSize, largeEventCount, maxTime)
	defer eventChannel.Close()

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))
	eventChannel.buffer([]byte("three"))
	eventChannel.flush()

	data, _ := readChanOrTimeout(t, dataSent)
	assert.Equal(t, "onetwothree", readGz(data))
}

func TestEventChannelClose(t *testing.T) {
	dataSent := make(chan []byte)
	send := newSender(dataSent)
	clockMock := clock.NewMock()

	eventChannel := NewEventChannel(send, clockMock, largeBufferSize, largeEventCount, maxTime)

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))
	eventChannel.buffer([]byte("three"))
	eventChannel.Close()

	data, _ := readChanOrTimeout(t, dataSent)
	assert.Equal(t, "onetwothree", readGz(data))
}

func TestEventChannelPush(t *testing.T) {
	dataSent := make(chan []byte)
	send := newSender(dataSent)
	clockMock := clock.NewMock()

	eventChannel := NewEventChannel(send, clockMock, largeBufferSize, largeEventCount, 1*time.Second)
	defer eventChannel.Close()

	eventChannel.Push([]byte("1"))
	eventChannel.Push([]byte("2"))
	eventChannel.Push([]byte("3"))

	clockMock.Add(1 * time.Second) // trigger event timer

	data, _ := readChanOrTimeout(t, dataSent)
	assert.ElementsMatch(t, []byte{'1', '2', '3'}, []byte(readGz(data)))
}
