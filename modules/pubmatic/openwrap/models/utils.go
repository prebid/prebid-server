package models

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	ImpressionIDSeparator = `::`
)

var SyncerMap map[string]usersync.Syncer

func SetSyncerMap(s map[string]usersync.Syncer) {
	SyncerMap = s
}

// IsCTVAPIRequest will return true if reqAPI is from CTV EndPoint
func IsCTVAPIRequest(api string) bool {
	return api == "/video/json" || api == "/video/vast" || api == "/video/openrtb"
}

// EdsStatusFromRequest returns ext.wrapper.edsstatus when present on the bid request.
func EdsStatusFromRequest(req *openrtb2.BidRequest) *int {
	if req == nil || len(req.Ext) == 0 {
		return nil
	}

	edsStatus, err := jsonparser.GetInt(req.Ext, "wrapper", "edsstatus")
	if err != nil {
		return nil
	}

	return ptrutil.ToPtr(int(edsStatus))
}

// ResolveEdsStatus returns edsstatus for logging. SDK bidding integrations use signal data only.
func ResolveEdsStatus(fromSignalOnly bool, wrapper RequestExtWrapper, signal *openrtb2.BidRequest) *int {
	if fromSignalOnly {
		return EdsStatusFromRequest(signal)
	}

	return wrapper.EdsStatus
}

func GetRequestExtWrapper(request []byte, wrapperLocation ...string) (RequestExtWrapper, error) {
	extWrapper := RequestExtWrapper{
		SSAuctionFlag: -1,
	}

	if len(wrapperLocation) == 0 {
		wrapperLocation = []string{"ext", "prebid", "bidderparams", "pubmatic", "wrapper"}
	}

	extWrapperBytes, _, _, err := jsonparser.Get(request, wrapperLocation...)
	if err != nil {
		return extWrapper, fmt.Errorf("request.ext.wrapper not found: %v", err)
	}

	err = json.Unmarshal(extWrapperBytes, &extWrapper)
	if err != nil {
		return extWrapper, fmt.Errorf("failed to decode request.ext.wrapper : %v", err)
	}

	return extWrapper, nil
}

func GetTest(request []byte) (int64, error) {
	test, err := jsonparser.GetInt(request, "test")
	if err != nil {
		return test, fmt.Errorf("request.test not found: %v", err)
	}
	return test, nil
}

func GetSize(width, height int64) string {
	return fmt.Sprintf("%dx%d", width, height)
}

// CreatePartnerKey returns key with partner appended
func CreatePartnerKey(partner, key string) string {
	if partner == "" {
		return key
	}
	return key + "_" + partner
}

// GetCreativeType gets adformat from creative(adm) of the bid
func GetCreativeType(bid *openrtb2.Bid, bidExt *BidExt, impCtx *ImpCtx) string {
	if bidExt.Prebid != nil && len(bidExt.Prebid.Type) > 0 {
		return string(bidExt.Prebid.Type)
	}
	if bid.AdM == "" {
		return ""
	}
	if openrtb_ext.IsVideo(bid.AdM) {
		return Video
	}

	if impCtx.Native != nil {
		if openrtb_ext.IsNative(bid.AdM) {
			return Native
		}
	}
	return Banner
}

func IsDefaultBid(bid *openrtb2.Bid) bool {
	return bid.Price == 0 && bid.DealID == "" && bid.W == 0 && bid.H == 0
}

// GetAdFormat returns adformat of the bid.
// for default bid it refers to impression object
// for non-default bids it uses creative(adm) of the bid
func GetAdFormat(bid *openrtb2.Bid, bidExt *BidExt, impCtx *ImpCtx) string {
	if bid == nil || impCtx == nil {
		return ""
	}
	if IsDefaultBid(bid) {
		if impCtx.IsBanner {
			return Banner
		}
		if impCtx.Video != nil {
			return Video
		}
		if impCtx.Native != nil {
			return Native
		}
		return ""
	}
	if bidExt == nil {
		return ""
	}
	return GetCreativeType(bid, bidExt, impCtx)
}

func GetRevenueShare(partnerConfig map[string]string) float64 {
	var revShare float64

	if val, ok := partnerConfig[REVSHARE]; ok {
		revShare, _ = strconv.ParseFloat(val, 64)
	}
	return revShare
}

