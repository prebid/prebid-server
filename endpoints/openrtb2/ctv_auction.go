package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/endpoints/events"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PubMatic-OpenWrap/etree"
	"github.com/buger/jsonparser"
	uuid "github.com/gofrs/uuid"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/combination"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/constant"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/impressions"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/response"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/types"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/util"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/iputil"
)

//CTV Specific Endpoint
type ctvEndpointDeps struct {
	endpointDeps
	request        *openrtb2.BidRequest
	reqExt         *openrtb_ext.ExtRequestAdPod
	impData        []*types.ImpData
	videoSeats     []*openrtb2.SeatBid //stores pure video impression bids
	impIndices     map[string]int
	isAdPodRequest bool

	//Prebid Specific
	ctx    context.Context
	labels metrics.Labels
}

//NewCTVEndpoint new ctv endpoint object
func NewCTVEndpoint(
	ex exchange.Exchange,
	validator openrtb_ext.BidderParamValidator,
	requestsByID stored_requests.Fetcher,
	videoFetcher stored_requests.Fetcher,
	accounts stored_requests.AccountFetcher,
	//categories stored_requests.CategoryFetcher,
	cfg *config.Configuration,
	met metrics.MetricsEngine,
	pbsAnalytics analytics.PBSAnalyticsModule,
	disabledBidders map[string]string,
	defReqJSON []byte,
	bidderMap map[string]openrtb_ext.BidderName) (httprouter.Handle, error) {

	if ex == nil || validator == nil || requestsByID == nil || accounts == nil || cfg == nil || met == nil {
		return nil, errors.New("NewCTVEndpoint requires non-nil arguments.")
	}
	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	ipValidator := iputil.PublicNetworkIPValidator{
		IPv4PrivateNetworks: cfg.RequestValidation.IPv4PrivateNetworksParsed,
		IPv6PrivateNetworks: cfg.RequestValidation.IPv6PrivateNetworksParsed,
	}

	return httprouter.Handle((&ctvEndpointDeps{
		endpointDeps: endpointDeps{
			ex,
			validator,
			requestsByID,
			videoFetcher,
			accounts,
			cfg,
			met,
			pbsAnalytics,
			disabledBidders,
			defRequest,
			defReqJSON,
			bidderMap,
			nil,
			nil,
			ipValidator,
		},
	}).CTVAuctionEndpoint), nil
}

