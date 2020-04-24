package deploy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// WritePIDFile writes the process identification number (PID) of ad_service
// to a file for Ops `grace-shepherd.sh` script to manage the app
func WritePIDFile(dir string, mode os.FileMode) (int, error) {
	if dir == "" {
		return 0, errors.New("no path provided")
	}

	pid := os.Getpid()

	filepath := fmt.Sprintf("%s/%d.pid", strings.TrimSuffix(dir, "/"), pid)
	err := ioutil.WriteFile(filepath, []byte(strconv.Itoa(pid)), mode)
	if err != nil {
		return pid, err
	}

	return pid, nil
}
