package remotefilesyncer

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testRemoteFileProcessor struct {
	setDataPath func(datapath string) error
}

func (p *testRemoteFileProcessor) SetDataPath(datapath string) error {
	return p.setDataPath(datapath)
}

func makeTestRemoteFileProcessor(setDataPath func(datapath string) error) RemoteFileProcessor {
	return &testRemoteFileProcessor{setDataPath: setDataPath}
}

func makeTestRemoteFileProcessorOK() RemoteFileProcessor {
	return makeTestRemoteFileProcessor(func(datapath string) error {
		return nil
	})
}

type testHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (c *testHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.do(req)
}

func makeTestHTTPClient(do func(req *http.Request) (*http.Response, error)) HTTPClient {
	return &testHTTPClient{do: do}
}

func makeTestHTTPClient200() HTTPClient {
	return makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte("imdata"))),
		}, nil
	})
}

func createFile(datapath string, content string) {
	file, _ := os.Create(datapath)
	file.Write([]byte(content))
	file.Close()
}

func createTempFile(dir string, content string) (string, error) {
	file, err := os.CreateTemp(dir, "")
	if err != nil {
		return "", err
	}
	defer file.Close()
	file.Write([]byte(content))
	return file.Name(), nil
}

func assertFileContent(t *testing.T, datapath string, content string) {
	file, err := os.Open(datapath)
	assert.NoError(t, err, "File should exist")
	defer file.Close()
	buf := new(bytes.Buffer)
	io.Copy(buf, file)
	assert.Equal(t, content, buf.String(), "File content should be correct")
}

func TestOptionsValidate(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		processor      RemoteFileProcessor
		client         HTTPClient
		downloadURL    string
		saveFilePath   string
		tmpFilePath    string
		retryCount     int
		retryInterval  time.Duration
		timeout        time.Duration
		updateInterval time.Duration
		hasError       bool
	}{
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			false,
		},
		{
			nil,
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			nil,
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			"",
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			"",
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			-1,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			-10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			-10 * time.Second,
			0,
			true,
		},
		{
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			-1,
			true,
		},
	}

	for _, test := range tests {
		t.Run("OptionsValidate", func(t *testing.T) {
			opts := Options{
				Processor:      test.processor,
				Client:         test.client,
				DownloadURL:    test.downloadURL,
				SaveFilePath:   test.saveFilePath,
				TmpFilePath:    test.tmpFilePath,
				RetryCount:     test.retryCount,
				RetryInterval:  test.retryInterval,
				Timeout:        test.timeout,
				UpdateInterval: test.updateInterval,
			}
			err := opts.Validate()
			if test.hasError {
				assert.Error(t, err, "Options.Validate() should return error")
			} else {
				assert.NoError(t, err, "Options.Validate() should not return error. Error: %v", err)
			}
		})
	}
}

