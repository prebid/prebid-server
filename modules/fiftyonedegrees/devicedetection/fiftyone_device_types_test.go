package devicedetection

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/stretchr/testify/assert"
)

func TestFiftyOneDtToRTB(t *testing.T) {
	cases := []struct {
		fiftyOneDt string
		rtbDt      adcom1.DeviceType
	}{
		{
			fiftyOneDt: "Phone",
			rtbDt:      adcom1.DevicePhone,
		},
		{
			fiftyOneDt: "Console",
			rtbDt:      adcom1.DeviceSetTopBox,
		},
		{
			fiftyOneDt: "Desktop",
			rtbDt:      adcom1.DevicePC,
		},
		{
			fiftyOneDt: "EReader",
			rtbDt:      adcom1.DevicePC,
		},
		{
			fiftyOneDt: "IoT",
			rtbDt:      adcom1.DeviceConnected,
		},
		{
			fiftyOneDt: "Kiosk",
			rtbDt:      adcom1.DeviceOOH,
		},
		{
			fiftyOneDt: "MediaHub",
			rtbDt:      adcom1.DeviceSetTopBox,
		},
		{
			fiftyOneDt: "Mobile",
			rtbDt:      adcom1.DeviceMobile,
		},
		{
			fiftyOneDt: "Router",
			rtbDt:      adcom1.DeviceConnected,
		},
		{
			fiftyOneDt: "SmallScreen",
			rtbDt:      adcom1.DeviceConnected,
		},
		{
			fiftyOneDt: "SmartPhone",
			rtbDt:      adcom1.DevicePhone,
		},
		{
			fiftyOneDt: "SmartSpeaker",
			rtbDt:      adcom1.DeviceConnected,
		},
		{
			fiftyOneDt: "SmartWatch",
			rtbDt:      adcom1.DeviceConnected,
		},
		{
			fiftyOneDt: "Tablet",
			rtbDt:      adcom1.DeviceTablet,
		},
		{
			fiftyOneDt: "Tv",
			rtbDt:      adcom1.DeviceTV,
		},
		{
			fiftyOneDt: "Vehicle Display",
			rtbDt:      adcom1.DevicePC,
		},
		{
			fiftyOneDt: "Unknown",
			rtbDt:      adcom1.DevicePC,
		},
	}

	for _, c := range cases {
		t.Run(c.fiftyOneDt, func(t *testing.T) {
			assert.Equal(t, c.rtbDt, fiftyOneDtToRTB(c.fiftyOneDt))
		})
	}
}
