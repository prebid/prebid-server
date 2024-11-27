package remotefilesyncer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/prebid/prebid-server/v3/util/task"

	"github.com/golang/glog"
)

var (
	ErrSyncInProgress = errors.New("sync in progress")
)

type RemoteFileProcessor interface {
	SetDataPath(datapath string) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Options struct {
	Processor      RemoteFileProcessor
	Client         HTTPClient
	DownloadURL    string
	SaveFilePath   string
	TmpFilePath    string
	RetryCount     int
	RetryInterval  time.Duration
	Timeout        time.Duration
	UpdateInterval time.Duration
}

func (o Options) Validate() error {
	if o.Processor == nil || o.Client == nil {
		return fmt.Errorf("processor and client must not be nil")
	}
	if o.DownloadURL == "" || o.SaveFilePath == "" || o.TmpFilePath == "" {
		return fmt.Errorf("downloadURL, saveFilePath and tmpFilePath must not be empty")
	}
	if o.RetryCount < 0 || o.RetryInterval < 0 {
		return fmt.Errorf("retryCount and retryInterval must not be negative")
	}
	if o.Timeout < 0 || o.UpdateInterval < 0 {
		return fmt.Errorf("timeout and updateInterval must not be negative")
	}
	return nil
}

type RemoteFileSyncer struct {
	ttask          *task.TickerTask
	done           chan struct{}
	processor      RemoteFileProcessor
	client         HTTPClient
	syncing        atomic.Bool
	downloadURL    string
	saveFilePath   string
	tmpFilePath    string
	retryCount     int
	retryInterval  time.Duration
	timeout        time.Duration
	updateInterval time.Duration
}

func NewRemoteFileSyncer(opt Options) (*RemoteFileSyncer, error) {
	if err := opt.Validate(); err != nil {
		return nil, err
	}
	if err := createAndCheckWritePermissionsFor(opt.SaveFilePath); err != nil {
		return nil, err
	}
	if err := createAndCheckWritePermissionsFor(opt.TmpFilePath); err != nil {
		return nil, err
	}

	syncer := &RemoteFileSyncer{
		done:           make(chan struct{}),
		processor:      opt.Processor,
		client:         opt.Client,
		downloadURL:    opt.DownloadURL,
		saveFilePath:   opt.SaveFilePath,
		tmpFilePath:    opt.TmpFilePath,
		retryCount:     opt.RetryCount,
		retryInterval:  opt.RetryInterval,
		timeout:        opt.Timeout,
		updateInterval: opt.UpdateInterval,
	}
	syncer.ttask = task.NewTickerTaskWithOptions(task.Options{
		Interval: opt.UpdateInterval,
		Runner: task.NewFuncRunner(func() error {
			err := syncer.updateIfNeeded()
			if err != nil {
				glog.Errorf("updateIfNeeded error: %v", err)
			}
			return nil
		}),
		SkipInitialRun: true,
	})
	return syncer, nil
}

// Start sync now and starts a ticker to sync the file periodically.
func (s *RemoteFileSyncer) Start() error {
	if _, err := os.Stat(s.saveFilePath); errors.Is(err, os.ErrNotExist) {
		errRun := s.run()
		if errRun != nil {
			glog.Errorf("run error: %v", errRun)
			return errRun
		}
	} else {
		errPSF := s.processSavedFile()
		if errPSF != nil {
			glog.Errorf("process saved file error: %v", errPSF)
			return errPSF
		}
	}

	s.ttask.Start()

	return nil
}

// Stop the ticker and close the done channel.
func (s *RemoteFileSyncer) Stop() {
	if s.ttask != nil {
		s.ttask.Stop()
	}
	close(s.done)
}

// run starts the job.
// there is only one syncing process allowed at a time.
func (s *RemoteFileSyncer) run() error {
	if !s.syncing.CompareAndSwap(false, true) {
		return ErrSyncInProgress
	}
	defer s.syncing.Store(false)

	for retries := 0; ; retries++ {
		err := s.sync()
		if err == nil {
			if errPSF := s.processSavedFile(); errPSF == nil {
				break
			} else {
				glog.Infof("process saved file error: %v", errPSF)
			}
		} else {
			glog.Infof("sync file error: %v", err)
		}

		if retries >= s.retryCount {
			return fmt.Errorf("sync file max retries exceeded (%d)", s.retryCount)
		}

		select {
		case <-time.After(s.retryInterval):
			continue
		case <-s.done:
			return errors.New("sync file stopped")
		}
	}

	return nil
}

func (s *RemoteFileSyncer) sync() error {
	err := downloadFileFromURL(s.client, s.downloadURL, s.tmpFilePath, s.timeout)
	if err != nil {
		return err
	}

	err = os.Rename(s.tmpFilePath, s.saveFilePath)
	if err != nil {
		_ = os.Remove(s.tmpFilePath)
		return err
	}
	return nil
}

func (s *RemoteFileSyncer) processSavedFile() error {
	if err := s.processor.SetDataPath(s.saveFilePath); err != nil {
		_ = os.Remove(s.saveFilePath)
		return err
	}
	return nil
}

func (s *RemoteFileSyncer) updateIfNeeded() error {
	fileinfo, err := os.Stat(s.saveFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return s.run()
	}

	remoteSize, err := remoteFileSize(s.client, s.downloadURL, s.timeout)
	if err != nil {
		return err
	}
	if remoteSize != fileinfo.Size() {
		return s.run()
	}
	return nil
}

func createAndCheckWritePermissionsFor(datapath string) error {
	dir := filepath.Dir(datapath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	temp, err := os.CreateTemp(dir, "permission_test_")
	if err != nil {
		return fmt.Errorf("no write permission in directory: %v", err)
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	r, err := os.Open(temp.Name())
	if err != nil {
		return fmt.Errorf("no read permission in directory: %v", err)
	}
	defer r.Close()

	return nil
}

// downloadFileFromURL downloads a file to the datapath. overwrite datapath if it exists.
func downloadFileFromURL(client HTTPClient, url string, datapath string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	output, err := os.Create(datapath)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, resp.Body); err != nil {
		return err
	}
	return nil
}

func remoteFileSize(client HTTPClient, url string, timeout time.Duration) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	return resp.ContentLength, nil
}
