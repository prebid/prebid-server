package exchange

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/bidadjustment"
	"github.com/prebid/prebid-server/v3/config/util"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/experiment/adscert"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/version"

	"github.com/prebid/openrtb/v20/adcom1"
	nativeRequests "github.com/prebid/openrtb/v20/native1/request"
	nativeResponse "github.com/prebid/openrtb/v20/native1/response"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"golang.org/x/net/context/ctxhttp"
)

// AdaptedBidder defines the contract needed to participate in an Auction within an Exchange.
//
// This interface exists to help segregate core auction logic.
//
// Any logic which can be done _within a single Seat_ goes inside one of these.
// Any logic which _requires responses from all Seats_ goes inside the Exchange.
//
// This interface differs from adapters.Bidder to help minimize code duplication across the
// adapters.Bidder implementations.
type AdaptedBidder interface {
	// requestBid fetches bids for the given request.
	//
	// An AdaptedBidder *may* return two non-nil values here. Errors should describe situations which
	// make the bid (or no-bid) "less than ideal." Common examples include:
	//
	// 1. Connection issues.
	// 2. Imps with Media Types which this Bidder doesn't support.
	// 3. The Context timeout expired before all expected bids were returned.
	// 4. The Server sent back an unexpected Response, so some bids were ignored.
	//
	// Any errors will be user-facing in the API.
	// Error messages should help publishers understand what might account for "bad" bids.
	requestBid(ctx context.Context, bidderRequest BidderRequest, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, adsCertSigner adscert.Signer, bidRequestOptions bidRequestOptions, alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes, hookExecutor hookexecution.StageExecutor, ruleToAdjustments openrtb_ext.AdjustmentsByDealID) ([]*entities.PbsOrtbSeatBid, extraBidderRespInfo, []error)
}

// bidRequestOptions holds additional options for bid request execution to maintain clean code and reasonable number of parameters
type bidRequestOptions struct {
	accountDebugAllowed    bool
	headerDebugAllowed     bool
	addCallSignHeader      bool
	bidAdjustments         map[string]float64
	tmaxAdjustments        *TmaxAdjustmentsPreprocessed
	bidderRequestStartTime time.Time
	responseDebugAllowed   bool
}

type extraBidderRespInfo struct {
	respProcessingStartTime time.Time
	seatNonBidBuilder       SeatNonBidBuilder
}

type extraAuctionResponseInfo struct {
	fledge                  *openrtb_ext.Fledge
	bidsFound               bool
	bidderResponseStartTime time.Time
	seatNonBidBuilder       SeatNonBidBuilder
}

const ImpIdReqBody = "Stored bid response for impression id: "

// Possible values of compression types Prebid Server can support for bidder compression
const (
	Gzip string = "GZIP"
)

// AdaptBidder converts an adapters.Bidder into an exchange.AdaptedBidder.
//
// The name refers to the "Adapter" architecture pattern, and should not be confused with a Prebid "Adapter"
// (which is being phased out and replaced by Bidder for OpenRTB auctions)
func AdaptBidder(bidder adapters.Bidder, client *http.Client, cfg *config.Configuration, me metrics.MetricsEngine, name openrtb_ext.BidderName, debugInfo *config.DebugInfo, endpointCompression string) AdaptedBidder {
	return &BidderAdapter{
		Bidder:     bidder,
		BidderName: name,
		Client:     client,
		me:         me,
		config: bidderAdapterConfig{
			Debug:               cfg.Debug,
			DisableConnMetrics:  cfg.Metrics.Disabled.AdapterConnectionMetrics,
			DebugInfo:           config.DebugInfo{Allow: parseDebugInfo(debugInfo)},
			EndpointCompression: endpointCompression,
		},
	}
}

func parseDebugInfo(info *config.DebugInfo) bool {
	if info == nil {
		return true
	}
	return info.Allow
}