func TestNewRemoteFileSyncer(t *testing.T) {
	dir := t.TempDir()
	readdir := filepath.Join(dir, "readonly")
	os.MkdirAll(readdir, 0555)

	tests := []struct {
		name           string
		processor      RemoteFileProcessor
		client         HTTPClient
		downloadURL    string
		saveFilePath   string
		tmpFilePath    string
		retryCount     int
		retryInterval  time.Duration
		timeout        time.Duration
		updateInterval time.Duration
		hasError       bool
	}{
		{
			"New syncer, successful",
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			false,
		},
		{
			"New syncer, invalid options",
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			-1,
			true,
		},
		{
			"New syncer, read-only save file path",
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(readdir, "foo"),
			filepath.Join(dir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
		{
			"New syncer, read-only tmp file path",
			makeTestRemoteFileProcessorOK(),
			makeTestHTTPClient200(),
			"http://example.com",
			filepath.Join(dir, "foo"),
			filepath.Join(readdir, "tmp"),
			0,
			10 * time.Second,
			10 * time.Second,
			0,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := Options{
				Processor:      test.processor,
				Client:         test.client,
				DownloadURL:    test.downloadURL,
				SaveFilePath:   test.saveFilePath,
				TmpFilePath:    test.tmpFilePath,
				RetryCount:     test.retryCount,
				RetryInterval:  test.retryInterval,
				Timeout:        test.timeout,
				UpdateInterval: test.updateInterval,
			}
			syncer, err := NewRemoteFileSyncer(opts)
			if test.hasError {
				assert.Error(t, err, "NewRemoteFileSyncer should return error")
			} else {
				assert.NoError(t, err, "NewRemoteFileSyncer should not return error. Error: %v", err)
				assert.NotNil(t, syncer.ttask)
			}
		})
	}
}

func TestRemoteFileSyncerStart(t *testing.T) {
	const filecontent = "imdata"
	var (
		processorCalled  int64 = 0
		clientHeadCalled int64 = 0
		clientGetCalled  int64 = 0
	)
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor: makeTestRemoteFileProcessor(func(datapath string) error {
			atomic.AddInt64(&processorCalled, 1)
			return nil
		}),
		Client: makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
			switch req.Method {
			case http.MethodHead:
				atomic.AddInt64(&clientHeadCalled, 1)
				return &http.Response{
					StatusCode:    200,
					ContentLength: int64(len(filecontent)),
				}, nil
			case http.MethodGet:
				atomic.AddInt64(&clientGetCalled, 1)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte(filecontent))),
				}, nil
			}
			return nil, assert.AnError
		}),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(t.TempDir(), "foo"),
		TmpFilePath:    filepath.Join(t.TempDir(), "tmp"),
		RetryCount:     0,
		RetryInterval:  10 * time.Second,
		Timeout:        10 * time.Second,
		UpdateInterval: 10 * time.Millisecond,
	})
	err := syncer.Start()
	defer syncer.Stop()

	// wait for updating to be checked
	<-time.After(20 * time.Millisecond)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&processorCalled), "Processor should be called once")
	assert.Equal(t, int64(1), atomic.LoadInt64(&clientGetCalled), "HTTPClient should be called once for GET")
	assert.True(t, atomic.LoadInt64(&clientHeadCalled) >= 1, "HTTPClient should be called at least once for HEAD")
	assertFileContent(t, syncer.saveFilePath, filecontent)
}

func TestRemoteFileSyncerStartIsSyncing(t *testing.T) {
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor:      makeTestRemoteFileProcessorOK(),
		Client:         makeTestHTTPClient200(),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(t.TempDir(), "foo"),
		TmpFilePath:    filepath.Join(t.TempDir(), "tmp"),
		RetryCount:     0,
		RetryInterval:  10 * time.Second,
		Timeout:        10 * time.Second,
		UpdateInterval: 0,
	})
	syncer.syncing.Store(true) // Set syncing to true to know run called
	err := syncer.Start()
	defer syncer.Stop()

	assert.Equal(t, ErrSyncInProgress, err)
	assert.NoFileExists(t, syncer.saveFilePath)
}

func TestRemoteFileSyncerStartFileExists(t *testing.T) {
	dir := t.TempDir()
	datapath, _ := createTempFile(dir, "imdata")

	var (
		processorCalled int64 = 0
		clientCalled    int64 = 0
	)
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor: makeTestRemoteFileProcessor(func(datapath string) error {
			atomic.AddInt64(&processorCalled, 1)
			return nil
		}),
		Client: makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt64(&clientCalled, 1)
			return nil, nil
		}),
		DownloadURL:    "http://example.com",
		SaveFilePath:   datapath,
		TmpFilePath:    filepath.Join(t.TempDir(), "tmp"),
		RetryCount:     0,
		RetryInterval:  10 * time.Second,
		Timeout:        10 * time.Second,
		UpdateInterval: 0,
	})
	err := syncer.Start()
	defer syncer.Stop()

	assert.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&processorCalled), "Processor should be called once")
	assert.Equal(t, int64(0), atomic.LoadInt64(&clientCalled), "HTTPClient should not be called")
}

