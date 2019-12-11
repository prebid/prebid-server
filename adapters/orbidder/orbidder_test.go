package orbidder

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalOrbidderExtImp(t *testing.T) {
	ext := json.RawMessage(`{"orbidder":{"accountId":"orbidder-test", "placementId":"center-banner"}}`)

	UnmarshalOrbidderExtImp(ext)
}