func GetNetEcpm(price float64, revShare float64) float64 {
	if revShare == 0 {
		return ToFixed(price, BID_PRECISION)
	}
	price = price * (1 - revShare/100)
	return ToFixed(price, BID_PRECISION)
}

func GetGrossEcpm(price float64) float64 {
	return ToFixed(price, BID_PRECISION)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func ExtractDomain(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "http://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	return u.Host, nil
}

// hybrid/web request would have bidder params prepopulated.
// TODO: refer request.ext.prebid.channel.name = pbjs instead?
func IsHybrid(body []byte) bool {
	_, _, _, err := jsonparser.Get(body, "imp", "[0]", "ext", "prebid", "bidder", "pubmatic")
	return err == nil
}

// GetVersionLevelPropertyFromPartnerConfig returns a Version level property from the partner config map
func GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap map[int]map[string]string, propertyName string) string {
	if versionLevelConfig, ok := partnerConfigMap[VersionLevelConfigID]; ok && versionLevelConfig != nil {
		return versionLevelConfig[propertyName]
	}
	return ""
}

const (
	//The following are the headerds related to IP address
	XForwarded      = "X-FORWARDED-FOR"
	SourceIP        = "SOURCE_IP"
	ClusterClientIP = "X_CLUSTER_CLIENT_IP"
	RemoteAddr      = "REMOTE_ADDR"
	RlnClientIP     = "RLNCLIENTIPADDR"
	XDeviceIP       = "X-Device-IP"
)

// List of IP header keys in priority order
var ipHeaders = []string{
	RlnClientIP,     // For API
	XDeviceIP,       // HTTP_X-Device-IP
	SourceIP,        // HTTP_SOURCE_IP
	ClusterClientIP, // HTTP_X_CLUSTER_CLIENT_IP
	XForwarded,      // HTTP_X_FORWARDED_IP
	RemoteAddr,      // REMOTE_ADDR
}

// GetIP extracts IP considering precedence
func GetIP(in *http.Request) string {
	// Iterate over the headers to find the first non-empty IP
	for _, header := range ipHeaders {
		if ip := in.Header.Get(header); ip != "" {
			// If multiple IPs are found, split and return the first one
			ips := strings.Split(ip, ",")
			return ips[0]
		}
	}

	// Fallback to parsing the RemoteAddr if no headers have an IP
	ip, _, err := net.SplitHostPort(in.RemoteAddr)
	if err != nil {
		glog.Errorf("[GetIP] status:[invalid_ip] ip:[%s] error:[%s]", in.RemoteAddr, err)
		return ""
	}
	return ip
}

func Atof(value string, decimalplaces int) (float64, error) {

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	value = fmt.Sprintf("%."+strconv.Itoa(decimalplaces)+"f", floatValue)
	floatValue, err = strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	return floatValue, nil
}

// IsPubmaticCorePartner returns true when the partner is pubmatic or internally an alias of pubmatic
func IsPubmaticCorePartner(partnerName string) bool {
	if partnerName == string(openrtb_ext.BidderPubmatic) || partnerName == BidderPubMaticSecondaryAlias {
		return true
	}
	return false
}

// wraps error with error msg
func ErrorWrap(cErr, nErr error) error {
	if cErr == nil {
		return nErr
	}

	return errors.Wrap(cErr, nErr.Error())
}

// getImpressionID will return impression id and sequence number
func GetImpressionID(id string) (string, int) {
	//get original impression id and sequence number
	ImpID, sequenceNumber := DecodeImpressionID(id)
	if sequenceNumber < 0 {
		return id, -1
	}

	return ImpID, sequenceNumber
}

func DecodeImpressionID(id string) (string, int) {
	index := strings.LastIndex(id, ImpressionIDSeparator)
	if index == -1 {
		return id, 0
	}

	sequence, err := strconv.Atoi(id[index+2:])
	if err != nil || sequence == 0 {
		return id, 0
	}

	return id[:index], sequence
}

func GetSizeForPlatform(width, height int64, platform string) string {
	s := fmt.Sprintf("%dx%d", width, height)
	return s
}

