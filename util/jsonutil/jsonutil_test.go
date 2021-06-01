package jsonutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDropSingleElementAfterAnotherElement(t *testing.T) {
	inputData := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                    "test": 1,
                    "consented_providers": [1608,765,492,1365,5678,1545,2563,1411]
                }
            }`)
	res, err := DropElement(inputData, "consented_providers")

	expectedRes := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                    "test": 1
                }
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}

func TestDropSingleElementBeforeAnotherElement(t *testing.T) {
	inputData := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                    "consented_providers": [1608,765,492,1365,5678,1545,2563,1411],
                    "test": 1
                }
            }`)
	res, err := DropElement(inputData, "consented_providers")

	expectedRes := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                    "test": 1
                }
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}

func TestDropSingleElementSingleElement(t *testing.T) {
	inputData := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                    "consented_providers": [1608,765,492,1365,5678,1545,2563,1411]
                }
            }`)
	res, err := DropElement(inputData, "consented_providers")

	expectedRes := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                }
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}

func TestDropSingleElementSingleElementString(t *testing.T) {
	inputData := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                    "consented_providers": "test"
                }
            }`)
	res, err := DropElement(inputData, "consented_providers")

	expectedRes := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {
                }
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}

func TestDropParentElementBetweenTwoElements(t *testing.T) {
	inputData := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {"consented_providers": [1608,765,492,1365,5678,1545,2563,1411], "test": 1
                },"test": 123
            }`)
	res, err := DropElement(inputData, "consented_providers_settings")

	expectedRes := []byte(`{
                "consent": "TESTCONSENT","test": 123
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}

func TestDropParentElementBeforeElement(t *testing.T) {
	inputData := []byte(`{
                "consented_providers_settings": {"consented_providers": [1608,765,492,1365,5678,1545,2563,1411], "test": 1
                },"test": 123
            }`)
	res, err := DropElement(inputData, "consented_providers_settings")

	expectedRes := []byte(`{"test": 123
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}

func TestDropParentElementAfterElement(t *testing.T) {
	inputData := []byte(`{
                "consent": "TESTCONSENT",
                "consented_providers_settings": {"consented_providers": [1608,765,492,1365,5678,1545,2563,1411], "test": 1
                }
            }`)
	res, err := DropElement(inputData, "consented_providers_settings")

	expectedRes := []byte(`{
                "consent": "TESTCONSENT"
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}

func TestDropParentElementOnlyElement(t *testing.T) {
	inputData := []byte(`{
                "consented_providers_settings": {"consented_providers": [1608,765,492,1365,5678,1545,2563,1411], "test": 1
                }
            }`)
	res, err := DropElement(inputData, "consented_providers_settings")

	expectedRes := []byte(`{
            }`)

	assert.NoError(t, err, "Error should be nil")
	assert.Equal(t, expectedRes, res, "Result is incorrect")

}