type BidderAdapter struct {
	Bidder     adapters.Bidder
	BidderName openrtb_ext.BidderName
	Client     *http.Client
	me         metrics.MetricsEngine
	config     bidderAdapterConfig
}

type bidderAdapterConfig struct {
	Debug               config.Debug
	DisableConnMetrics  bool
	DebugInfo           config.DebugInfo
	EndpointCompression string
}

func (bidder *BidderAdapter) requestBid(ctx context.Context, bidderRequest BidderRequest, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, adsCertSigner adscert.Signer, bidRequestOptions bidRequestOptions, alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes, hookExecutor hookexecution.StageExecutor, ruleToAdjustments openrtb_ext.AdjustmentsByDealID) ([]*entities.PbsOrtbSeatBid, extraBidderRespInfo, []error) {
	request := openrtb_ext.RequestWrapper{BidRequest: bidderRequest.BidRequest}
	reject := hookExecutor.ExecuteBidderRequestStage(&request, string(bidderRequest.BidderName))
	seatNonBidBuilder := SeatNonBidBuilder{}
	if reject != nil {
		return nil, extraBidderRespInfo{}, []error{reject}
	}

	var (
		reqData         []*adapters.RequestData
		errs            []error
		responseChannel chan *httpCallInfo
		extraRespInfo   extraBidderRespInfo
	)

	// rebuild request after modules execution
	request.RebuildRequest()
	bidderRequest.BidRequest = request.BidRequest

	//check if real request exists for this bidder or it only has stored responses
	dataLen := 0
	if len(bidderRequest.BidRequest.Imp) > 0 {
		// Reducing the amount of time bidders have to compensate for the processing time used by PBS to fetch a stored request (if needed), validate the OpenRTB request and split it into multiple requests sanitized for each bidder
		// As well as for the time needed by PBS to prepare the auction response
		if bidRequestOptions.tmaxAdjustments != nil && bidRequestOptions.tmaxAdjustments.IsEnforced {
			bidderRequest.BidRequest.TMax = getBidderTmax(&bidderTmaxCtx{ctx}, bidderRequest.BidRequest.TMax, *bidRequestOptions.tmaxAdjustments)
		}
		reqData, errs = bidder.Bidder.MakeRequests(bidderRequest.BidRequest, reqInfo)

		if len(reqData) == 0 {
			// If the adapter failed to generate both requests and errors, this is an error.
			if len(errs) == 0 {
				errs = append(errs, &errortypes.FailedToRequestBids{Message: "The adapter failed to generate any bid requests, but also failed to generate an error explaining why"})
			}
			return nil, extraBidderRespInfo{}, errs
		}
		xPrebidHeader := version.BuildXPrebidHeaderForRequest(bidderRequest.BidRequest, version.Ver)

		for i := 0; i < len(reqData); i++ {
			if reqData[i].Headers != nil {
				reqData[i].Headers = reqData[i].Headers.Clone()
			} else {
				reqData[i].Headers = http.Header{}
			}
			reqData[i].Headers.Add("X-Prebid", xPrebidHeader)
			if reqInfo.GlobalPrivacyControlHeader == "1" {
				reqData[i].Headers.Add("Sec-GPC", reqInfo.GlobalPrivacyControlHeader)
			}
			if bidRequestOptions.addCallSignHeader {
				startSignRequestTime := time.Now()
				signatureMessage, err := adsCertSigner.Sign(reqData[i].Uri, reqData[i].Body)
				bidder.me.RecordAdsCertSignTime(time.Since(startSignRequestTime))
				if err != nil {
					bidder.me.RecordAdsCertReq(false)
					errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("AdsCert signer is enabled but cannot sign the request: %s", err.Error())})
				}
				if err == nil && len(signatureMessage) > 0 {
					reqData[i].Headers.Add(adscert.SignHeader, signatureMessage)
					bidder.me.RecordAdsCertReq(true)
				}
			}

		}
		// Make any HTTP requests in parallel.
		// If the bidder only needs to make one, save some cycles by just using the current one.
		dataLen = len(reqData) + len(bidderRequest.BidderStoredResponses)
		responseChannel = make(chan *httpCallInfo, dataLen)
		if len(reqData) == 1 {
			responseChannel <- bidder.doRequest(ctx, reqData[0], bidRequestOptions.bidderRequestStartTime, bidRequestOptions.tmaxAdjustments)
		} else {
			for _, oneReqData := range reqData {
				go func(data *adapters.RequestData) {
					responseChannel <- bidder.doRequest(ctx, data, bidRequestOptions.bidderRequestStartTime, bidRequestOptions.tmaxAdjustments)
				}(oneReqData) // Method arg avoids a race condition on oneReqData
			}
		}
	}
	if len(bidderRequest.BidderStoredResponses) > 0 {
		//if stored bid responses are present - replace impIds and add them as is to responseChannel <- stored responses
		if responseChannel == nil {
			dataLen = dataLen + len(bidderRequest.BidderStoredResponses)
			responseChannel = make(chan *httpCallInfo, dataLen)
		}
		for impId, bidResp := range bidderRequest.BidderStoredResponses {
			go func(id string, resp json.RawMessage) {
				responseChannel <- prepareStoredResponse(id, resp)
			}(impId, bidResp)
		}
	}

	defaultCurrency := "USD"
	seatBidMap := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		bidderRequest.BidderName: {
			Bids:      make([]*entities.PbsOrtbBid, 0, dataLen),
			Currency:  defaultCurrency,
			HttpCalls: make([]*openrtb_ext.ExtHttpCall, 0, dataLen),
			Seat:      string(bidderRequest.BidderName),
		},
	}

	// If the bidder made multiple requests, we still want them to enter as many bids as possible...
	// even if the timeout occurs sometime halfway through.
	for i := 0; i < dataLen; i++ {
		httpInfo := <-responseChannel
		// If this is a test bid, capture debugging info from the requests.
		// Write debug data to ext in case if:
		// - headerDebugAllowed (debug override header specified correct) - it overrides all other debug restrictions
		// - account debug is allowed
		// - bidder debug is allowed
		if bidRequestOptions.headerDebugAllowed {
			seatBidMap[bidderRequest.BidderName].HttpCalls = append(seatBidMap[bidderRequest.BidderName].HttpCalls, makeExt(httpInfo))
		} else {
			if bidRequestOptions.accountDebugAllowed {
				if bidder.config.DebugInfo.Allow {
					seatBidMap[bidderRequest.BidderName].HttpCalls = append(seatBidMap[bidderRequest.BidderName].HttpCalls, makeExt(httpInfo))
				} else {
					debugDisabledWarning := errortypes.Warning{
						WarningCode: errortypes.BidderLevelDebugDisabledWarningCode,
						Message:     "debug turned off for bidder",
					}
					errs = append(errs, &debugDisabledWarning)
				}
			}
		}

		if httpInfo.err == nil {
			extraRespInfo.respProcessingStartTime = time.Now()
			bidResponse, moreErrs := bidder.Bidder.MakeBids(bidderRequest.BidRequest, httpInfo.request, httpInfo.response)
			errs = append(errs, moreErrs...)

			if bidResponse != nil {
				reject := hookExecutor.ExecuteRawBidderResponseStage(bidResponse, string(bidder.BidderName))
				if reject != nil {
					errs = append(errs, reject)
					continue
				}
				// Setup default currency as `USD` is not set in bid request nor bid response
				if bidResponse.Currency == "" {
					bidResponse.Currency = defaultCurrency
				}
				if len(bidderRequest.BidRequest.Cur) == 0 {
					bidderRequest.BidRequest.Cur = []string{defaultCurrency}
				}

				// Try to get a conversion rate
				// Try to get the first currency from request.cur having a match in the rate converter,
				// and use it as currency
				var conversionRate float64
				var err error
				for _, bidReqCur := range bidderRequest.BidRequest.Cur {
					if conversionRate, err = conversions.GetRate(bidResponse.Currency, bidReqCur); err == nil {
						seatBidMap[bidderRequest.BidderName].Currency = bidReqCur
						break
					}
				}

				// Only do this for request from mobile app
				if bidderRequest.BidRequest.App != nil {
					for i := 0; i < len(bidResponse.Bids); i++ {
						if bidResponse.Bids[i].BidType == openrtb_ext.BidTypeNative {
							nativeMarkup, moreErrs := addNativeTypes(bidResponse.Bids[i].Bid, bidderRequest.BidRequest)
							errs = append(errs, moreErrs...)

							if nativeMarkup != nil {
								markup, err := jsonutil.Marshal(*nativeMarkup)
								if err != nil {
									errs = append(errs, err)
								} else {
									bidResponse.Bids[i].Bid.AdM = string(markup)
								}
							}
						}
					}
				}

				// FLEDGE auctionconfig responses are sent separate from bids
				if bidResponse.FledgeAuctionConfigs != nil {
					if fledgeAuctionConfigs := seatBidMap[bidderRequest.BidderName].FledgeAuctionConfigs; fledgeAuctionConfigs != nil {
						seatBidMap[bidderRequest.BidderName].FledgeAuctionConfigs = append(fledgeAuctionConfigs, bidResponse.FledgeAuctionConfigs...)
					} else {
						seatBidMap[bidderRequest.BidderName].FledgeAuctionConfigs = bidResponse.FledgeAuctionConfigs
					}
				}

				if len(bidderRequest.BidderStoredResponses) > 0 {
					//set imp ids back to response for bids with stored responses
					for i := 0; i < len(bidResponse.Bids); i++ {
						if httpInfo.request.Uri == "" {
							reqBody := string(httpInfo.request.Body)
							re := regexp.MustCompile(ImpIdReqBody)
							reqBodySplit := re.Split(reqBody, -1)
							reqImpId := reqBodySplit[1]
							// replace impId if "replaceimpid" is true or not specified
							if bidderRequest.ImpReplaceImpId[reqImpId] {
								bidResponse.Bids[i].Bid.ImpID = reqImpId
							}
						}
					}
				}

				if err == nil {
					// Conversion rate found, using it for conversion
					for i := 0; i < len(bidResponse.Bids); i++ {

						bidderName := bidderRequest.BidderName
						if bidResponse.Bids[i].Seat != "" {
							bidderName = bidResponse.Bids[i].Seat
						}

						if valid, err := alternateBidderCodes.IsValidBidderCode(bidderRequest.BidderName.String(), bidderName.String()); !valid {
							if err != nil {
								err = &errortypes.Warning{
									WarningCode: errortypes.AlternateBidderCodeWarningCode,
									Message:     err.Error(),
								}
								errs = append(errs, err)
							}
							continue
						}

						adjustmentFactor := 1.0
						if givenAdjustment, ok := bidRequestOptions.bidAdjustments[(strings.ToLower(bidderName.String()))]; ok {
							adjustmentFactor = givenAdjustment
						} else if givenAdjustment, ok := bidRequestOptions.bidAdjustments[(strings.ToLower(bidderRequest.BidderName.String()))]; ok {
							adjustmentFactor = givenAdjustment
						}

						originalBidCpm := 0.0
						currencyAfterAdjustments := ""
						if bidResponse.Bids[i].Bid != nil {
							originalBidCpm = bidResponse.Bids[i].Bid.Price
							bidResponse.Bids[i].Bid.Price = bidResponse.Bids[i].Bid.Price * adjustmentFactor * conversionRate

							bidType := getBidTypeForAdjustments(bidResponse.Bids[i].BidType, bidResponse.Bids[i].Bid.ImpID, bidderRequest.BidRequest.Imp)
							bidResponse.Bids[i].Bid.Price, currencyAfterAdjustments = bidadjustment.Apply(ruleToAdjustments, bidResponse.Bids[i], bidderRequest.BidderName, seatBidMap[bidderRequest.BidderName].Currency, reqInfo, bidType)
						}

						if _, ok := seatBidMap[bidderName]; !ok {
							// Initalize seatBidMap entry as this is first extra bid with seat bidderName
							seatBidMap[bidderName] = &entities.PbsOrtbSeatBid{
								Bids:     make([]*entities.PbsOrtbBid, 0, dataLen),
								Currency: seatBidMap[bidderRequest.BidderName].Currency,
								// Do we need to fill httpCalls for this?. Can we refer one from adaptercode for debugging?
								HttpCalls: seatBidMap[bidderRequest.BidderName].HttpCalls,
								Seat:      bidderName.String(),
							}
						}

						seatBidMap[bidderName].Bids = append(seatBidMap[bidderName].Bids, &entities.PbsOrtbBid{
							Bid:            bidResponse.Bids[i].Bid,
							BidMeta:        bidResponse.Bids[i].BidMeta,
							BidType:        bidResponse.Bids[i].BidType,
							BidVideo:       bidResponse.Bids[i].BidVideo,
							DealPriority:   bidResponse.Bids[i].DealPriority,
							OriginalBidCPM: originalBidCpm,
							OriginalBidCur: bidResponse.Currency,
							AdapterCode:    bidderRequest.BidderCoreName,
						})
						seatBidMap[bidderName].Currency = currencyAfterAdjustments
					}
				} else {
					// If no conversions found, do not handle the bid
					errs = append(errs, err)
				}
			}
		} else {
			errs = append(errs, httpInfo.err)
			nonBidReason := httpInfoToNonBidReason(httpInfo)
			seatNonBidBuilder.rejectImps(httpInfo.request.ImpIDs, nonBidReason, string(bidderRequest.BidderName))
		}
	}

	seatBids := make([]*entities.PbsOrtbSeatBid, 0, len(seatBidMap))
	for _, seatBid := range seatBidMap {
		seatBids = append(seatBids, seatBid)
	}

	extraRespInfo.seatNonBidBuilder = seatNonBidBuilder
	return seatBids, extraRespInfo, errs
}