func GetKGPSV(bid openrtb2.Bid, bidExt *BidExt, bidderMeta PartnerData, adformat string, tagId string, div string, source string) (string, string) {
	kgpv := bidderMeta.KGPV
	kgpsv := bidderMeta.MatchedSlot
	kgp := bidderMeta.KGP
	isRegex := bidderMeta.IsRegex
	// 1. nobid
	if IsDefaultBid(&bid) {
		//NOTE: kgpsv = bidderMeta.MatchedSlot above. Use the same
		if kgp == ADUNIT_SOURCE_VASTTAG_KGP {
			kgpv, kgpsv = getVASTBidderKGPVFromBidResponse(kgpv, bidExt)
		} else if !isRegex && kgpv != "" { // unmapped pubmatic's slot
			kgpsv = kgpv
		} else if !isRegex {
			kgpv = kgpsv
		}
	} else if kgp == ADUNIT_SOURCE_VASTTAG_KGP {
		kgpv, kgpsv = getVASTBidderKGPVFromBidResponse(kgpv, bidExt)
	} else if !isRegex {
		if kgpv != "" { // unmapped pubmatic's slot
			kgpsv = kgpv
		} else if adformat == Video { // Check when adformat is video, bid.W and bid.H has to be zero with Price !=0. Ex: UOE-9222(0x0 default kgpv and kgpsv for video bid)
			// 2. valid video bid
			// kgpv has regex, do not generate slotName again
			// kgpsv could be unmapped or mapped slot, generate slotName with bid.W = bid.H = 0
			kgpsv = GenerateSlotName(0, 0, bidderMeta.KGP, tagId, div, source)
			kgpv = kgpsv // original /43743431/DMDemo1234@300x250 but new could be /43743431/DMDemo1234@0x0
		} else if bid.H != 0 && bid.W != 0 { // Check when bid.H and bid.W will be zero with Price !=0. Ex: MobileInApp-MultiFormat-OnlyBannerMapping_Criteo_Partner_Validaton
			// 3. valid bid
			// kgpv has regex, do not generate slotName again
			// kgpsv could be unmapped or mapped slot, generate slotName again based on bid.H and bid.W
			kgpsv = GenerateSlotName(bid.H, bid.W, bidderMeta.KGP, tagId, div, source)
			kgpv = kgpsv
		}
	}
	if kgpv == "" {
		kgpv = kgpsv
	}
	return kgpv, kgpsv
}

// Harcode would be the optimal. We could make it configurable like _AU_@_W_x_H_:%s@%dx%d entries in pbs.yaml
// mysql> SELECT DISTINCT key_gen_pattern FROM wrapper_mapping_template;
// +----------------------+
// | key_gen_pattern      |
// +----------------------+
// | _AU_@_W_x_H_         |
// | _DIV_@_W_x_H_        |
// | _W_x_H_@_W_x_H_      |
// | _DIV_                |
// | _AU_@_DIV_@_W_x_H_   |
// | _AU_@_SRC_@_VASTTAG_ |
// +----------------------+
// 6 rows in set (0.21 sec)
func GenerateSlotName(h, w int64, kgp, tagid, div, src string) string {
	// func (H, W, Div), no need to validate, will always be non-nil
	switch kgp {
	case "_AU_", "_RE_": // adunitconfig or defaultmappingKGP
		return tagid
	case "_DIV_":
		return div
	case "_AU_@_W_x_H_", "_RE_@_W_x_H_":
		return fmt.Sprintf("%s@%dx%d", tagid, w, h)
	case "_DIV_@_W_x_H_":
		return fmt.Sprintf("%s@%dx%d", div, w, h)
	case "_W_x_H_@_W_x_H_":
		return fmt.Sprintf("%dx%d@%dx%d", w, h, w, h)
	case "_AU_@_DIV_@_W_x_H_":
		return fmt.Sprintf("%s@%s@%dx%d", tagid, div, w, h)
	case "_AU_@_SRC_@_VASTTAG_":
		return fmt.Sprintf("%s@%s@", tagid, src)
	default:
		// TODO: check if we need to fallback to old generic flow (below)
		// Add this cases in a map and read it from yaml file
	}
	return ""
}

func RoundToTwoDigit(value float64) float64 {
	output := math.Pow(10, float64(2))
	return float64(math.Round(value*output)) / output
}