func TestRemoteFileSyncerStartFileExistsInvalid(t *testing.T) {
	dir := t.TempDir()
	datapath, _ := createTempFile(dir, "imdata")

	var (
		processorCalled int64 = 0
		clientCalled    int64 = 0
	)
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor: makeTestRemoteFileProcessor(func(datapath string) error {
			atomic.AddInt64(&processorCalled, 1)
			return assert.AnError
		}),
		Client: makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt64(&clientCalled, 1)
			return nil, nil
		}),
		DownloadURL:    "http://example.com",
		SaveFilePath:   datapath,
		TmpFilePath:    filepath.Join(t.TempDir(), "tmp"),
		RetryCount:     0,
		RetryInterval:  10 * time.Second,
		Timeout:        10 * time.Second,
		UpdateInterval: 0,
	})
	err := syncer.Start()
	defer syncer.Stop()

	assert.Error(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&processorCalled), "Processor should be called once")
	assert.Equal(t, int64(0), atomic.LoadInt64(&clientCalled), "HTTPClient should not be called")
}

func TestRemoteFileSyncerStop(t *testing.T) {
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor:      makeTestRemoteFileProcessorOK(),
		Client:         makeTestHTTPClient200(),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(t.TempDir(), "foo"),
		TmpFilePath:    filepath.Join(t.TempDir(), "tmp"),
		RetryCount:     0,
		RetryInterval:  10 * time.Second,
		Timeout:        10 * time.Second,
		UpdateInterval: 0,
	})
	syncer.Stop()

	_, ok := <-syncer.done
	assert.False(t, ok, "done channel should be closed")
	_, ok = <-syncer.ttask.Done()
	assert.False(t, ok, "TickerTask should be closed")
}

func TestRemoteFileSyncerRunRetryWhenSyncErr(t *testing.T) {
	dir := t.TempDir()

	var (
		clientCalled int64 = 0
	)
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor: makeTestRemoteFileProcessorOK(),
		Client: makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt64(&clientCalled, 1)
			return nil, assert.AnError
		}),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(dir, "foo"),
		TmpFilePath:    filepath.Join(dir, "tmp"),
		RetryCount:     2,
		RetryInterval:  10 * time.Millisecond,
		Timeout:        10 * time.Millisecond,
		UpdateInterval: 0,
	})
	err := syncer.run()
	assert.Error(t, err)
	assert.Equal(t, int64(3), clientCalled, "HTTPClient should be called 1 + 2 times")
}

func TestRemoteFileSyncerRunRetryWhenProcessSavedFileErr(t *testing.T) {
	dir := t.TempDir()

	var (
		processorCalled int64 = 0
	)
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor: makeTestRemoteFileProcessor(func(datapath string) error {
			atomic.AddInt64(&processorCalled, 1)
			return assert.AnError
		}),
		Client:         makeTestHTTPClient200(),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(dir, "foo"),
		TmpFilePath:    filepath.Join(dir, "tmp"),
		RetryCount:     2,
		RetryInterval:  10 * time.Millisecond,
		Timeout:        10 * time.Millisecond,
		UpdateInterval: 0,
	})
	err := syncer.run()
	assert.Error(t, err)
	assert.Equal(t, int64(3), processorCalled, "processor should be called 1 + 2 times")
}

func TestRemoteFileSyncerRunRetryWhenTaskStop(t *testing.T) {
	dir := t.TempDir()

	var (
		clientCalled int64 = 0
	)
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor: makeTestRemoteFileProcessorOK(),
		Client: makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt64(&clientCalled, 1)
			return nil, assert.AnError
		}),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(dir, "foo"),
		TmpFilePath:    filepath.Join(dir, "tmp"),
		RetryCount:     2,
		RetryInterval:  10 * time.Millisecond,
		Timeout:        10 * time.Millisecond,
		UpdateInterval: 0,
	})
	syncer.Stop()
	err := syncer.run()
	assert.Error(t, err)
	assert.Equal(t, int64(1), clientCalled, "HTTPClient should be called 1 times because syncer is stopped")
}

