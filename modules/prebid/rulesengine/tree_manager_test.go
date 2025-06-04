package rulesengine

import (
	"fmt"
	"sync"
	"testing"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/stretchr/testify/assert"
)

func TestTreeManagerRun(t *testing.T) {
	schemaFile := "config/rules-engine-schema.json"
	validator, err := config.CreateSchemaValidator(schemaFile)
	assert.NoError(t, err, fmt.Sprintf("could not create schema validator using file %s", schemaFile))

	tm := &treeManager{
		done:     make(chan struct{}),
		requests: make(chan buildInstruction, 10),
		//schemaValidator: nil,
		schemaValidator: validator,
	}
	cache := testCache{}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := tm.Run(&cache)
		wg.Done()
		assert.NoError(t, err)
	}()

	tm.done <- struct{}{}

	wg.Wait()
	assert.Equal(t, int64(1), glog.Stats.Info.Lines(), "Expected some error logs to be generated")
	//type testInput struct{
	//	cacher cacher
	//	request buildInstruction
	//}
	//testCases := []struct {
	//	desc string
	//	in testInput
	//	expected string
	//}{
	//	{
	//		desc: "",
	//		in: testInput{},
	//		expected: "",
	//	},
	//}
	//for _, tc := range testCases {
	//	// set test
	//	// run
	//	t.Run(tc.desc, func(t *testing.T) {
	//		// assertions
	//		assert.Equal(t, tc.expected, out, tc.desc)
	//	})
	//}
}

//func TestTreeManagerRun2(t *testing.T) {
//	// Mock cacher
//	mockCacher := &MockCacher{}
//
//	// Create a treeManager instance
//	tm := &treeManager{
//		done:            make(chan struct{}),
//		requests:        make(chan buildInstruction, 10),
//		schemaValidator: nil, // Assume schemaValidator is set up correctly
//	}
//
//	// Start the treeManager in a goroutine
//	go func() {
//		err := tm.Run(mockCacher)
//		assert.NoError(t, err)
//	}()
//
//	// Send a build instruction to the treeManager
//	tm.requests <- buildInstruction{
//		accountID: "test_account",
//		config:    nil, // Assume config is set up correctly
//	}
//
//	// Close the done channel to stop the treeManager
//	close(tm.done)
//
//	// Check if the cacher received the correct account ID
//	assert.Equal(t, "test_account", mockCacher.lastAccountID)
//}

type testCache struct {
	entry *cacheEntry
}

func (c *testCache) Get(id accountID) *cacheEntry {
	return c.entry
}
func (c *testCache) Set(id accountID, data *cacheEntry) {
	c.entry = data
	return
}
func (c *testCache) Delete(id accountID) {
	return
}