func (deps *ctvEndpointDeps) CTVAuctionEndpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer util.TimeTrack(time.Now(), "CTVAuctionEndpoint")

	var request *openrtb2.BidRequest
	var response *openrtb2.BidResponse
	var err error
	var errL []error

	ao := analytics.AuctionObject{
		Status: http.StatusOK,
		Errors: make([]error, 0),
	}

	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.
	start := time.Now()
	//Prebid Stats
	deps.labels = metrics.Labels{
		Source:        metrics.DemandUnknown,
		RType:         metrics.ReqTypeVideo,
		PubID:         metrics.PublisherUnknown,
		CookieFlag:    metrics.CookieFlagUnknown,
		RequestStatus: metrics.RequestStatusOK,
	}
	defer func() {
		deps.metricsEngine.RecordRequest(deps.labels)
		deps.metricsEngine.RecordRequestTime(deps.labels, time.Since(start))
		deps.analytics.LogAuctionObject(&ao)
	}()

	//Parse ORTB Request and do Standard Validation
	request, errL = deps.parseRequest(r)
	if errortypes.ContainsFatalError(errL) && writeError(errL, w, &deps.labels) {
		return
	}

	util.JLogf("Original BidRequest", request) //TODO: REMOVE LOG

	//init
	deps.init(request)

	//Set Default Values
	deps.setDefaultValues()
	util.JLogf("Extensions Request Extension", deps.reqExt)
	util.JLogf("Extensions ImpData", deps.impData)

	//Validate CTV BidRequest
	if err := deps.validateBidRequest(); err != nil {
		errL = append(errL, err...)
		writeError(errL, w, &deps.labels)
		return
	}

	if deps.isAdPodRequest {
		//Create New BidRequest
		request = deps.createBidRequest(request)
		util.JLogf("CTV BidRequest", request) //TODO: REMOVE LOG
	}

	//Parsing Cookies and Set Stats
	usersyncs := usersync.ParsePBSCookieFromRequest(r, &(deps.cfg.HostCookie))
	if request.App != nil {
		deps.labels.Source = metrics.DemandApp
		deps.labels.RType = metrics.ReqTypeVideo
		deps.labels.PubID = getAccountID(request.App.Publisher)
	} else { //request.Site != nil
		deps.labels.Source = metrics.DemandWeb
		if usersyncs.LiveSyncCount() == 0 {
			deps.labels.CookieFlag = metrics.CookieFlagNo
		} else {
			deps.labels.CookieFlag = metrics.CookieFlagYes
		}
		deps.labels.PubID = getAccountID(request.Site.Publisher)
	}

	deps.ctx = context.Background()

	// Look up account now that we have resolved the pubID value
	account, acctIDErrs := accountService.GetAccount(deps.ctx, deps.cfg, deps.accounts, deps.labels.PubID)
	if len(acctIDErrs) > 0 {
		errL = append(errL, acctIDErrs...)
		writeError(errL, w, &deps.labels)
		return
	}

	//Setting Timeout for Request
	timeout := deps.cfg.AuctionTimeouts.LimitAuctionTimeout(time.Duration(request.TMax) * time.Millisecond)
	if timeout > 0 {
		var cancel context.CancelFunc
		deps.ctx, cancel = context.WithDeadline(deps.ctx, start.Add(timeout))
		defer cancel()
	}

	response, err = deps.holdAuction(request, usersyncs, account, start)

	ao.Request = request
	ao.Response = response
	if err != nil || nil == response {
		deps.labels.RequestStatus = metrics.RequestStatusErr
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/video Critical error: %v", err)
		ao.Status = http.StatusInternalServerError
		ao.Errors = append(ao.Errors, err)
		return
	}
	util.JLogf("BidResponse", response) //TODO: REMOVE LOG

	if deps.isAdPodRequest {
		//Validate Bid Response
		if err := deps.validateBidResponse(request, response); err != nil {
			errL = append(errL, err)
			writeError(errL, w, &deps.labels)
			return
		}

		//Create Impression Bids
		deps.getBids(response)

		//Do AdPod Exclusions
		bids := deps.doAdPodExclusions()

		//Create Bid Response
		response = deps.createBidResponse(response, bids)
		util.JLogf("CTV BidResponse", response) //TODO: REMOVE LOG
	}

	// Response Generation
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	// Fixes #328
	w.Header().Set("Content-Type", "application/json")

	// If an error happens when encoding the response, there isn't much we can do.
	// If we've sent _any_ bytes, then Go would have sent the 200 status code first.
	// That status code can't be un-sent... so the best we can do is log the error.
	if err := enc.Encode(response); err != nil {
		deps.labels.RequestStatus = metrics.RequestStatusNetworkErr
		ao.Errors = append(ao.Errors, fmt.Errorf("/openrtb2/video Failed to send response: %v", err))
	}
}

func (deps *ctvEndpointDeps) holdAuction(request *openrtb2.BidRequest, usersyncs *usersync.PBSCookie, account *config.Account, startTime time.Time) (*openrtb2.BidResponse, error) {
	defer util.TimeTrack(time.Now(), fmt.Sprintf("Tid:%v CTVHoldAuction", deps.request.ID))

	//Hold OpenRTB Standard Auction
	if len(request.Imp) == 0 {
		//Dummy Response Object
		return &openrtb2.BidResponse{ID: request.ID}, nil
	}

	auctionRequest := exchange.AuctionRequest{
		BidRequest:   request,
		Account:      *account,
		UserSyncs:    usersyncs,
		RequestType:  deps.labels.RType,
		StartTime:    startTime,
		LegacyLabels: deps.labels,
	}

	return deps.ex.HoldAuction(deps.ctx, auctionRequest, nil)
}

