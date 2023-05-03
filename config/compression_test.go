package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReqCompressionCfgIsSupported(t *testing.T) {
	testCases := []struct {
		description     string
		cfg             ReqCompression
		compressionKind CompressionKind
		wantSupported   bool
	}{
		{
			description: "Compression type not supported",
			cfg: ReqCompression{
				Enabled: true,
				Kind:    []string{"gzip"},
			},
			compressionKind: CompressionKind("invalid"),
			wantSupported:   false,
		},
		{
			description: "Compression type supported",
			cfg: ReqCompression{
				Enabled: true,
				Kind:    []string{"gzip"},
			},
			compressionKind: CompressionGZIP,
			wantSupported:   true,
		},
		{
			description: "Compression not enabled",
			cfg: ReqCompression{
				Enabled: false,
				Kind:    []string{"gzip"},
			},
			compressionKind: CompressionGZIP,
			wantSupported:   false,
		},
	}

	for _, test := range testCases {
		assert.Empty(t, test.cfg.Validate([]error{}), test.description)
		got := test.cfg.IsSupported(test.compressionKind)
		assert.Equal(t, got, test.wantSupported, test.description)
	}
}

func TestCompressionKindIsValid(t *testing.T) {
	testCases := []struct {
		description string
		compression CompressionKind
		wantIsValid bool
	}{
		{
			description: "Compression type not supported",
			compression: CompressionKind("invalid"),
			wantIsValid: false,
		},
		{
			description: "Compression type supported",
			compression: CompressionGZIP,
			wantIsValid: true,
		},
	}

	for _, test := range testCases {
		got := test.compression.IsValid()
		assert.Equal(t, got, test.wantIsValid, test.description)
	}
}

func TestReqCompressionCfgValidate(t *testing.T) {
	testCases := []struct {
		description string
		cfg         ReqCompression
		wantErrs    []error
	}{
		{
			description: "Compression type not supported",
			cfg: ReqCompression{
				Enabled: true,
				Kind:    []string{"foo"},
			},
			wantErrs: []error{errors.New("compression type foo is not valid")},
		},
		{
			description: "Compression type supported",
			cfg: ReqCompression{
				Enabled: true,
				Kind:    []string{"gzip"},
			},
			wantErrs: []error{},
		},
		{
			description: "Compression type supported but compression kind value not in lower case",
			cfg: ReqCompression{
				Enabled: true,
				Kind:    []string{"GZIP"},
			},
			wantErrs: []error{},
		},
		{
			description: "Compression not enabled",
			cfg: ReqCompression{
				Enabled: false,
				Kind:    []string{"gzip"},
			},
			wantErrs: []error{},
		},
		{
			description: "Compression enabled but no compression types specified",
			cfg: ReqCompression{
				Enabled: true,
				Kind:    []string{},
			},
			wantErrs: []error{errors.New("compression is enabled but no compression types are specified")},
		},
	}

	for _, test := range testCases {
		got := test.cfg.Validate([]error{})
		assert.Equal(t, got, test.wantErrs, test.description)
	}
}

func TestRespCompressionCfgValidate(t *testing.T) {
	testCases := []struct {
		description string
		cfg         RespCompression
		wantErrs    []error
	}{
		{
			description: "Compression type not supported",
			cfg: RespCompression{
				Enabled: true,
				Kind:    "foo",
			},
			wantErrs: []error{errors.New("compression type foo is not valid")},
		},
		{
			description: "Compression type supported",
			cfg: RespCompression{
				Enabled: true,
				Kind:    "gzip",
			},
			wantErrs: []error{},
		},
		{
			description: "Compression type supported but compression kind value not in lower case",
			cfg: RespCompression{
				Enabled: true,
				Kind:    "GZIP",
			},
			wantErrs: []error{},
		},
		{
			description: "Compression not enabled",
			cfg: RespCompression{
				Enabled: false,
				Kind:    "gzip",
			},
			wantErrs: []error{},
		},
	}

	for _, test := range testCases {
		got := test.cfg.Validate([]error{})
		assert.Equal(t, got, test.wantErrs, test.description)
	}
}