// GetBidLevelFloorsDetails return floorvalue and floorrulevalue
func GetBidLevelFloorsDetails(bidExt BidExt, impCtx ImpCtx,
	currencyConversion func(from, to string, value float64) (float64, error)) (fv, frv float64) {
	var floorCurrency string
	frv = NotSet

	//Set the fv from bid.ext.mbmf if it is set
	if bidExt.MultiBidMultiFloorValue > 0 {
		fv = RoundToTwoDigit(bidExt.MultiBidMultiFloorValue)
		frv = RoundToTwoDigit(bidExt.MultiBidMultiFloorValue)
		return
	}

	if bidExt.Prebid != nil && bidExt.Prebid.Floors != nil {
		floorCurrency = bidExt.Prebid.Floors.FloorCurrency
		fv = RoundToTwoDigit(bidExt.Prebid.Floors.FloorValue)
		frv = fv
		if bidExt.Prebid.Floors.FloorRuleValue > 0 {
			frv = RoundToTwoDigit(bidExt.Prebid.Floors.FloorRuleValue)
		}
	}

	// if floor values are not set from bid.ext then fall back to imp.bidfloor
	if frv == NotSet && impCtx.BidFloor != 0 {
		fv = RoundToTwoDigit(impCtx.BidFloor)
		frv = fv
		floorCurrency = impCtx.BidFloorCur
	}

	// convert the floor values in USD currency
	if floorCurrency != "" && floorCurrency != USD {
		value, _ := currencyConversion(floorCurrency, USD, fv)
		fv = RoundToTwoDigit(value)
		value, _ = currencyConversion(floorCurrency, USD, frv)
		frv = RoundToTwoDigit(value)
	}

	if frv == NotSet {
		frv = 0 // set it back to 0
	}

	return
}

// GetFloorsDetails returns floors details from response.ext.prebid
func GetFloorsDetails(responseExt openrtb_ext.ExtBidResponse) (floorDetails FloorsDetails) {
	if responseExt.Prebid != nil && responseExt.Prebid.Floors != nil {
		floors := responseExt.Prebid.Floors
		if floors.Skipped != nil {
			floorDetails.Skipfloors = ptrutil.ToPtr(0)
			if *floors.Skipped {
				floorDetails.Skipfloors = ptrutil.ToPtr(1)
			}
		}
		if floors.Data != nil && len(floors.Data.ModelGroups) > 0 {
			floorDetails.FloorModelVersion = floors.Data.ModelGroups[0].ModelVersion
		}
		if len(floors.PriceFloorLocation) > 0 {
			if source, ok := FloorSourceMap[floors.PriceFloorLocation]; ok {
				floorDetails.FloorSource = &source
			}
		}
		if status, ok := FetchStatusMap[floors.FetchStatus]; ok {
			floorDetails.FloorFetchStatus = &status
		}
		floorDetails.FloorProvider = floors.FloorProvider
		if floors.Data != nil && len(floors.Data.FloorProvider) > 0 {
			floorDetails.FloorProvider = floors.Data.FloorProvider
		}
		if floors.Enforcement != nil && floors.Enforcement.EnforcePBS != nil && *floors.Enforcement.EnforcePBS {
			floorDetails.FloorType = HardFloor
		}
	}
	return floorDetails
}

func getVASTBidderKGPVFromBidResponse(slotKey string, bidExt *BidExt) (string, string) {
	if bidExt.Prebid != nil && bidExt.Prebid.Video != nil && len(bidExt.Prebid.Video.VASTTagID) > 0 {
		tagID := bidExt.Prebid.Video.VASTTagID
		if index := strings.LastIndex(tagID, "@"); index > 0 {
			return tagID, tagID[:index+1]
		}
	}
	return slotKey, slotKey
}

func GetGrossEcpmFromNetEcpm(netEcpm float64, revShare float64) float64 {
	if revShare == 100 {
		return 0
	}
	originalBidPrice := netEcpm / (1 - revShare/100)
	return ToFixed(originalBidPrice, BID_PRECISION)
}

func GetBidAdjustmentValue(revShare float64) float64 {
	return (1 - revShare/100)
}

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func GetMultiFloors(impMultiFloorsMap map[string]*MultiFloors, impID string) []float64 {
	if impMultiFloorsMap == nil {
		return nil
	}

	mf, ok := impMultiFloorsMap[impID]
	if !ok || mf == nil {
		return nil
	}

	multifloors := []float64{mf.Tier1, mf.Tier2, mf.Tier3}
	if mf.Tier4 > 0 {
		multifloors = append(multifloors, mf.Tier4)
	}
	if mf.Tier5 > 0 {
		multifloors = append(multifloors, mf.Tier5)
	}
	return multifloors
}
