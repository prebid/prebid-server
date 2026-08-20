package aps

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"strconv"

	"github.com/buger/jsonparser"
	jsoniter "github.com/json-iterator/go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
)

func ApplyAPSResponse(rctx models.RequestCtx, bidResponse *openrtb2.BidResponse) *openrtb2.BidResponse {
	if rctx.Endpoint != models.EndpointAPS || bidResponse.NBR != nil || rctx.APS.Reject {
		return bidResponse
	}

	bids := getBids(rctx, bidResponse)
	if len(bids) == 0 {
		return bidResponse
	}

	*bidResponse = openrtb2.BidResponse{
		ID:    bidResponse.ID,
		BidID: bidResponse.SeatBid[0].Bid[0].ID,
		Cur:   bidResponse.Cur,
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: bids,
			},
		},
	}

	return bidResponse
}

func getBids(rctx models.RequestCtx, bidResponse *openrtb2.BidResponse) []openrtb2.Bid {
	if len(bidResponse.SeatBid) == 0 || len(bidResponse.SeatBid[0].Bid) == 0 {
		return nil
	}

	applyAPSBidExpIfMissing(rctx, bidResponse)

	if err := setBidResponseExtForAdm(bidResponse, rctx.PubIDStr, rctx.ProfileIDStr); err != nil {
		return nil
	}

	serializedResponse, err := jsoniter.Marshal(bidResponse)
	if err != nil {
		return nil
	}

	// Compress the serialized response using gzip with fallback to original
	compressedResponse := compressResponse(serializedResponse)

	bid := bidResponse.SeatBid[0].Bid[0]

	//setting complete compressed bid responnse as signal data in the bid.AdM field
	bid.AdM = string(compressedResponse)

	if len(bid.Ext) == 0 {
		bid.Ext = []byte(`{}`)
	}
	bid.Ext, _ = jsonparser.Set(bid.Ext, []byte(`""`), "sdkbridge", "placementId")

	return []openrtb2.Bid{bid}
}

func setBidResponseExtForAdm(bidResponse *openrtb2.BidResponse, publisherID, profileID string) error {
	if len(bidResponse.Ext) == 0 {
		bidResponse.Ext = []byte(`{}`)
	}

	var err error
	bidResponse.Ext, err = jsonparser.Set(bidResponse.Ext, []byte(strconv.Quote(publisherID)), "publisherid")
	if err != nil {
		return err
	}
	bidResponse.Ext, err = jsonparser.Set(bidResponse.Ext, []byte(profileID), "profileid")
	return err
}

// compressResponse attempts to gzip compress and base64 encode input data.
// If compression fails, it returns original uncompressed data base64 encoded.
func compressResponse(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(data)

	if err != nil {
		// If gzip compression fails, return original data base64 encoded
		return []byte(base64.StdEncoding.EncodeToString(data))
	}
	err = gz.Close()
	if err != nil {
		// If closing gzip writer fails, return original data base64 encoded
		return []byte(base64.StdEncoding.EncodeToString(data))
	}

	return []byte(base64.StdEncoding.EncodeToString(buf.Bytes()))
}

func SetAPSResponseReject(rctx models.RequestCtx, bidResponse *openrtb2.BidResponse) bool {
	if rctx.Endpoint != models.EndpointAPS {
		return false
	}

	reject := false
	if bidResponse.NBR != nil {
		if !rctx.Debug {
			reject = true
		}
	} else if len(bidResponse.SeatBid) == 0 || len(bidResponse.SeatBid[0].Bid) == 0 {
		reject = true
	}
	return reject
}
