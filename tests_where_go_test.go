package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	EveryDirOptions struct {
		preWalkDirFuncs []fs.WalkDirFunc
	}
	EveryDirOpt func(*EveryDirOptions)
)

// WithPreWalkDirFuncs returns an EveryDirOpt that adds functions to call before walking each directory.
func WithPreWalkDirFuncs(funcs ...fs.WalkDirFunc) EveryDirOpt {
	return func(o *EveryDirOptions) {
		o.preWalkDirFuncs = append(o.preWalkDirFuncs, funcs...)
	}
}

func newEveryDirOptions(opts ...EveryDirOpt) EveryDirOptions {
	opt := EveryDirOptions{}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

// FirstFunc finds the first element of a slice returning that match and true otherwise false
func FirstFunc[Slice ~[]T, T any](s Slice, matchFunc func(T) bool) (T, bool) {
	for _, v := range s {
		if matchFunc(v) {
			return v, true
		}
	}
	var ret T
	return ret, false
}

// EveryDirWithGoCodeHasTests asserts that every directory in the current working directory
func EveryDirWithGoCodeHasTests(t *testing.T, opts ...EveryDirOpt) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	everyDirOptions := newEveryDirOptions(opts...)
	assert.NoError(t, filepath.WalkDir(wd, func(path string, d os.DirEntry, err error) error {
		for _, f := range everyDirOptions.preWalkDirFuncs {
			err = f(path, d, err)
		}
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			t.Run(path, func(t *testing.T) {
				t.Parallel()
				matches, err := filepath.Glob(filepath.Join(path, "*.go"))
				require.NoError(t, err)
				if nonTestFile, ok := FirstFunc(matches, func(match string) bool {
					return !strings.HasSuffix(match, "_test.go")
				}); ok {
					_, ok = FirstFunc(matches, func(match string) bool {
						return strings.HasSuffix(match, "_test.go")
					})
					assert.Truef(t, ok, "found directory with go code but without a any tests %s", filepath.Dir(nonTestFile))
				}
			})
		}
		return nil
	}))
}
