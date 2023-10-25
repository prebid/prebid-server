package criteoretail

import (
	"encoding/json"
	"math/rand"
	"strconv"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var mockProductDetails = map[string]interface{}{
    "ProductName": "Comet Professional Multi Purpose Disinfecting - Sanitizing Liquid Bathroom Cleaner Spray, 32 fl oz. (19214)",
    "Image": "https://www.staples-3p.com/s7/is/image/Staples/7246D7B1-E4B9-4AA7-9261723D16E5023B_sc7?$std$",
    "ProductPage": "//b.va.us.criteo.com/rm?dest=https%3a%2f%2fwww.staples.com%2fcomet-sanitizing-bathroom-cleaner-32-oz%2fproduct_24442430%3fcid%3dBNR%3a24442430%26ci_src%3d17588969%26ci_sku%3d24442430%26KPID%3d24442430&sig=1-4QbIhl7enF8z5Znj9uK8DV4gv_b2ZKe2Lh5HS3-bdpk&rm_e=YwVUlwDHD1yRccXhY_Ge0ja0KIBxHmS8Afyav-5cSdx0QvY4RquAAXx_8LEJg4urqqOpSIf06OGnwvoRI2oQbLyeSkU-r4ssFHxwbZJBEQDNB8ba9rr-NE9n9Ba2-hUZoUEGeOueYCRUznfSVOsfaYFyIxiM9DG2D_YpIOyLi0OXr6M-x7Os8_l01yD-Ckf9SG-8b_dPTfo_Jvlp2wHRWRbz_JexsXMrF1nDg7Gg6cYrZs3fMPF2wbGgNN9ijxC2zGQ2SiWy0kiEOofJAjp8F6Bxa445dQnGhLJNdlmZMeMrzIVSLb4elmaQvlcCVHF14PG_kabsptpJsPc7W8x7ONQgkwKL1gfXgU4IDjCRMZoVckV0RZY8v3t8a_QlkF9BHss4t5TtH4u4_tRgqdwKVBhl9OV5fSEsMJ1P-ir3ddIceuzyZX8WXSbxOebRf4i15xls9t6s9-zIxSFiVr_AT_HwU5SnIeVPlCES7CBUSxy-_NqnSRTTGdqCvVi_ElReUyghh7w7_TwwOUm4qMeAd-cEhhBQMPQHRXDElAVOIHyhWp7R1u-GzgSWfF93jxdR0Z7kN2bvhvPzi5Cqf3QSxA&ev=4",
    "Rating": "5.0",
    "RenderingAttributes": "{\"additional_producturl\":\"https://www.staples.com/comet-sanitizing-bathroom-cleaner-32-oz/product_24442430?cid=BNR:24442430&ci_src=17588969&ci_sku=24442430&KPID=24442430\",\"brand\":\"comet\",\"delivery_message\":\"1 Business Day\",\"google_product_type\":\"Cleaning Supplies > Cleaning Chemicals & Wipes > Cleaning Chemicals\",\"issellersku\":\"0\",\"mapViolation\":\"0\",\"numberOfReviews\":\"0\",\"pid\":\"24442430\",\"pr_count\":\"6\",\"price\":\"12.59\",\"price_in_cart_flag\":\"0\",\"producturl\":\"http://www.staples.com/product_24442430\",\"sale\":\"12.59\",\"sellingpacksize\":\"Each\",\"shippingCost\":\"0\",\"shipping_cost\":\"Free if order is over $49.99\",\"staples_brand\":\"Comet\",\"taxonomy_text\":\"cleaning supplies>cleaning chemicals & wipes>cleaning chemicals & wipes>cleaning chemicals\"}",
    "adid": "1",
    "shortDescription": "Disinfecting/sanitizing bathroom cleaner for infection prevention and control",
    "MatchType": "sku",
    "ParentSKU": "24442430",
}

const (
	MAX_COUNT   = 9
	IMP_URL  = "//b.va.us.criteo.com/rm?rm_e=V371SUC6Q0sB1j7Qlln9yr4HmnAeOoZf79-n9l7MezTyqJTRG9e0i5nSRGOMvkJVIy916h08clCmFSUIeYuJI1pNc1Skg2jiJnnLr9_IDUiIGwaE1-r6TQZtzwz2CuFXuc5ZNn86gzWlM3ciRwey0bumgKtGwi3zX1NWgsg0HmKEVLkwjHslFwdVSg6phjt29NuQ-fxPnp4SOschyHefI0HNM0AvREMQkLcYMGKkADJ07E0FomZhagxPFhBE4Qrk0m3AB46CHNsu3E7IVJ2RwAatybYFDQP2rgTr--mFQ0jlZUggctRZvMAQkd1e17YPW-x8mQd_NqPVSYHvo6peUfSUhgeRyvBf2NtZWMG4NBaaXCHZZuNFDT2_DYCcQfeFanp1wOYNii3_-WLBdiEhWgvIVH5psM-xgV8hEnZSlz-__UtOMVTaXdZiRDEouaHnEaaZTV7Vi8clDSz04v2nbTSL2ta1sl7-EzDEtQAfftjVd42k3OzofBbad57JXSZ9hbJaomW5r8yibxkbdjEv1u-Nldm-HkIIc0xsWMLCUog&ev=4"
	CLICK_URL    = "//b.va.us.criteo.com/rm?rm_e=4Cwm-UKrl2ok74hbwFbixP5V17YxlhWthucJj_ny-DoqhUxhNLckmIt2T8Xpc3U0pq3aIZBPY4tp8MQQ53PKfRascJnYUUkz3RI-z5JZB5nlNOq9_lPZxCjtd0YDHZvWw2Bu05u1SvnOiHNpBQpqBw68R4Fsbsed78vQ4DnOtnntXzS9xCqVHOKS6sIQIInjAiFvcEIa4MU7PrOTa1KoHGM-qmUQlaCZxKzADq0MTA3Nr5xaFZFBiZhyCjWxss27YVCbNjXHS8N4xSiCaCkrN9tZcRVtKzc-CdmKcUss-ZfHog_Q_1X76qc59nklcPznF6ax-EDx8xTQQ_cZnja_c_QnSrH732fuHG9vbuEhHl-tbRo3XCEI39PRupAcMB75k_a--VnxgOjgdJryWp6Sl2Qme23qHv2YdZkBBcxt4Z78KnjFeBGwPPA98wNSig5g15QN_Dae3K-XEmSrAFDjWnZmXb4PuVVeP72VOSaG9_YLwx_L4vsXzeDdnjM6w3WRP6yIqc_ZVBnloH8I1T31KayJfxKkh2Fs4a5-1NDgcgZoxmqfbtpd5LIbgPBexD8Sw7H9efxx8ql_EpgmnxMBTA&ev=4"
)
func (a *CriteoRetailAdapter) GetMockResponse(internalRequest *openrtb2.BidRequest) *adapters.BidderResponse {
	requestCount := GetRequestSlotCount(internalRequest)
	impiD := internalRequest.Imp[0].ID

	responseF := GetMockBids(requestCount, impiD)
	return responseF
}

func GetRequestSlotCount(internalRequest *openrtb2.BidRequest) int {
	impArray := internalRequest.Imp
	reqCount := 0
	for _, eachImp := range impArray {
		var commerceExt openrtb_ext.ExtImpCommerce
		json.Unmarshal(eachImp.Ext, &commerceExt)
		reqCount += commerceExt.ComParams.SlotsRequested
	}
	return reqCount
}

func GetRandomProductID() string {
	min := 100000
	max := 600000
	randomN := rand.Intn(max-min+1) + min
	t := strconv.Itoa(randomN)
	return t
}

func GetRandomBidPrice() float64 {
	min := 1.0
	max := 15.0
	untruncated := min + rand.Float64()*(max-min)
	truncated := float64(int(untruncated*100)) / 100
	return truncated
}

func GetRandomClickPrice(max float64) float64 {
	min := 1.0
	untruncated := min + rand.Float64()*(max-min)
	truncated := float64(int(untruncated*100)) / 100
	return truncated
}

func GetMockBids(requestCount int, ImpID string) *adapters.BidderResponse {
	var typedArray []*adapters.TypedBid

	if requestCount > MAX_COUNT {
		requestCount = MAX_COUNT
	}
	
	for i := 1; i <= requestCount; i++ {
		productid := GetRandomProductID()
		bidPrice := GetRandomBidPrice()
		clickPrice := GetRandomClickPrice(bidPrice)
		bidID := adapters.GenerateUniqueBidIDComm()
		impID := ImpID + "_" + strconv.Itoa(i)

		bidExt := &openrtb_ext.ExtBidCommerce{
			ProductId:  productid,
			ClickPrice: clickPrice,
			ClickUrl: CLICK_URL,
			ProductDetails: mockProductDetails,
		}

		bid := &openrtb2.Bid{
			ID:    bidID,
			ImpID: impID,
			Price: bidPrice,
			IURL: IMP_URL,
		}

		adapters.AddDefaultFieldsComm(bid)

		bidExtJSON, err1 := json.Marshal(bidExt)
		if nil == err1 {
			bid.Ext = json.RawMessage(bidExtJSON)
		}

		typedbid := &adapters.TypedBid{
			Bid:  bid,
			Seat: openrtb_ext.BidderName(SEAT_CRITEORETAIL),
		}
		typedArray = append(typedArray, typedbid)
	}

	responseF := &adapters.BidderResponse{
		Bids: typedArray,
	}
	return responseF
}

