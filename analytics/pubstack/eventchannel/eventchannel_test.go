package eventchannel

import (
	"bytes"
	"compress/gzip"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

var maxByteSize = int64(15)
var maxEventCount = int64(3)
var maxTime = 2 * time.Hour

func TestEventChannel_isBufferFull(t *testing.T) {

	send := func(_ []byte) error { return nil }

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))

	assert.Equal(t, eventChannel.isBufferFull(), false)

	eventChannel.buffer([]byte("three"))
	assert.Equal(t, eventChannel.isBufferFull(), true)

	eventChannel.reset()
	assert.Equal(t, eventChannel.isBufferFull(), false)

	eventChannel.buffer([]byte("big-event-abcdefghijklmnopqrstuvwxyz"))
	assert.Equal(t, eventChannel.isBufferFull(), true)
}

func TestEventChannel_reset(t *testing.T) {

	send := func(_ []byte) error { return nil }

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	assert.Equal(t, eventChannel.metrics.eventCount, int64(0))
	assert.Equal(t, eventChannel.metrics.bufferSize, int64(0))

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))

	assert.NotEqual(t, eventChannel.metrics.eventCount, int64(0))
	assert.NotEqual(t, eventChannel.metrics.bufferSize, int64(0))

	eventChannel.reset()

	assert.Equal(t, eventChannel.buff.Len(), 0)
	assert.Equal(t, eventChannel.metrics.eventCount, int64(0))
	assert.Equal(t, eventChannel.metrics.bufferSize, int64(0))
}

func TestEventChannel_flush(t *testing.T) {
	data := bytes.Buffer{}
	send := func(payload []byte) error {
		data.Write(payload)
		return nil
	}
	maxByteSize := int64(15)
	maxEventCount := int64(3)
	maxTime := 2 * time.Hour

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))
	eventChannel.buffer([]byte("three"))
	eventChannel.flush()
	time.Sleep(10 * time.Millisecond)

	gr, _ := gzip.NewReader(bytes.NewBuffer(data.Bytes()))
	defer gr.Close()

	received, _ := ioutil.ReadAll(gr)
	assert.Equal(t, string(received), "onetwothree")
}
