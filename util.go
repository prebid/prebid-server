package main

import (
	"os"

	"github.com/prebid/prebid-server/deploy"
)

func handleDeployPID(path string, mode os.FileMode) (int, error) {
	// make directory if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, mode)
	}

	return deploy.WritePIDFile(path, mode)
}
