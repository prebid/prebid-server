package exchange

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestGetCurrency(t *testing.T) {
	bidRequest := new(openrtb.BidRequest)
	bidRequest.ID = "this-request-id"
	bidRequest.Imp = make([]openrtb.Imp, 1)
	impExt := make(map[string]map[string]string)
	impExt["rubicon"] = make(map[string]string)
	impExt["appnexus"] = make(map[string]string)
	thisImpExt, _ := json.Marshal(impExt)
	bidRequest.Imp[0].Ext = thisImpExt

	seatBids := make([]openrtb.SeatBid, 2)
	seatBids[0].Seat = "rubicon"
	seatBids[0].Bid = make([]openrtb.Bid, 1)
	seatBids[0].Bid[0].Price = 4.43
	seatBids[1].Seat = "appnexus"
	seatBids[1].Bid = make([]openrtb.Bid, 1)
	seatBids[1].Bid[0].Price = 3.11

	adapterExtra := map[openrtb_ext.BidderName]*seatResponseExtra{}
	adapterExtra["rubicon"] = &seatResponseExtra{ResponseTimeMillis: 50, Errors: []string{}}
	adapterExtra["appnexus"] = &seatResponseExtra{ResponseTimeMillis: 50, Errors: []string{"error 1"}}

	currency := getCurrency(validLatestCurrencyRates, &seatBids, bidRequest, &adapterExtra)
	if currency != "USD" {
		t.Fatalf("Currency should be default USD since request currency was not set.")
	}
	if len(seatBids) != 2 {
		t.Fatalf("There should still be 2 seat bids since none should have been dropped.")
	}

	bidRequest.Cur = []string{"USD"}
	currency = getCurrency(validLatestCurrencyRates, &seatBids, bidRequest, &adapterExtra)
	if currency != "USD" {
		t.Fatalf("Currency should be default USD since it was set to that in request.")
	}
	if len(seatBids) != 2 {
		t.Fatalf("There should still be 2 seat bids since none should have been dropped.")
	}

	bidRequest.Cur = []string{"AUD"}
	bidExt := make(map[string]string)
	bidExt["ad_server_currency"] = "USD"
	rpBidExt, _ := json.Marshal(bidExt)
	seatBids[0].Bid[0].Ext = rpBidExt
	currency = getCurrency(validLatestCurrencyRates, &seatBids, bidRequest, &adapterExtra)
	if currency != "AUD" {
		t.Fatalf("Currency should be AUD since it was set to that in request.")
	}
	if len(seatBids) != 1 || seatBids[0].Seat != "rubicon" {
		t.Fatalf("There should be 1 seat bid left (rubicon) because appnexus bid should have been dropped since they did not specify a bid currency.")
	}

	bidExt["ad_server_currency"] = "BAM"
	rpBidExt, _ = json.Marshal(bidExt)
	seatBids[0].Bid[0].Ext = rpBidExt
	_ = getCurrency(validLatestCurrencyRates, &seatBids, bidRequest, &adapterExtra)
	if len(seatBids) != 0 {
		t.Fatalf("There shouldn't be any seat bids left since rubicon bid is now returning an invalid currency in bid response.")
	}
}

func TestConvertCurrencyWithCurrentRates(t *testing.T) {
	requestCurrency := "AUD"
	bidCurrency := "GBP"
	bid := openrtb.Bid{
		Price: 2.31,
	}
	nilCurrencyRates := []byte{}
	bidderErrors := []string{}

	convertedBid, successfulConversion := convertCurrencyWithCurrentRates(nilCurrencyRates, &bidderErrors, requestCurrency, bidCurrency, bid)
	if successfulConversion == true || convertedBid != 0.0 {
		t.Fatalf("Currency conversion should have failed since currency rates were not proviced.")
	}
	if len(bidderErrors) != 1 {
		t.Fatalf("Should have 1 bidder error since currency conversion failed once.")
	}

	convertedBid, successfulConversion = convertCurrencyWithCurrentRates(invalidLatestCurrencyRates, &bidderErrors, requestCurrency, bidCurrency, bid)
	if successfulConversion == true || convertedBid != 0.0 {
		t.Fatalf("Currency conversion should have failed since currency rates were not in valid format proviced.")
	}
	if len(bidderErrors) != 2 {
		t.Fatalf("Should have 2 bidder errors since currency conversion failed twice.")
	}

	convertedBid, successfulConversion = convertCurrencyWithCurrentRates(otherLatestCurrencyRates, &bidderErrors, requestCurrency, bidCurrency, bid)
	if successfulConversion == true || convertedBid != 0.0 {
		t.Fatalf("Currency conversion should have failed since currencies we needed were not in the rates data.")
	}
	if len(bidderErrors) != 3 {
		t.Fatalf("Should have gotten 3 bidder errors since currency conversion failed thrice.")
	}

	convertedBid, successfulConversion = convertCurrencyWithCurrentRates(validLatestCurrencyRates, &bidderErrors, requestCurrency, bidCurrency, bid)
	if successfulConversion != true || convertedBid != 3.99 {
		t.Fatalf("Currency conversion should have been successful, expected converted bid is %f, got %f", 3.99, convertedBid)
	}
	if len(bidderErrors) != 3 {
		t.Fatalf("Should still have 3 bidder errors since last currency conversion was successful.")
	}
}

