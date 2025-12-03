package ssl

import (
	"crypto/x509"
	"fmt"
	"os"
)

func CreateCertPool() (*x509.CertPool, error) {
	return x509.SystemCertPool()
}

// AppendPEMFileToRootCAPool appends certificates from a PEM file to the provided certificate pool.
// This is a helper method intended for use in main startup code to append specific certificates
// to the system certificate pool.
func AppendPEMFileToRootCAPool(certPool *x509.CertPool, pemFileName string) (*x509.CertPool, error) {
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	if pemFileName != "" {
		pemCerts, err := os.ReadFile(pemFileName)
		if err != nil {
			return certPool, fmt.Errorf("Failed to read file %s: %v", pemFileName, err)
		}

		certPool.AppendCertsFromPEM(pemCerts)
	}

	return certPool, nil
}