func TestRemoteFileSyncerSync(t *testing.T) {
	dir := t.TempDir()
	readdir := filepath.Join(dir, "readonly")
	os.MkdirAll(readdir, 0555)

	tests := []struct {
		name         string
		client       HTTPClient
		saveFilePath string
		hasErr       bool
	}{
		{
			"Sync successful",
			makeTestHTTPClient200(),
			filepath.Join(dir, "foo"),
			false,
		},
		{
			"Sync failed, client returns error",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return nil, assert.AnError
			}),
			filepath.Join(dir, "foo"),
			true,
		},
		{
			"Sync failed, save file path is read-only",
			makeTestHTTPClient200(),
			readdir,
			true,
		},
	}

	syncer, _ := NewRemoteFileSyncer(Options{
		Processor:      makeTestRemoteFileProcessorOK(),
		Client:         makeTestHTTPClient(nil),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(dir, "foo"),
		TmpFilePath:    filepath.Join(dir, "tmp"),
		RetryCount:     0,
		RetryInterval:  10 * time.Second,
		Timeout:        10 * time.Second,
		UpdateInterval: 0,
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			syncer.client = test.client
			syncer.saveFilePath = test.saveFilePath
			err := syncer.sync()
			if test.hasErr {
				assert.Error(t, err, "RemoteFileSyncer.sync should return error")
			} else {
				assert.NoError(t, err, "RemoteFileSyncer.sync should not return error. Error %v", err)
			}
		})
	}
}

func TestRemoteFileSyncerProcessSavedFile(t *testing.T) {
	dir := t.TempDir()
	datapath, _ := createTempFile(dir, "imdata")
	syncer := &RemoteFileSyncer{
		processor: makeTestRemoteFileProcessor(func(datapath string) error {
			return assert.AnError
		}),
		saveFilePath: datapath,
	}
	err := syncer.processSavedFile()
	assert.Error(t, err)
	assert.NoFileExists(t, datapath, "should remove file if process failed")
}

func TestRemoteFileSyncerUpdateIfNeeded(t *testing.T) {
	dir := t.TempDir()
	syncer, _ := NewRemoteFileSyncer(Options{
		Processor:      makeTestRemoteFileProcessorOK(),
		Client:         makeTestHTTPClient(nil),
		DownloadURL:    "http://example.com",
		SaveFilePath:   filepath.Join(dir, "foo"),
		TmpFilePath:    filepath.Join(dir, "tmp"),
		RetryCount:     0,
		RetryInterval:  10 * time.Second,
		Timeout:        10 * time.Second,
		UpdateInterval: 0,
	})

	tests := []struct {
		name         string
		client       HTTPClient
		saveFilePath string
		saveFileData string
		needed       bool
		hasError     bool
	}{
		{
			"File not exists",
			makeTestHTTPClient200(),
			filepath.Join(dir, "foo"),
			"",
			true,
			true,
		},
		{
			"File exists, get content length error",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return nil, assert.AnError
			}),
			filepath.Join(dir, "foo"),
			"imdata",
			false,
			true,
		},
		{
			"File exists, content length is different",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{ContentLength: 1}, nil
			}),
			filepath.Join(dir, "foo"),
			"imdata",
			true,
			true,
		},
		{
			"File exists, content length is the same",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{ContentLength: 6}, nil
			}),
			filepath.Join(dir, "foo"),
			"imdata",
			false,
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.saveFileData != "" {
				createFile(test.saveFilePath, test.saveFileData)
			}
			syncer.client = test.client
			syncer.saveFilePath = test.saveFilePath
			syncer.syncing.Store(true) // let run return a known error for checking run is called

			err := syncer.updateIfNeeded()
			if test.hasError {
				assert.Error(t, err, "RemoteFileSyncer.updateIfNeeded should return error")
			} else {
				assert.NoError(t, err, "RemoteFileSyncer.updateIfNeeded should not return error. Error: %v", err)
			}
			if test.needed {
				assert.Equal(t, ErrSyncInProgress, err)
			}
		})
	}
}