func addNativeTypes(bid *openrtb2.Bid, request *openrtb2.BidRequest) (*nativeResponse.Response, []error) {
	var errs []error
	var nativeMarkup nativeResponse.Response
	if err := jsonutil.UnmarshalValid(json.RawMessage(bid.AdM), &nativeMarkup); err != nil || len(nativeMarkup.Assets) == 0 {
		// Some bidders are returning non-IAB compliant native markup. In this case Prebid server will not be able to add types. E.g Facebook
		return nil, errs
	}

	nativeImp, err := getNativeImpByImpID(bid.ImpID, request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	var nativePayload nativeRequests.Request
	if err := jsonutil.UnmarshalValid(json.RawMessage((*nativeImp).Request), &nativePayload); err != nil {
		errs = append(errs, err)
	}

	for _, asset := range nativeMarkup.Assets {
		if err := setAssetTypes(asset, nativePayload); err != nil {
			errs = append(errs, err)
		}
	}

	return &nativeMarkup, errs
}

func setAssetTypes(asset nativeResponse.Asset, nativePayload nativeRequests.Request) error {
	if asset.Img != nil {
		if asset.ID == nil {
			return errors.New("Response Image asset doesn't have an ID")
		}
		if tempAsset, err := getAssetByID(*asset.ID, nativePayload.Assets); err == nil {
			if tempAsset.Img != nil {
				if tempAsset.Img.Type != 0 {
					asset.Img.Type = tempAsset.Img.Type
				}
			} else {
				return fmt.Errorf("Response has an Image asset with ID:%d present that doesn't exist in the request", *asset.ID)
			}
		} else {
			return err
		}
	}

	if asset.Data != nil {
		if asset.ID == nil {
			return errors.New("Response Data asset doesn't have an ID")
		}
		if tempAsset, err := getAssetByID(*asset.ID, nativePayload.Assets); err == nil {
			if tempAsset.Data != nil {
				if tempAsset.Data.Type != 0 {
					asset.Data.Type = tempAsset.Data.Type
				}
			} else {
				return fmt.Errorf("Response has a Data asset with ID:%d present that doesn't exist in the request", *asset.ID)
			}
		} else {
			return err
		}
	}
	return nil
}

func getNativeImpByImpID(impID string, request *openrtb2.BidRequest) (*openrtb2.Native, error) {
	for _, impInRequest := range request.Imp {
		if impInRequest.ID == impID && impInRequest.Native != nil {
			return impInRequest.Native, nil
		}
	}
	return nil, errors.New("Could not find native imp")
}

func getAssetByID(id int64, assets []nativeRequests.Asset) (nativeRequests.Asset, error) {
	for _, asset := range assets {
		if id == asset.ID {
			return asset, nil
		}
	}
	return nativeRequests.Asset{}, fmt.Errorf("Unable to find asset with ID:%d in the request", id)
}

var authorizationHeader = http.CanonicalHeaderKey("authorization")

func filterHeader(h http.Header) http.Header {
	clone := h.Clone()
	clone.Del(authorizationHeader)
	return clone
}

// makeExt transforms information about the HTTP call into the contract class for the PBS response.
func makeExt(httpInfo *httpCallInfo) *openrtb_ext.ExtHttpCall {
	ext := &openrtb_ext.ExtHttpCall{}

	if httpInfo != nil && httpInfo.request != nil {
		ext.Uri = httpInfo.request.Uri
		ext.RequestBody = string(httpInfo.request.Body)
		ext.RequestHeaders = filterHeader(httpInfo.request.Headers)

		if httpInfo.err == nil && httpInfo.response != nil {
			ext.ResponseBody = string(httpInfo.response.Body)
			ext.Status = httpInfo.response.StatusCode
		}
	}

	return ext
}

// doRequest makes a request, handles the response, and returns the data needed by the
// Bidder interface.
func (bidder *BidderAdapter) doRequest(ctx context.Context, req *adapters.RequestData, bidderRequestStartTime time.Time, tmaxAdjustments *TmaxAdjustmentsPreprocessed) *httpCallInfo {
	return bidder.doRequestImpl(ctx, req, glog.Warningf, bidderRequestStartTime, tmaxAdjustments)
}

func (bidder *BidderAdapter) doRequestImpl(ctx context.Context, req *adapters.RequestData, logger util.LogMsg, bidderRequestStartTime time.Time, tmaxAdjustments *TmaxAdjustmentsPreprocessed) *httpCallInfo {
	requestBody, err := getRequestBody(req, bidder.config.EndpointCompression)
	if err != nil {
		return &httpCallInfo{
			request: req,
			err:     err,
		}
	}
	httpReq, err := http.NewRequest(req.Method, req.Uri, requestBody)
	if err != nil {
		return &httpCallInfo{
			request: req,
			err:     err,
		}
	}
	httpReq.Header = req.Headers

	// If adapter connection metrics are not disabled, add the client trace
	// to get complete connection info into our metrics
	if !bidder.config.DisableConnMetrics {
		ctx = bidder.addClientTrace(ctx)
	}
	bidder.me.RecordOverheadTime(metrics.PreBidder, time.Since(bidderRequestStartTime))

	if tmaxAdjustments != nil && tmaxAdjustments.IsEnforced {
		if hasShorterDurationThanTmax(&bidderTmaxCtx{ctx}, *tmaxAdjustments) {
			bidder.me.RecordTMaxTimeout()
			return &httpCallInfo{
				request: req,
				err:     &errortypes.TmaxTimeout{Message: "exceeded tmax duration"},
			}
		}
	}

	httpCallStart := time.Now()
	httpResp, err := ctxhttp.Do(ctx, bidder.Client, httpReq)
	if err != nil {
		if err == context.DeadlineExceeded {
			err = &errortypes.Timeout{Message: err.Error()}
			var corebidder adapters.Bidder = bidder.Bidder
			// The bidder adapter normally stores an info-aware bidder (a bidder wrapper)
			// rather than the actual bidder. So we need to unpack that first.
			if b, ok := corebidder.(*adapters.InfoAwareBidder); ok {
				corebidder = b.Bidder
			}
			if tb, ok := corebidder.(adapters.TimeoutBidder); ok {
				// Toss the timeout notification call into a go routine, as we are out of time'
				// and cannot delay processing. We don't do anything result, as there is not much
				// we can do about a timeout notification failure. We do not want to get stuck in
				// a loop of trying to report timeouts to the timeout notifications.
				go bidder.doTimeoutNotification(tb, req, logger)
			}

		}
		return &httpCallInfo{
			request: req,
			err:     err,
		}
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return &httpCallInfo{
			request: req,
			err:     err,
		}
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 400 {
		err = &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Server responded with failure status: %d. Set request.test = 1 for debugging info.", httpResp.StatusCode),
		}
	}

	bidder.me.RecordBidderServerResponseTime(time.Since(httpCallStart))
	return &httpCallInfo{
		request: req,
		response: &adapters.ResponseData{
			StatusCode: httpResp.StatusCode,
			Body:       respBody,
			Headers:    httpResp.Header,
		},
		err: err,
	}
}

func (bidder *BidderAdapter) doTimeoutNotification(timeoutBidder adapters.TimeoutBidder, req *adapters.RequestData, logger util.LogMsg) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	toReq, errL := timeoutBidder.MakeTimeoutNotification(req)
	if toReq != nil && len(errL) == 0 {
		httpReq, err := http.NewRequest(toReq.Method, toReq.Uri, bytes.NewBuffer(toReq.Body))
		if err == nil {
			httpReq.Header = req.Headers
			httpResp, err := ctxhttp.Do(ctx, bidder.Client, httpReq)
			success := (err == nil && httpResp.StatusCode >= 200 && httpResp.StatusCode < 300)
			bidder.me.RecordTimeoutNotice(success)
			if bidder.config.Debug.TimeoutNotification.Log && !(bidder.config.Debug.TimeoutNotification.FailOnly && success) {
				var msg string
				if err == nil {
					msg = fmt.Sprintf("TimeoutNotification: status:(%d) body:%s", httpResp.StatusCode, string(toReq.Body))
				} else {
					msg = fmt.Sprintf("TimeoutNotification: error:(%s) body:%s", err.Error(), string(toReq.Body))
				}
				// If logging is turned on, and logging is not disallowed via FailOnly
				util.LogRandomSample(msg, logger, bidder.config.Debug.TimeoutNotification.SamplingRate)
			}
		} else {
			bidder.me.RecordTimeoutNotice(false)
			if bidder.config.Debug.TimeoutNotification.Log {
				msg := fmt.Sprintf("TimeoutNotification: Failed to make timeout request: method(%s), uri(%s), error(%s)", toReq.Method, toReq.Uri, err.Error())
				util.LogRandomSample(msg, logger, bidder.config.Debug.TimeoutNotification.SamplingRate)
			}
		}
	} else if bidder.config.Debug.TimeoutNotification.Log {
		reqJSON, err := jsonutil.Marshal(req)
		var msg string
		if err == nil {
			msg = fmt.Sprintf("TimeoutNotification: Failed to generate timeout request: error(%s), bidder request(%s)", errL[0].Error(), string(reqJSON))
		} else {
			msg = fmt.Sprintf("TimeoutNotification: Failed to generate timeout request: error(%s), bidder request marshal failed(%s)", errL[0].Error(), err.Error())
		}
		util.LogRandomSample(msg, logger, bidder.config.Debug.TimeoutNotification.SamplingRate)
	}

}

