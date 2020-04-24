// +build !integration

package deploy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWritePIDFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pid-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cases := []struct {
		tag  string
		dir  string
		mode os.FileMode

		expErr error
	}{
		// NOTE: depending on umask (-S) certain permissions may not be possible
		{"write pid file", tmpDir, 0644, nil},
		{"dir does not exist", "foo", 0, errors.New("no such file or directory")},
	}

	for _, c := range cases {
		t.Run(c.tag, func(t *testing.T) {
			pid, err := WritePIDFile(c.dir, c.mode)
			if !cmpFuzzyErr(err, c.expErr) {
				t.Fatalf("incorrect error writing pid, got '%v', exp '%v'", err, c.expErr)
			}

			if err == nil {
				filename := fmt.Sprintf("%d.pid", pid)
				filepath := filepath.Join(c.dir, filename)

				stat, err := os.Stat(filepath)
				if os.IsNotExist(err) {
					t.Fatalf("file '%s' does not exist", filepath)
				}

				content, err := ioutil.ReadFile(filepath)
				if err != nil {
					t.Fatal(err)
				}
				expPID := strconv.Itoa(pid)
				if string(content) != expPID {
					t.Errorf("incorrect pid in file, got '%s', exp '%s'", string(content), expPID)
				}

				if stat.Mode() != c.mode {
					t.Errorf("incorrect mode, got '%v', exp '%v'", stat.Mode(), c.mode)
				}
			}
		})
	}

}

func cmpFuzzyErr(x, y error) bool {
	if x == nil && y == nil {
		return true
	}

	return x != nil && y != nil && strings.Contains(x.Error(), y.Error())
}