func TestCreateAndCheckWritePermissionsFor(t *testing.T) {
	dir := t.TempDir()
	readdir := filepath.Join(dir, "readonly")
	os.MkdirAll(readdir, 0555)

	tests := []struct {
		name     string
		datapath string
		hasError bool
	}{
		{
			"Directory write permissions are granted",
			filepath.Join(dir, "foo"),
			false,
		},
		{
			"Directory write permissions are granted to a nested directory",
			filepath.Join(dir, "foo/bar"),
			false,
		},
		{
			"Directory is read-only, can not create file",
			filepath.Join(readdir, "foo"),
			true,
		},
		{
			"Directory is read-only, can not create directory",
			filepath.Join(readdir, "foo/bar"),
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := createAndCheckWritePermissionsFor(test.datapath)
			if test.hasError {
				assert.Error(t, err, "should return error")
				assert.NoFileExists(t, test.datapath, "should not create file")
			} else {
				assert.NoError(t, err, "should not return error. Error: %v", err)
			}
		})
	}
}

func TestDownloadFileFromURL(t *testing.T) {
	tests := []struct {
		name     string
		client   HTTPClient
		url      string
		datapath string
		timeout  time.Duration
		data     string
		hasError bool
	}{
		{
			"Succesful. Downloaded data is 'imdata'",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte("imdata"))),
				}, nil
			}),
			"http://example.com",
			filepath.Join(t.TempDir(), "foo"),
			1 * time.Second,
			"imdata",
			false,
		},
		{
			"Download from a invalid URL",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte("imdata"))),
				}, nil
			}),
			"!@#$%^&*()",
			filepath.Join(t.TempDir(), "foo"),
			1 * time.Second,
			"",
			true,
		},
		{
			"Download from a valid URL, returns error",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return nil, assert.AnError
			}),
			"http://example.com",
			filepath.Join(t.TempDir(), "foo"),
			1 * time.Second,
			"",
			true,
		},
		{
			"Download from a valid URL, returns status code 404",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 404}, nil
			}),
			"http://example.com",
			filepath.Join(t.TempDir(), "foo"),
			1 * time.Second,
			"",
			true,
		},
		{
			"Download from a valid URL, save to a directory",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte("imdata"))),
				}, nil
			}),
			"http://example.com",
			t.TempDir(),
			1 * time.Second,
			"",
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := downloadFileFromURL(test.client, test.url, test.datapath, test.timeout)
			if test.hasError {
				assert.Error(t, err, "DownloadFileFromURL should return error")
			} else {
				assert.NoError(t, err, "DownloadFileFromURL should not return error. Error: %v", err)
				assertFileContent(t, test.datapath, test.data)
			}
		})
	}
}

func TestRemoteFileSize(t *testing.T) {
	tests := []struct {
		name     string
		client   HTTPClient
		url      string
		timeout  time.Duration
		length   int64
		hasError bool
	}{
		{
			"Successful. ContentLength is 100",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					ContentLength: 100,
				}, nil
			}),
			"http://example.com",
			1 * time.Second,
			100,
			false,
		},
		{
			"Request a invalid URL",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					ContentLength: 100,
				}, nil
			}),
			"!@#$%^&*()",
			1 * time.Second,
			0,
			true,
		},
		{
			"Request a valid URL, returns error",
			makeTestHTTPClient(func(req *http.Request) (*http.Response, error) {
				return nil, assert.AnError
			}),
			"http://example.com",
			1 * time.Second,
			0,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			length, err := remoteFileSize(test.client, test.url, test.timeout)
			if test.hasError {
				assert.Error(t, err, "RemoteFileSize should return error")
			} else {
				assert.NoError(t, err, "RemoteFileSize should not return error. Error: %v", err)
				assert.Equal(t, test.length, length, "RemoteFileSize should return correct file size")
			}
		})
	}
}
