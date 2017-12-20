package adapters

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"testing"
)

// TestInfoFile makes sure that every bidder contains a valid info.yaml file.
func TestInfoFile(t *testing.T) {
	fileInfos, err := ioutil.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read local directory.")
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() && fileInfo.Name() != "adapterstest" {
			infoFile, err := os.Open(fmt.Sprintf("./%s/info.yaml", fileInfo.Name()))
			if err != nil {
				t.Errorf("Error: %v. Did you forget to add it?", err)
				continue
			}

			content, err := ioutil.ReadAll(infoFile)
			if err != nil {
				t.Errorf("Failed to read adapters/%s/info.yaml: %s", fileInfo.Name(), err.Error())
				continue
			}
			var fileInfoContent infoFileStructure
			if err := yaml.Unmarshal(content, &fileInfoContent); err != nil {
				t.Errorf("Error interpreting content from adapters/%s/info.yaml: %s", fileInfo.Name(), err.Error())
				continue
			}
			if err := fileInfoContent.validate(); err != nil {
				t.Errorf("Invalid content in adapters/%s/info.yaml: %s", fileInfo.Name(), err.Error())
			}
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