/********************* BidRequest Processing *********************/

func (deps *ctvEndpointDeps) init(req *openrtb2.BidRequest) {
	deps.request = req
	deps.impData = make([]*types.ImpData, len(req.Imp))
	deps.impIndices = make(map[string]int, len(req.Imp))

	for i := range req.Imp {
		deps.impIndices[req.Imp[i].ID] = i
		deps.impData[i] = &types.ImpData{}
	}
}

func (deps *ctvEndpointDeps) readVideoAdPodExt() (err []error) {
	for index, imp := range deps.request.Imp {
		if nil != imp.Video {
			vidExt := openrtb_ext.ExtVideoAdPod{}
			if len(imp.Video.Ext) > 0 {
				errL := json.Unmarshal(imp.Video.Ext, &vidExt)
				if nil != err {
					err = append(err, errL)
					continue
				}

				imp.Video.Ext = jsonparser.Delete(imp.Video.Ext, constant.CTVAdpod)
				imp.Video.Ext = jsonparser.Delete(imp.Video.Ext, constant.CTVOffset)
				if string(imp.Video.Ext) == `{}` {
					imp.Video.Ext = nil
				}
			}

			if nil == vidExt.AdPod {
				if nil == deps.reqExt {
					continue
				}
				vidExt.AdPod = &openrtb_ext.VideoAdPod{}
			}

			//Use Request Level Parameters
			if nil != deps.reqExt {
				vidExt.AdPod.Merge(&deps.reqExt.VideoAdPod)
			}

			//Set Default Values
			vidExt.SetDefaultValue()
			vidExt.AdPod.SetDefaultAdDurations(imp.Video.MinDuration, imp.Video.MaxDuration)

			deps.impData[index].VideoExt = &vidExt
		}
	}
	return err
}

func (deps *ctvEndpointDeps) readRequestExtension() (err []error) {
	if len(deps.request.Ext) > 0 {

		//TODO: use jsonparser library for get adpod and remove that key
		extAdPod, jsonType, _, errL := jsonparser.Get(deps.request.Ext, constant.CTVAdpod)

		if nil != errL {
			//parsing error
			if jsonparser.NotExist != jsonType {
				//assuming key not present
				err = append(err, errL)
				return
			}
		} else {
			deps.reqExt = &openrtb_ext.ExtRequestAdPod{}

			if errL := json.Unmarshal(extAdPod, deps.reqExt); nil != errL {
				err = append(err, errL)
				return
			}

			deps.reqExt.SetDefaultValue()

			//removing key from extensions
			deps.request.Ext = jsonparser.Delete(deps.request.Ext, constant.CTVAdpod)
			if string(deps.request.Ext) == `{}` {
				deps.request.Ext = nil
			}
		}
	}

	return
}

func (deps *ctvEndpointDeps) readExtensions() (err []error) {
	if errL := deps.readRequestExtension(); nil != errL {
		err = append(err, errL...)
	}

	if errL := deps.readVideoAdPodExt(); nil != errL {
		err = append(err, errL...)
	}
	return err
}

func (deps *ctvEndpointDeps) setIsAdPodRequest() {
	deps.isAdPodRequest = false
	for _, data := range deps.impData {
		if nil != data.VideoExt && nil != data.VideoExt.AdPod {
			deps.isAdPodRequest = true
			break
		}
	}
}

//setDefaultValues will set adpod and other default values
func (deps *ctvEndpointDeps) setDefaultValues() {
	//read and set extension values
	deps.readExtensions()

	//set request is adpod request or normal request
	deps.setIsAdPodRequest()
}

