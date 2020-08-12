package files

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestNewFilesLoader(t *testing.T) {
	ev := NewFilesLoader(config.FileFetcherConfig{
		Path: "../../backends/file_fetcher/test",
	})
	theSave := <-ev.Saves()

	assert.Equal(t, 2, len(theSave.Requests))
	assert.JSONEq(t, `{"test":"foo"}`, string(theSave.Requests["1"]))

	assert.Equal(t, len(theSave.Imps), 1)
	assert.JSONEq(t, `{"imp":true}`, string(theSave.Imps["some-imp"]))
}
