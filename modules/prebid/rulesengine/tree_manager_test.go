package rulesengine

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

func TestTreeManagerShutdown(t *testing.T) {
	//schemaFile := "config/rules-engine-schema.json"
	//validator, err := config.CreateSchemaValidator(schemaFile)
	//assert.NoError(t, err, fmt.Sprintf("could not create schema validator using file %s", schemaFile))

	tm := &treeManager{
		done:            make(chan struct{}),
		requests:        make(chan buildInstruction, 10),
		schemaValidator: nil,
		//schemaValidator: validator,
	}
	cache := NewCache()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := tm.Run(cache)
		wg.Done()
		assert.NoError(t, err)
	}()

	//tm.done <- struct{}{}
	tm.Shutdown()

	wg.Wait()
	assert.Equal(t, int64(1), glog.Stats.Info.Lines(), "Expected some error logs to be generated")
}

// if the glog cannot be reset, only compare elements in the cache before v after the run
func TestTreeManagerRun(t *testing.T) {
	//schemaFile := "config/rules-engine-schema.json"
	//validator, err := config.CreateSchemaValidator(schemaFile)
	//assert.NoError(t, err, fmt.Sprintf("could not create schema validator using file %s", schemaFile))

	type testInput struct {
		cachedData       map[accountID]*cacheEntry
		validator        *gojsonschema.Schema
		buildInstruction buildInstruction
	}

	testCases := []struct {
		desc                    string
		in                      testInput
		expectedErrorLogEntries int
	}{
		{
			desc: "nil request.config",
			in: testInput{
				//cacher:    &testCache{},
				//validator: validator,
				cachedData: map[accountID]*cacheEntry{
					"account-id-one": {
						hashedConfig: "hash1",
					},
				},
				buildInstruction: buildInstruction{
					config: nil,
				},
			},
			expectedErrorLogEntries: 2,
		},
		{
			desc: "request.accountID not found",
			in: testInput{
				//cacher:    &testCache{},
				//validator: validator,
				cachedData: map[accountID]*cacheEntry{
					"account-id-one": {
						hashedConfig: "hash1",
					},
				},
				buildInstruction: buildInstruction{
					accountID: "account-id-two",
				},
			},
			expectedErrorLogEntries: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// set test
			var wg sync.WaitGroup
			wg.Add(2)
			logger := &mockLogger{
				f: func() {
					wg.Done()
				}
			}

			tm := &treeManager{
				done:            make(chan struct{}),
				requests:        make(chan buildInstruction, 10),
				schemaValidator: tc.in.validator,
				monitor: logger,
			}
			cache := NewCache()
			cache.m.Store(tc.in.cachedData)

			// run
			go func() {
				err := tm.Run(cache)
				assert.NoError(t, err)
			}()
			tm.requests <- tc.in.buildInstruction
			wg.Wait()

			// assertions
			assert.Equal(t, int64(tc.expectedErrorLogEntries), glog.Stats.Error.Lines())
			assert.Equal(t, int64(0), glog.Stats.Info.Lines())
			tm.done <- struct{}{}
		})
	}
}

type mockLogger struct{
	mock.Mock
	f func()
}

// func (logger *treeManagerLogger) logError(format string, a ...any) {
func (logger *testLogger) logError(msg string) {
	m.Called()
	f()
	return
}

// func (logger *treeManagerLogger) logInfo(format string, a ...any) {
func (logger *testLogger) logInfo(msg string) {
	m.Called()
	f()
	return
}

func captureStderr(f func()) (string, error) {
	old := os.Stderr // keep backup of the real stderr
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stderr = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// calling function which stderr we are going to capture:
	f()

	// back to normal state
	w.Close()
	os.Stderr = old // restoring the real stderr
	return <-outC, nil
}

func TestGlogError(t *testing.T) {
	stdErr, err := captureStderr(func() {
		glog.Error("Test error")
	})
	if err != nil {
		t.Errorf("should not be error, instead: %+v", err)
	}
	if !strings.HasSuffix(strings.TrimSpace(stdErr), "Test error") {
		t.Errorf("stderr should end by 'Test error' but it doesn't: %s", stdErr)
	}
}
