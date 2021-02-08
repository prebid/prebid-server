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
  unicornAuctionType     = 1
)

// UnicornAdapter describes a Smaato prebid server adapter.
type UnicornAdapter struct {
  endpoint string
}

// unicornImpExt is imp ext for UNICORN
type unicornImpExt struct {
  Bidder openrtb_ext.ExtImpUnicorn `json:"bidder"`
}

// unicornSourceExt is source ext for UNICORN
type unicornSourceExt struct {
  Stype  string `json:"stype"`
  Bidder string `json"bidder"`
}

// unicornExt is ext for UNICORN
type unicornExt struct {
  Prebid    *openrtb_ext.ExtImpPrebid `json:"prebid,omitempty"`
  AccountID int64                     `json:"accountId,omitempty"`
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
      return nil, []error{&errortypes.BadInput{
        Message: "COPPA is not supported",
      }}
    }
    if err := json.Unmarshal(request.Regs.Ext, &extRegs); err == nil {
      if extRegs.GDPR != nil && (*extRegs.GDPR == 1) {
        return nil, []error{&errortypes.BadInput{
          Message: "GDPR is not supported",
        }}
      }
      if extRegs.USPrivacy != "" {
        return nil, []error{&errortypes.BadInput{
          Message: "CCPA is not supported",
        }}
      }
    }
  }

  request.AT = unicornAuctionType

  imp, err := setImps(request, requestInfo)
  if err != nil {
    return nil, []error{err}
  }

  request.Imp = imp
  request.Cur = []string{unicornDefaultCurrency}

  request.Source.Ext, err = setSourceExt()
  if err != nil {
    return nil, []error{err}
  }

  request.Ext, err = setExt(request)
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
    Headers: getHeaders(request),
  }

  return []*adapters.RequestData{requestData}, nil
}

func getHeaders(request *openrtb.BidRequest) http.Header {
  headers := http.Header{}
  headers.Add("Content-Type", "application/json;charset=utf-8")
  headers.Add("Accept", "application/json")
  headers.Add("X-Openrtb-Version", "2.5")

  if request.Device != nil {
    if len(request.Device.UA) > 0 {
      headers.Add("User-Agent", request.Device.UA)
    }

    if len(request.Device.IPv6) > 0 {
      headers.Add("X-Forwarded-For", request.Device.IPv6)
    }

    if len(request.Device.IP) > 0 {
      headers.Add("X-Forwarded-For", request.Device.IP)
    }
  }

  return headers
}

func setImps(request *openrtb.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]openrtb.Imp, error) {
  for i := 0; i < len(request.Imp); i++ {
    imp := &request.Imp[i]

    var ext unicornImpExt
    err := json.Unmarshal(imp.Ext, &ext)

    if err != nil {
      return nil, &errortypes.BadInput{
        Message: fmt.Sprintf("Error while decoding imp[%d].ext: %s", i, err),
      }
    }

    var placementID string
    if ext.Bidder.PlacementID != "" {
      placementID = ext.Bidder.PlacementID
    } else {
      placementID, err = getStoredRequestImpID(imp)
      if err != nil {
        return nil, &errortypes.BadInput{
          Message: fmt.Sprintf("Error get StoredRequestImpID from imp[%d]: %s", i, err),
        }
      }
    }

    ext.Bidder.PlacementID = placementID

    imp.Ext, err = json.Marshal(ext)
    if err != nil {
      return nil, &errortypes.BadInput{
        Message: fmt.Sprintf("Error while encoding imp[%d].ext: %s", i, err),
      }
    }

    secure := int8(1)
    imp.Secure = &secure
    imp.TagID = placementID
  }
  return request.Imp, nil
}

func getStoredRequestImpID(imp *openrtb.Imp) (string, error) {
  var impExt map[string]json.RawMessage

  err := json.Unmarshal(imp.Ext, &impExt)

  if err != nil {
    return "", fmt.Errorf("Error while decoding ext because: %s", err)
  }

  rawPrebidExt, ok := impExt[openrtb_ext.PrebidExtKey]

  if !ok {
    return "", fmt.Errorf("ext.prebid is null")
  }

  var prebidExt openrtb_ext.ExtImpPrebid

  err = json.Unmarshal(rawPrebidExt, &prebidExt)

  if err != nil {
    return "", fmt.Errorf("cannot decoding ext.prebid because: %s", err)
  }

  if prebidExt.StoredRequest == nil {
    return "", fmt.Errorf("ext.prebid.storedrequest is null")
  }

  return prebidExt.StoredRequest.ID, nil
}

func setSourceExt() (json.RawMessage, error) {
  siteExt, err := json.Marshal(unicornSourceExt{
    Stype:  "prebid_server_uncn",
    Bidder: "unicorn",
  })
  if err != nil {
    return nil, &errortypes.BadInput{
      Message: fmt.Sprintf("Error while encoding source.ext, err: %s", err),
    }
  }
  return siteExt, nil
}

func setExt(request *openrtb.BidRequest) (json.RawMessage, error) {
  accountID, err := jsonparser.GetInt(request.Imp[0].Ext, "bidder", "accountId")
  if err != nil {
    accountID = 0
  }
  var decodedExt *unicornExt
  err = json.Unmarshal(request.Ext, &decodedExt)
  if err != nil {
    decodedExt = &unicornExt{
      Prebid: nil,
    }
  }
  decodedExt.AccountID = accountID

  ext, err := json.Marshal(decodedExt)
  if err != nil {
    return nil, &errortypes.BadInput{
      Message: fmt.Sprintf("Error while encoding ext, err: %s", err),
    }
  }
  return ext, nil
}

// MakeBids unpacks the server's response into Bids.
func (a *UnicornAdapter) MakeBids(request *openrtb.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {

  if responseData.StatusCode == http.StatusNoContent {
    return nil, nil
  }

  if responseData.StatusCode == http.StatusBadRequest {
    err := &errortypes.BadInput{
      Message: "Unexpected http status code: 400",
    }
    return nil, []error{err}
  }

  if responseData.StatusCode != http.StatusOK {
    err := &errortypes.BadServerResponse{
      Message: fmt.Sprintf("Unexpected http status code: %d", responseData.StatusCode),
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
      var bidType openrtb_ext.BidType
      for _, imp := range request.Imp {
        if imp.ID == bid.ImpID {
          if imp.Banner != nil {
            bidType = openrtb_ext.BidTypeBanner
          }
          if imp.Native != nil {
            bidType = openrtb_ext.BidTypeNative
          }
        }
      }
      b := &adapters.TypedBid{
        Bid:     &bid,
        BidType: bidType,
      }
      bidResponse.Bids = append(bidResponse.Bids, b)
    }
  }
  return bidResponse, nil
}