package adservertargeting

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/prebid/openrtb/v17/openrtb2"
	"strings"
)

type respCache struct {
	// only holds upper level bid response data except of ext and bids
	bidRespData map[string]string
}

func (brh *respCache) GetRespData(bidderResp *openrtb2.BidResponse, field string) (string, error) {
	if len(brh.bidRespData) == 0 {
		// this code should be modified if there are changes in Op[enRtb format
		// it's up to date with OpenRTB 2.5 and 2.6
		brh.bidRespData = make(map[string]string)
		brh.bidRespData["id"] = bidderResp.ID
		brh.bidRespData["bidid"] = bidderResp.BidID
		brh.bidRespData["cur"] = bidderResp.Cur
		brh.bidRespData["customdata"] = bidderResp.CustomData
		brh.bidRespData["nbr"] = fmt.Sprint(bidderResp.NBR.Val())
	}

	value, exists := brh.bidRespData[strings.ToLower(field)]
	if exists {
		return value, nil
	} else {
		return "", errors.Errorf("key not found for field in bid response: %s", field)
	}
}