//validateBidRequest will validate AdPod specific mandatory Parameters and returns error
func (deps *ctvEndpointDeps) validateBidRequest() (err []error) {
	//validating video extension adpod configurations
	if nil != deps.reqExt {
		err = deps.reqExt.Validate()
	}

	for index, imp := range deps.request.Imp {
		if nil != imp.Video && nil != deps.impData[index].VideoExt {
			ext := deps.impData[index].VideoExt
			if errL := ext.Validate(); nil != errL {
				err = append(err, errL...)
			}

			if nil != ext.AdPod {
				if errL := ext.AdPod.ValidateAdPodDurations(imp.Video.MinDuration, imp.Video.MaxDuration, imp.Video.MaxExtended); nil != errL {
					err = append(err, errL...)
				}
			}
		}

	}
	return
}

/********************* Creating CTV BidRequest *********************/

//createBidRequest will return new bid request with all things copy from bid request except impression objects
func (deps *ctvEndpointDeps) createBidRequest(req *openrtb2.BidRequest) *openrtb2.BidRequest {
	ctvRequest := *req

	//get configurations for all impressions
	deps.getAllAdPodImpsConfigs()

	//createImpressions
	ctvRequest.Imp = deps.createImpressions()

	//TODO: remove adpod extension if not required to send further
	return &ctvRequest
}

//getAllAdPodImpsConfigs will return all impression adpod configurations
func (deps *ctvEndpointDeps) getAllAdPodImpsConfigs() {
	for index, imp := range deps.request.Imp {
		if nil == imp.Video || nil == deps.impData[index].VideoExt || nil == deps.impData[index].VideoExt.AdPod {
			continue
		}
		deps.impData[index].Config = deps.getAdPodImpsConfigs(&imp, deps.impData[index].VideoExt.AdPod)
		if 0 == len(deps.impData[index].Config) {
			errorCode := new(int)
			*errorCode = 101
			deps.impData[index].ErrorCode = errorCode
		}
	}
}

//getAdPodImpsConfigs will return number of impressions configurations within adpod
func (deps *ctvEndpointDeps) getAdPodImpsConfigs(imp *openrtb2.Imp, adpod *openrtb_ext.VideoAdPod) []*types.ImpAdPodConfig {
	selectedAlgorithm := impressions.MinMaxAlgorithm
	labels := metrics.PodLabels{AlgorithmName: impressions.MonitorKey[selectedAlgorithm], NoOfImpressions: new(int)}

	// monitor
	start := time.Now()
	impGen := impressions.NewImpressions(imp.Video.MinDuration, imp.Video.MaxDuration, adpod, selectedAlgorithm)
	impRanges := impGen.Get()
	*labels.NoOfImpressions = len(impRanges)
	deps.metricsEngine.RecordPodImpGenTime(labels, start)

	config := make([]*types.ImpAdPodConfig, len(impRanges))
	for i, value := range impRanges {
		config[i] = &types.ImpAdPodConfig{
			ImpID:          util.GetCTVImpressionID(imp.ID, i+1),
			MinDuration:    value[0],
			MaxDuration:    value[1],
			SequenceNumber: int8(i + 1), /* Must be starting with 1 */
		}
	}
	return config[:]
}

//createImpressions will create multiple impressions based on adpod configurations
func (deps *ctvEndpointDeps) createImpressions() []openrtb2.Imp {
	impCount := 0
	for _, imp := range deps.impData {
		if nil == imp.ErrorCode {
			if len(imp.Config) == 0 {
				impCount = impCount + 1
			} else {
				impCount = impCount + len(imp.Config)
			}
		}
	}

	count := 0
	imps := make([]openrtb2.Imp, impCount)
	for index, imp := range deps.request.Imp {
		if nil == deps.impData[index].ErrorCode {
			adPodConfig := deps.impData[index].Config
			if len(adPodConfig) == 0 {
				//non adpod request it will be normal video impression
				imps[count] = imp
				count++
			} else {
				//for adpod request it will create new impression based on configurations
				for _, config := range adPodConfig {
					imps[count] = *(newImpression(&imp, config))
					count++
				}
			}
		}
	}
	return imps[:]
}