type httpCallInfo struct {
	request  *adapters.RequestData
	response *adapters.ResponseData
	err      error
}

// This function adds an httptrace.ClientTrace object to the context so, if connection with the bidder
// endpoint is established, we can keep track of whether the connection was newly created, reused, and
// the time from the connection request, to the connection creation.
func (bidder *BidderAdapter) addClientTrace(ctx context.Context) context.Context {
	var connStart, dnsStart, tlsStart time.Time

	trace := &httptrace.ClientTrace{
		// GetConn is called before a connection is created or retrieved from an idle pool
		GetConn: func(hostPort string) {
			connStart = time.Now()
		},
		// GotConn is called after a successful connection is obtained
		GotConn: func(info httptrace.GotConnInfo) {
			connWaitTime := time.Since(connStart)

			bidder.me.RecordAdapterConnections(bidder.BidderName, info.Reused, connWaitTime)
		},
		// DNSStart is called when a DNS lookup begins.
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		// DNSDone is called when a DNS lookup ends.
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsLookupTime := time.Since(dnsStart)

			bidder.me.RecordDNSTime(dnsLookupTime)
		},

		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},

		TLSHandshakeDone: func(tls.ConnectionState, error) {
			tlsHandshakeTime := time.Since(tlsStart)

			bidder.me.RecordTLSHandshakeTime(tlsHandshakeTime)
		},
	}
	return httptrace.WithClientTrace(ctx, trace)
}

