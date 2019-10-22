package ssl

import (
	//"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendPEMFileToRootCAPool(t *testing.T) {
	certPool := GetRootCAPool()
	subjects := certPool.Subjects()

	hardCodedSubNum := len(subjects)
	assert.True(t, hardCodedSubNum > 0)

	AppendPEMFileToRootCAPool(certPool, "mockcertificates/mock-certs.pem")
	subjects = certPool.Subjects()
	subNumIncludingFile := len(subjects)
	assert.True(t, subNumIncludingFile > hardCodedSubNum, "subNumIncludingFile should be greater than hardCodedSubNum")
}