//newImpression will clone existing impression object and create video object with ImpAdPodConfig.
func newImpression(imp *openrtb2.Imp, config *types.ImpAdPodConfig) *openrtb2.Imp {
	video := *imp.Video
	video.MinDuration = config.MinDuration
	video.MaxDuration = config.MaxDuration
	video.Sequence = config.SequenceNumber
	video.MaxExtended = 0
	//TODO: remove video adpod extension if not required

	newImp := *imp
	newImp.ID = config.ImpID
	//newImp.BidFloor = 0
	newImp.Video = &video
	return &newImp
}

/********************* Prebid BidResponse Processing *********************/

//validateBidResponse
func (deps *ctvEndpointDeps) validateBidResponse(req *openrtb2.BidRequest, resp *openrtb2.BidResponse) error {
	//remove bids withoug cat and adomain

	return nil
}

//getBids reads bids from bidresponse object
func (deps *ctvEndpointDeps) getBids(resp *openrtb2.BidResponse) {
	var vseat *openrtb2.SeatBid
	result := make(map[string]*types.AdPodBid)

	for i := range resp.SeatBid {
		seat := resp.SeatBid[i]
		vseat = nil

		for j := range seat.Bid {
			bid := &seat.Bid[j]

			if len(bid.ID) == 0 {
				bidID, err := uuid.NewV4()
				if nil != err {
					continue
				}
				bid.ID = bidID.String()
			}

			if bid.Price == 0 {
				//filter invalid bids
				continue
			}

			originalImpID, sequenceNumber := deps.getImpressionID(bid.ImpID)
			if sequenceNumber < 0 {
				continue
			}

			value, err := util.GetTargeting(openrtb_ext.HbCategoryDurationKey, openrtb_ext.BidderName(seat.Seat), *bid)
			if nil == err {
				// ignore error
				addTargetingKey(bid, openrtb_ext.HbCategoryDurationKey, value)
			}

			value, err = util.GetTargeting(openrtb_ext.HbpbConstantKey, openrtb_ext.BidderName(seat.Seat), *bid)
			if nil == err {
				// ignore error
				addTargetingKey(bid, openrtb_ext.HbpbConstantKey, value)
			}

			index := deps.impIndices[originalImpID]
			if len(deps.impData[index].Config) == 0 {
				//adding pure video bids
				if vseat == nil {
					vseat = &openrtb2.SeatBid{
						Seat:  seat.Seat,
						Group: seat.Group,
						Ext:   seat.Ext,
					}
					deps.videoSeats = append(deps.videoSeats, vseat)
				}
				vseat.Bid = append(vseat.Bid, *bid)
			} else {
				//reading extension, ingorning parsing error
				ext := openrtb_ext.ExtBid{}
				if nil != bid.Ext {
					json.Unmarshal(bid.Ext, &ext)
				}

				//Adding adpod bids
				impBids, ok := result[originalImpID]
				if !ok {
					impBids = &types.AdPodBid{
						OriginalImpID: originalImpID,
						SeatName:      constant.PrebidCTVSeatName,
					}
					result[originalImpID] = impBids
				}

				//making unique bid.id's per impression
				bid.ID = util.GetUniqueBidID(bid.ID, len(impBids.Bids)+1)

				impBids.Bids = append(impBids.Bids, &types.Bid{
					Bid:               bid,
					FilterReasonCode:  constant.CTVRCDidNotGetChance,
					Duration:          getAdDuration(*bid, deps.impData[index].Config[sequenceNumber-1].MaxDuration),
					DealTierSatisfied: util.GetDealTierSatisfied(&ext),
				})
			}
		}
	}

	//Sort Bids by Price
	for index, imp := range deps.request.Imp {
		impBids, ok := result[imp.ID]
		if ok {
			//sort bids
			sort.Slice(impBids.Bids[:], func(i, j int) bool { return impBids.Bids[i].Price > impBids.Bids[j].Price })
			deps.impData[index].Bid = impBids
		}
	}
}

