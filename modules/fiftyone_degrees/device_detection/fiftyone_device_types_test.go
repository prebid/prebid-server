package device_detection

import (
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFiftyOneToRTB(t *testing.T) {
	cases := []struct {
		fiftyOneDt string
		rtbDt      adcom1.DeviceType
	}{
		{
			fiftyOneDt: "Desktop",
			rtbDt:      adcom1.DevicePC,
		},
		{
			fiftyOneDt: "SmartPhone",
			rtbDt:      adcom1.DeviceMobile,
		},
		{
			fiftyOneDt: "Tablet",
			rtbDt:      adcom1.DeviceTablet,
		},
		{
			fiftyOneDt: "Unknown",
			rtbDt:      adcom1.DevicePC,
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.rtbDt, fiftyOneDtToRTB(c.fiftyOneDt))
	}
}
