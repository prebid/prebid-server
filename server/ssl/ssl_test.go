package ssl

import (
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendPEMFileToRootCAPool(t *testing.T) {
	t.Run("append-to-empty", func(t *testing.T) {
		var certPool *x509.CertPool = nil

		certificatesFile := "mockcertificates/mock-certs.pem"
		certPool, err := AppendPEMFileToRootCAPool(certPool, certificatesFile)

		require.NoError(t, err)
		subjects := certPool.Subjects()
		require.Equal(t, len(subjects), 1)
	})

	t.Run("fail", func(t *testing.T) {
		var certPool *x509.CertPool

		certificatesFile := "mockcertificates/NO-FILE.pem"
		_, err := AppendPEMFileToRootCAPool(certPool, certificatesFile)

		// expect an error from a file which doesn't exist
		assert.Error(t, err)
	})
}