func TestConvertCurrencyWithRates(t *testing.T) {
	requestCurrency := "AUD"
	bidCurrency := "GBP"
	bidPrice := 2.31

	rates := openrtb_ext.ConversionRates{}
	convertedRate, err1 := convertCurrencyWithRates(rates, requestCurrency, bidCurrency, bidPrice)
	if convertedRate != 0 || err1 == nil {
		t.Fatalf("Currency conversion should have failed since currency rates were nil.")
	}

	rates = openrtb_ext.ConversionRates{}
	if err2 := json.Unmarshal(validDirectConversions, &rates); err2 != nil {
		t.Fatalf("Error unmarshalling conversion rates: %s", err2)
	}
	convertedRate, err3 := convertCurrencyWithRates(rates, requestCurrency, bidCurrency, bidPrice)
	if convertedRate != 3.99 || err3 != nil {
		t.Fatalf("Currency conversion should have been successful since direct conversion was available.")
	}

	rates = openrtb_ext.ConversionRates{}
	if err4 := json.Unmarshal(validIndirectConversions, &rates); err4 != nil {
		t.Fatalf("Error unmarshalling conversion rates: %s", err4)
	}
	convertedRate, err5 := convertCurrencyWithRates(rates, requestCurrency, bidCurrency, bidPrice)
	if convertedRate != 1.16 || err5 != nil {
		t.Fatalf("Currency conversion should have been successful since indirect conversion was available.")
	}

	rates = openrtb_ext.ConversionRates{}
	if err6 := json.Unmarshal(otherConversions, &rates); err6 != nil {
		t.Fatalf("Error unmarshalling conversion rates: %s", err6)
	}
	convertedRate, err7 := convertCurrencyWithRates(rates, requestCurrency, bidCurrency, bidPrice)
	if convertedRate != 0 || err7 == nil {
		t.Fatalf("Currency conversion should have failed since currency rates were not available for requested currencies.")
	}

	rates = openrtb_ext.ConversionRates{}
	if err8 := json.Unmarshal(invalidIndirectConversions, &rates); err8 != nil {
		t.Fatalf("Error unmarshalling conversion rates: %s", err8)
	}
	convertedRate, err9 := convertCurrencyWithRates(rates, requestCurrency, bidCurrency, bidPrice)
	if convertedRate != 0 || err9 == nil {
		t.Fatalf("Currency conversion should have failed since both currency rates were not available in indirect conversion.")
	}
}

var validLatestCurrencyRates = []byte(
	`{
		"dataAsOf":"2018-01-07",
		"conversions": {
			"GBP": {
				"AUD": 1.7282
			},
			"USD": {
				"AUD": 2.3455
			}
		}
	}`,
)

var otherLatestCurrencyRates = []byte(
	`{
		"dataAsOf":"2018-01-07",
		"conversions": {
			"USD": {
				"SAN": 3.2113
			}
		}
	}`,
)

var invalidLatestCurrencyRates = []byte(
	`{
		"not-conversions": {
			"GBP": {
				"AUD": 1.7282,
			}
		}
	}`,
)

var validDirectConversions = []byte(
	`{
		"GBP": {
			"AUD": 1.7282
		},
		"BAM": {
			"SAN": 3.4455
		}
	}`,
)

var validIndirectConversions = []byte(
	`{
		"USD": {
			"AUD": 1.7282,
			"GBP": 3.4556
		}
	}`,
)

var invalidIndirectConversions = []byte(
	`{
		"USD": {
			"AUD": 1.7282
		}
	}`,
)

var otherConversions = []byte(
	`{
		"USD": {
			"SAN": 3.2113
		},
		"SAN": {
			"BAM": 1.2234
		}
	}`,
)