//getImpressionID will return impression id and sequence number
func (deps *ctvEndpointDeps) getImpressionID(id string) (string, int) {
	//get original impression id and sequence number
	originalImpID, sequenceNumber := util.DecodeImpressionID(id)

	//check originalImpID  present in request or not
	index, ok := deps.impIndices[originalImpID]
	if !ok {
		//if not present check impression id present in request or not
		index, ok = deps.impIndices[id]
		if !ok {
			return id, -1
		}
		return originalImpID, 0
	}

	if sequenceNumber < 0 || sequenceNumber > len(deps.impData[index].Config) {
		return id, -1
	}

	return originalImpID, sequenceNumber
}

//doAdPodExclusions
func (deps *ctvEndpointDeps) doAdPodExclusions() types.AdPodBids {
	defer util.TimeTrack(time.Now(), fmt.Sprintf("Tid:%v doAdPodExclusions", deps.request.ID))

	result := types.AdPodBids{}
	for index := 0; index < len(deps.request.Imp); index++ {
		bid := deps.impData[index].Bid
		if nil != bid && len(bid.Bids) > 0 {
			//TODO: MULTI ADPOD IMPRESSIONS
			//duration wise buckets sorted
			buckets := util.GetDurationWiseBidsBucket(bid.Bids[:])

			//combination generator
			comb := combination.NewCombination(
				buckets,
				uint64(deps.request.Imp[index].Video.MinDuration),
				uint64(deps.request.Imp[index].Video.MaxDuration),
				deps.impData[index].VideoExt.AdPod)

			//adpod generator
			adpodGenerator := response.NewAdPodGenerator(deps.request, index, buckets, comb, deps.impData[index].VideoExt.AdPod, deps.metricsEngine)

			adpodBids := adpodGenerator.GetAdPodBids()
			if adpodBids != nil {
				adpodBids.OriginalImpID = bid.OriginalImpID
				adpodBids.SeatName = bid.SeatName
				result = append(result, adpodBids)
			}
		}
	}
	return result
}

/********************* Creating CTV BidResponse *********************/

//createBidResponse
func (deps *ctvEndpointDeps) createBidResponse(resp *openrtb2.BidResponse, adpods types.AdPodBids) *openrtb2.BidResponse {
	defer util.TimeTrack(time.Now(), fmt.Sprintf("Tid:%v createBidResponse", deps.request.ID))

	bidResp := &openrtb2.BidResponse{
		ID:         resp.ID,
		Cur:        resp.Cur,
		CustomData: resp.CustomData,
		SeatBid:    deps.getBidResponseSeatBids(adpods),
	}

	//NOTE: this should be called at last
	bidResp.Ext = deps.getBidResponseExt(resp)
	return bidResp
}

func (deps *ctvEndpointDeps) getBidResponseSeatBids(adpods types.AdPodBids) []openrtb2.SeatBid {
	seats := []openrtb2.SeatBid{}

	//append pure video request seats
	for _, seat := range deps.videoSeats {
		seats = append(seats, *seat)
	}

	var adpodSeat *openrtb2.SeatBid
	for _, adpod := range adpods {
		if len(adpod.Bids) == 0 {
			continue
		}

		bid := deps.getAdPodBid(adpod)
		if bid != nil {
			if nil == adpodSeat {
				adpodSeat = &openrtb2.SeatBid{
					Seat: adpod.SeatName,
				}
			}
			adpodSeat.Bid = append(adpodSeat.Bid, *bid.Bid)
		}
	}
	if nil != adpodSeat {
		seats = append(seats, *adpodSeat)
	}
	return seats[:]
}

