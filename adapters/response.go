package adapters

import (
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/v3/errortypes"
)

func CheckResponseStatusCodeForErrors(response *ResponseData) error {
	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}
	}

	if response.StatusCode != http.StatusOK {
		return &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}
	}

	return nil
}

func IsResponseStatusCodeNoContent(response *ResponseData) bool {
	return response.StatusCode == http.StatusNoContent
}
