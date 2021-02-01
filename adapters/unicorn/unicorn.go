package unicorn

import (
  "encoding/json"
  "fmt"
  "net/http"

  "github.com/buger/jsonparser"
  "github.com/mxmCherry/openrtb"
  "github.com/prebid/prebid-server/adapters"
  "github.com/prebid/prebid-server/config"
  "github.com/prebid/prebid-server/errortypes"
  "github.com/prebid/prebid-server/openrtb_ext"
)

const (
  unicornDefaultCurrency = "JPY"
  unicornAuctionType = 1
)

// UnicornAdapter describes a Smaato prebid server adapter.
type UnicornAdapter struct {
  endpoint string
}

type unicornImpExt struct {
  bidder unicornBidder `json:"bidder"`
}

type unicornBidder struct {
  accountID   int64  `json:"account_id"`
  publisherID int64  `json:"publisher_id"`
  mediaID     string `json:"media_id"`
  placementID string `json:"placement_id"`
}

type unicornAppExt struct {
  prebid unicornAppExtPrebid `json:"prebid"`
}

type unicornAppExtPrebid struct {
  version string `json:"version"`
  source string `json:"source"`
}

type unicornSourceExt struct {
  stype string `json:"stype"`
  bidder string `json"bidder"`
}

// Builder builds a new instance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
  bidder := &UnicornAdapter{
    endpoint: config.Endpoint,
  }
  return bidder, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *UnicornAdapter) MakeRequests(request *openrtb.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
  var extRegs openrtb_ext.ExtRegs
  if request.Regs != nil {
    if request.Regs.COPPA == 1 {
      return nil, []error{}
    }
    if err := json.Unmarshal(request.Regs.Ext, &extRegs); err == nil {
      if extRegs.GDPR != nil && (*extRegs.GDPR == 1) {
        return nil, []error{}
      }
      if extRegs.USPrivacy != "" {
        return nil, []error{}
      }
    }
  }

  request.AT = unicornAuctionType

  imp, err := getImps(request, requestInfo)
  if err != nil {
    return nil, []error{err}
  }

  request.Imp = imp
  request.Cur = []string{unicornDefaultCurrency}

  request.App.Ext, err = getAppExt(request)
  if err != nil {
    return nil, []error{err}
  }

  request.Source.Ext, err = getSourceExt()
  if err != nil {
    return nil, []error{err}
  }

  requestJSON, err := json.Marshal(request)
  if err != nil {
    return nil, []error{err}
  }

  requestData := &adapters.RequestData{
    Method:  "POST",
    Uri:     a.endpoint,
    Body:    requestJSON,
  }

  return []*adapters.RequestData{requestData}, nil
}

func getImps(request *openrtb.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]openrtb.Imp, error) {
  for i := 0; i < len(request.Imp); i++ {
    imp := request.Imp[i]

    placementID, err := jsonparser.GetString(request.Imp[0].Ext, "bidder", "placementId")
    if err != nil {
      placementID = imp.TagID
    }
    publisherID, err := jsonparser.GetInt(request.Imp[0].Ext, "bidder", "publisherId")
    if err != nil {
      publisherID = 0
    }
    mediaID, err := jsonparser.GetString(request.Imp[0].Ext, "bidder", "mediaId")
    if err != nil {
      mediaID = ""
    }
    accountID, err := jsonparser.GetInt(request.Imp[0].Ext, "bidder", "accountId")
    if err != nil {
      accountID = 0
    }

    secure := int8(1)
    imp.Secure = &secure
    imp.TagID = placementID
    impBidder := unicornBidder {
        accountID: accountID,
        publisherID: publisherID,
        mediaID: mediaID,
        placementID: placementID,
    }
    impExt := &unicornImpExt {
      bidder: impBidder,
    }
    imp.Ext, err = json.Marshal(impExt)
    if err != nil {
      return nil, err
    }
  }
  return request.Imp, nil
}

func getSourceExt() (json.RawMessage, error) {
  siteExt, err := json.Marshal(unicornSourceExt{
    stype: "prebid_uncn",
    bidder: "unicorn",
  })
  if err != nil {
    return nil, err
  }
  return siteExt, nil
}

func getAppExt(request *openrtb.BidRequest) (json.RawMessage, error) {
  appExtPrebid := unicornAppExtPrebid{
    version: request.App.Ver,
    source: "prebid-mobile",
  }
  appExt, err := json.Marshal(unicornAppExt{
    prebid: appExtPrebid,
  })
  if err != nil {
    return nil, err
  }
  return appExt, nil
}

// MakeBids unpacks the server's response into Bids.
func (a *UnicornAdapter) MakeBids(request *openrtb.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
  if responseData.StatusCode == http.StatusNoContent {
    return nil, nil
  }

  if responseData.StatusCode == http.StatusBadRequest {
    err := &errortypes.BadInput{
      Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
    }
    return nil, []error{err}
  }

  if responseData.StatusCode != http.StatusOK {
    err := &errortypes.BadServerResponse{
      Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
    }
    return nil, []error{err}
  }

  var response openrtb.BidResponse
  if err := json.Unmarshal(responseData.Body, &response); err != nil {
    return nil, []error{err}
  }

  bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
  bidResponse.Currency = response.Cur
  for _, seatBid := range response.SeatBid {
    for _, bid := range seatBid.Bid {
      b := &adapters.TypedBid{
        Bid:     &bid,
        BidType: "",
      }
      bidResponse.Bids = append(bidResponse.Bids, b)
    }
  }
  return bidResponse, nil
}