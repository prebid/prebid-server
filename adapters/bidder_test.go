package adapters

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"

	"gopkg.in/yaml.v2"
)

// TestInfoFile makes sure that every bidder contains a valid info.yaml file.
func TestInfoFile(t *testing.T) {
	fileInfos, err := ioutil.ReadDir("../static/bidder-info")
	if err != nil {
		t.Fatalf("Error reading the ../static/bidder-info directory: %v", err)
	}

	// Make sure that files exist for each BidderName
	for bidderName, _ := range openrtb_ext.BidderMap {
		if _, err := os.Stat(fmt.Sprintf("../static/bidder-info/%s.yaml", bidderName)); os.IsNotExist(err) {
			t.Errorf("static/bidder-info/%s.yaml not found. Did you forget to create it?", bidderName)
		}
	}

	// Make sure that all the files have valid content
	for _, fileInfo := range fileInfos {
		infoFile, err := os.Open(fmt.Sprintf("../static/bidder-info/%s", fileInfo.Name()))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
			continue
		}

		content, err := ioutil.ReadAll(infoFile)
		if err != nil {
			t.Errorf("Failed to read static/bidder-info/%s: %v", fileInfo.Name(), err)
			continue
		}
		var fileInfoContent infoFileStructure
		if err := yaml.Unmarshal(content, &fileInfoContent); err != nil {
			t.Errorf("Error interpreting content from static/bidder-info/%s: %v", fileInfo.Name(), err)
			continue
		}
		if err := fileInfoContent.validate(); err != nil {
			t.Errorf("Invalid content in static/bidder-info/%s: %v", fileInfo.Name(), err)
		}
	}
}

type infoFileStructure struct {
	Maintainer maintainerInfo `yaml:"maintainer"`
}

func (info *infoFileStructure) validate() error {
	return info.Maintainer.validate()
}

type maintainerInfo struct {
	Email string `yaml:"email"`
}

func (info *maintainerInfo) validate() (err error) {
	if info.Email == "" {
		err = errors.New("Missing required field: maintainer.email")
	}
	return
}
