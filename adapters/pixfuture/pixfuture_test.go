package pixfuture

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPixfuture, config.Adapter{
		Endpoint: "http://any.url",
	}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	dirs := []string{"pixfuturetest/exemplary", "pixfuturetest/supplemental"}

	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			files, err := filepath.Glob(filepath.Join(dir, "*.json"))
			if err != nil {
				t.Fatalf("Failed to glob JSON files in %s: %v", dir, err)
			}

			for _, file := range files {
				t.Run(filepath.Base(file), func(t *testing.T) {
					// Create a temp directory
					tmpDir, err := ioutil.TempDir("", "pixfuture_test_")
					if err != nil {
						t.Fatalf("Failed to create temp dir: %v", err)
					}
					defer os.RemoveAll(tmpDir)

					// Copy the single JSON file to temp dir
					src := file
					dst := filepath.Join(tmpDir, filepath.Base(file))
					input, err := ioutil.ReadFile(src)
					if err != nil {
						t.Fatalf("Failed to read %s: %v", src, err)
					}
					if err := ioutil.WriteFile(dst, input, 0644); err != nil {
						t.Fatalf("Failed to write %s: %v", dst, err)
					}

					t.Logf("Testing JSON file: %s", file)
					adapterstest.RunJSONBidderTest(t, tmpDir, bidder)
				})
			}
		})
	}
}