func prepareStoredResponse(impId string, bidResp json.RawMessage) *httpCallInfo {
	//always one element in reqData because stored response is mapped to single imp
	body := fmt.Sprintf("%s%s", ImpIdReqBody, impId)
	reqDataForStoredResp := adapters.RequestData{
		Method: "POST",
		Uri:    "",
		Body:   []byte(body), //use it to pass imp id for stored resp
	}
	respData := &httpCallInfo{
		request: &reqDataForStoredResp,
		response: &adapters.ResponseData{
			StatusCode: 200,
			Body:       bidResp,
		},
		err: nil,
	}
	return respData
}

func getBidTypeForAdjustments(bidType openrtb_ext.BidType, impID string, imp []openrtb2.Imp) string {
	if bidType == openrtb_ext.BidTypeVideo {
		for _, imp := range imp {
			if imp.ID == impID {
				if imp.Video != nil && imp.Video.Plcmt == adcom1.VideoPlcmtAccompanyingContent {
					return "video-outstream"
				}
				break
			}
		}
		return "video-instream"
	}
	return string(bidType)
}

func hasShorterDurationThanTmax(ctx bidderTmaxContext, tmaxAdjustments TmaxAdjustmentsPreprocessed) bool {
	if tmaxAdjustments.IsEnforced {
		if deadline, ok := ctx.Deadline(); ok {
			overheadNS := time.Duration(tmaxAdjustments.BidderNetworkLatencyBuffer+tmaxAdjustments.PBSResponsePreparationDuration) * time.Millisecond
			bidderTmax := deadline.Add(-overheadNS)

			remainingDuration := ctx.Until(bidderTmax).Milliseconds()
			return remainingDuration < int64(tmaxAdjustments.BidderResponseDurationMin)
		}
	}
	return false
}

func getRequestBody(req *adapters.RequestData, endpointCompression string) (*bytes.Buffer, error) {
	switch strings.ToUpper(endpointCompression) {
	case Gzip:
		// Compress to GZIP
		b := bytes.NewBuffer(make([]byte, 0, len(req.Body)))

		w := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(w)

		w.Reset(b)
		_, err := w.Write(req.Body)
		if err != nil {
			return nil, err
		}
		err = w.Close()
		if err != nil {
			return nil, err
		}

		// Set Header
		req.Headers.Set("Content-Encoding", "gzip")

		return b, nil
	default:
		return bytes.NewBuffer(req.Body), nil
	}
}

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}