//getBidResponseExt will return extension object
func (deps *ctvEndpointDeps) getBidResponseExt(resp *openrtb2.BidResponse) (data json.RawMessage) {
	var err error

	adpodExt := types.BidResponseAdPodExt{
		Response: *resp,
		Config:   make(map[string]*types.ImpData, len(deps.impData)),
	}

	for index, imp := range deps.impData {
		if nil != imp.VideoExt && nil != imp.VideoExt.AdPod {
			adpodExt.Config[deps.request.Imp[index].ID] = imp
		}

		if nil != imp.Bid && len(imp.Bid.Bids) > 0 {
			for _, bid := range imp.Bid.Bids {
				//update adm
				//bid.AdM = constant.VASTDefaultTag

				//add duration value
				raw, err := jsonparser.Set(bid.Ext, []byte(strconv.Itoa(int(bid.Duration))), "prebid", "video", "duration")
				if nil == err {
					bid.Ext = raw
				}

				//add bid filter reason value
				raw, err = jsonparser.Set(bid.Ext, []byte(strconv.Itoa(bid.FilterReasonCode)), "adpod", "aprc")
				if nil == err {
					bid.Ext = raw
				}
			}
		}
	}

	//Remove extension parameter
	adpodExt.Response.Ext = nil

	if nil == resp.Ext {
		bidResponseExt := &types.ExtCTVBidResponse{
			AdPod: &adpodExt,
		}

		data, err = json.Marshal(bidResponseExt)
		if err != nil {
			glog.Errorf("JSON Marshal Error: %v", err.Error())
			return nil
		}
	} else {
		data, err = json.Marshal(adpodExt)
		if err != nil {
			glog.Errorf("JSON Marshal Error: %v", err.Error())
			return nil
		}

		data, err = jsonparser.Set(resp.Ext, data, constant.CTVAdpod)
		if err != nil {
			glog.Errorf("JSONParser Set Error: %v", err.Error())
			return nil
		}
	}

	return data[:]
}

//getAdPodBid
func (deps *ctvEndpointDeps) getAdPodBid(adpod *types.AdPodBid) *types.Bid {
	bid := types.Bid{
		Bid: &openrtb2.Bid{},
	}

	//TODO: Write single for loop to get all details
	bidID, err := uuid.NewV4()
	if nil == err {
		bid.ID = bidID.String()
	} else {
		bid.ID = adpod.Bids[0].ID
	}

	bid.ImpID = adpod.OriginalImpID
	bid.Price = adpod.Price
	bid.ADomain = adpod.ADomain[:]
	bid.Cat = adpod.Cat[:]
	bid.AdM = *getAdPodBidCreative(deps.request.Imp[deps.impIndices[adpod.OriginalImpID]].Video, adpod)
	bid.Ext = getAdPodBidExtension(adpod)
	return &bid
}

//getAdPodBidCreative get commulative adpod bid details
func getAdPodBidCreative(video *openrtb2.Video, adpod *types.AdPodBid) *string {
	doc := etree.NewDocument()
	vast := doc.CreateElement(constant.VASTElement)
	sequenceNumber := 1
	var version float64 = 2.0

	for _, bid := range adpod.Bids {
		var newAd *etree.Element

		if strings.HasPrefix(bid.AdM, constant.HTTPPrefix) {
			newAd = etree.NewElement(constant.VASTAdElement)
			wrapper := newAd.CreateElement(constant.VASTWrapperElement)
			vastAdTagURI := wrapper.CreateElement(constant.VASTAdTagURIElement)
			vastAdTagURI.CreateCharData(bid.AdM)
		} else {
			adDoc := etree.NewDocument()
			if err := adDoc.ReadFromString(bid.AdM); err != nil {
				continue
			}

			// adjust bidid in video event trackers and update
			adjustBidIDInVideoEventTrackers(adDoc, bid.Bid)
			adm, err := adDoc.WriteToString()
			if nil != err {
				util.JLogf("ERROR, %v", err.Error())
			} else {
				bid.AdM = adm
			}

			vastTag := adDoc.SelectElement(constant.VASTElement)

			//Get Actual VAST Version
			bidVASTVersion, _ := strconv.ParseFloat(vastTag.SelectAttrValue(constant.VASTVersionAttribute, constant.VASTDefaultVersionStr), 64)
			version = math.Max(version, bidVASTVersion)

			ads := vastTag.SelectElements(constant.VASTAdElement)
			if len(ads) > 0 {
				newAd = ads[0].Copy()
			}
		}

		if nil != newAd {
			//creative.AdId attribute needs to be updated
			newAd.CreateAttr(constant.VASTSequenceAttribute, fmt.Sprint(sequenceNumber))
			vast.AddChild(newAd)
			sequenceNumber++
		}
	}

	if int(version) > len(constant.VASTVersionsStr) {
		version = constant.VASTMaxVersion
	}

	vast.CreateAttr(constant.VASTVersionAttribute, constant.VASTVersionsStr[int(version)])
	bidAdM, err := doc.WriteToString()
	if nil != err {
		fmt.Printf("ERROR, %v", err.Error())
		return nil
	}
	return &bidAdM
}

