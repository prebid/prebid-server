package ssl

import (
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCertsFromFilePoolExists(t *testing.T) {
	// Load hardcoded certificates found in ssl.go
	certPool := GetRootCAPool()

	// Assert loaded certificates by looking at the length of the subjects array of strings
	subjects := certPool.Subjects()
	hardCodedSubNum := len(subjects)
	assert.True(t, hardCodedSubNum > 0)

	// Load certificates from file
	certificatesFile := "mockcertificates/mock-certs.pem"
	certPool, err := AppendPEMFileToRootCAPool(certPool, certificatesFile)

	// Assert loaded certificates by looking at the length of the subjects array of strings
	assert.NoError(t, err, "Error thrown by AppendPEMFileToRootCAPool while loading file %s: %v", certificatesFile, err)
	subjects = certPool.Subjects()
	subNumIncludingFile := len(subjects)
	assert.True(t, subNumIncludingFile > hardCodedSubNum, "subNumIncludingFile should be greater than hardCodedSubNum")
}

func TestCertsFromFilePoolDontExist(t *testing.T) {
	// Empty certpool
	var certPool *x509.CertPool = nil

	// Load certificates from file
	certificatesFile := "mockcertificates/mock-certs.pem"
	certPool, err := AppendPEMFileToRootCAPool(certPool, certificatesFile)

	// Assert loaded certificates by looking at the length of the subjects array of strings
	assert.NoError(t, err, "Error thrown by AppendPEMFileToRootCAPool while loading file %s: %v", certificatesFile, err)
	subjects := certPool.Subjects()
	assert.Equal(t, len(subjects), 1, "We only loaded one certificate from the file, len(subjects) should equal 1")
}

func TestAppendPEMFileToRootCAPoolFail(t *testing.T) {
	// Empty certpool
	var certPool *x509.CertPool

	// In this test we are going to pass a file that does not exist as value of second argument
	fakeCertificatesFile := "mockcertificates/NO-FILE.pem"
	certPool, err := AppendPEMFileToRootCAPool(certPool, fakeCertificatesFile)

	// Assert AppendPEMFileToRootCAPool correctly throws an error when trying to load an nonexisting file
	assert.Errorf(t, err, "AppendPEMFileToRootCAPool should throw an error by while loading fake file %s \n", fakeCertificatesFile)
}
