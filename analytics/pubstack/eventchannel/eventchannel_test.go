package eventchannel

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var maxByteSize = int64(15)
var maxEventCount = int64(3)
var maxTime = 2 * time.Hour

func readGz(encoded bytes.Buffer) string {
	gr, _ := gzip.NewReader(bytes.NewBuffer(encoded.Bytes()))
	defer gr.Close()

	decoded, _ := ioutil.ReadAll(gr)
	return string(decoded)
}

func newSender(data *[]byte) Sender {
	mux := &sync.Mutex{}
	return func(payload []byte) error {
		mux.Lock()
		defer mux.Unlock()
		event := bytes.Buffer{}
		event.Write(payload)
		*data = append(*data, readGz(event)...)
		return nil
	}
}

func TestEventChannel_isBufferFull(t *testing.T) {
	send := func([]byte) error { return nil }

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	defer eventChannel.Close()

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))

	assert.False(t, eventChannel.isBufferFull())

	eventChannel.buffer([]byte("three"))

	assert.True(t, eventChannel.isBufferFull())

	eventChannel.reset()

	assert.False(t, eventChannel.isBufferFull())

	eventChannel.buffer([]byte("big-event-abcdefghijklmnopqrstuvwxyz"))

	assert.True(t, eventChannel.isBufferFull())
}

func TestEventChannel_reset(t *testing.T) {
	send := func([]byte) error { return nil }

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	defer eventChannel.Close()

	assert.Zero(t, eventChannel.metrics.eventCount)
	assert.Zero(t, eventChannel.metrics.bufferSize)

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))

	assert.NotZero(t, eventChannel.metrics.eventCount)
	assert.NotZero(t, eventChannel.metrics.bufferSize)

	eventChannel.reset()

	assert.Zero(t, eventChannel.buff.Len())
	assert.Zero(t, eventChannel.metrics.eventCount)
	assert.Zero(t, eventChannel.metrics.bufferSize)
}

func TestEventChannel_flush(t *testing.T) {
	data := make([]byte, 0)
	send := newSender(&data)

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	defer eventChannel.Close()

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))
	eventChannel.buffer([]byte("three"))
	eventChannel.flush()
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, "onetwothree", string(data))
}

func TestEventChannel_close(t *testing.T) {
	data := make([]byte, 0)
	send := newSender(&data)

	eventChannel := NewEventChannel(send, 15000, 15000, 2*time.Hour)

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))
	eventChannel.buffer([]byte("three"))
	eventChannel.Close()

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, "onetwothree", string(data))
}

func TestEventChannel_Push(t *testing.T) {
	data := make([]byte, 0)
	send := newSender(&data)

	eventChannel := NewEventChannel(send, 15000, 5, 5*time.Millisecond)
	defer eventChannel.Close()

	eventChannel.Push([]byte("1"))
	eventChannel.Push([]byte("2"))
	eventChannel.Push([]byte("3"))

	// wait for events to be pushed
	time.Sleep(10 * time.Millisecond)

	assert.ElementsMatch(t, []byte{'1', '2', '3'}, data)
}

func TestEventChannel_OutputFormat(t *testing.T) {
	toGzip := func(payload string) []byte {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)

		if _, err := zw.Write([]byte(payload)); err != nil {
			assert.Fail(t, err.Error())
		}

		if err := zw.Close(); err != nil {
			assert.Fail(t, err.Error())
		}
		return buf.Bytes()
	}

	data := make([]byte, 0)
	send := func(payload []byte) error {
		data = append(data, payload...)
		return nil
	}

	eventChannel := NewEventChannel(send, 15000, 10, 2*time.Minute)

	eventChannel.Push([]byte("one"))
	time.Sleep(1 * time.Millisecond)

	eventChannel.flush()

	eventChannel.Push([]byte("two"))
	time.Sleep(1 * time.Millisecond)

	eventChannel.Push([]byte("three"))
	time.Sleep(1 * time.Millisecond)

	eventChannel.Close()
	time.Sleep(1 * time.Millisecond)

	expected := append(toGzip("one"), toGzip("twothree")...)

	assert.Equal(t, expected, data)
}