//getAdPodBidExtension get commulative adpod bid details
func getAdPodBidExtension(adpod *types.AdPodBid) json.RawMessage {
	bidExt := &openrtb_ext.ExtCTVBid{
		ExtBid: openrtb_ext.ExtBid{
			Prebid: &openrtb_ext.ExtBidPrebid{
				Type:  openrtb_ext.BidTypeVideo,
				Video: &openrtb_ext.ExtBidPrebidVideo{},
			},
		},
		AdPod: &openrtb_ext.BidAdPodExt{
			RefBids: make([]string, len(adpod.Bids)),
		},
	}

	for i, bid := range adpod.Bids {
		bidExt.AdPod.RefBids[i] = bid.ID
		bidExt.Prebid.Video.Duration += int(bid.Duration)
		bid.FilterReasonCode = constant.CTVRCWinningBid
	}
	rawExt, _ := json.Marshal(bidExt)
	return rawExt
}

//getAdDuration determines the duration of video ad from given bid.
//it will try to get the actual ad duration returned by the bidder using prebid.video.duration
//if prebid.video.duration = 0 or there is error occured in determing it then
//impress
func getAdDuration(bid openrtb2.Bid, defaultDuration int64) int {
	duration, err := jsonparser.GetInt(bid.Ext, "prebid", "video", "duration")
	if nil != err || duration <= 0 {
		duration = defaultDuration
	}
	return int(duration)
}

func addTargetingKey(bid *openrtb2.Bid, key openrtb_ext.TargetingKey, value string) error {
	if nil == bid {
		return errors.New("Invalid bid")
	}

	raw, err := jsonparser.Set(bid.Ext, []byte(strconv.Quote(value)), "prebid", "targeting", string(key))
	if nil == err {
		bid.Ext = raw
	}
	return err
}

func adjustBidIDInVideoEventTrackers(doc *etree.Document, bid *openrtb2.Bid) {
	// adjusment: update bid.id with ctv module generated bid.id
	creatives := events.FindCreatives(doc)
	for _, creative := range creatives {
		trackingEvents := creative.FindElements("TrackingEvents/Tracking")
		if nil != trackingEvents {
			// update bidid= value with ctv generated bid id for this bid
			for _, trackingEvent := range trackingEvents {
				u, e := url.Parse(trackingEvent.Text())
				if nil == e {
					values, e := url.ParseQuery(u.RawQuery)
					// only do replacment if operId=8
					if nil == e && nil != values["bidid"] && nil != values["operId"] && values["operId"][0] == "8" {
						values.Set("bidid", bid.ID)
					} else {
						continue
					}

					//OTT-183: Fix
					if nil != values["operId"] && values["operId"][0] == "8" {
						operID := values.Get("operId")
						values.Del("operId")
						values.Add("_operId", operID) // _ (underscore) will keep it as first key
					}

					u.RawQuery = values.Encode() // encode sorts query params by key. _ must be first (assuing no other query param with _)
					// replace _operId with operId
					u.RawQuery = strings.ReplaceAll(u.RawQuery, "_operId", "operId")
					trackingEvent.SetText(u.String())
				}
			}
		}
	}
}
