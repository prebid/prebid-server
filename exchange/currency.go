package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"strconv"
)

const defaultCurrencyConversionError = "No currency conversion available."

type currencyData struct {
	DataAsOfDate    string                      `json:"dataAsOf"`
	ConversionRates openrtb_ext.ConversionRates `json:"conversions"`
}

func getCurrency(currencyRates []byte, seatBids *[]openrtb.SeatBid, bidRequest *openrtb.BidRequest, adapterExtra *map[openrtb_ext.BidderName]*seatResponseExtra) string {
	currencyToUse := defaultAdServerCurrency
	requestCurrency := defaultAdServerCurrency // Assuming default USD for request currency if one was not set

	if len(bidRequest.Cur) == 1 {
		// Setting request currency if one was set
		requestCurrency = bidRequest.Cur[0]
	}

	var requestExt openrtb_ext.ExtRequest
	if bidRequest.Ext != nil {
		json.Unmarshal(bidRequest.Ext, &requestExt)
	}

	newSeatBids := make([]openrtb.SeatBid, 0, len(*seatBids))

	// Iterate through bidder currencies and make sure they match with request currency
	for _, seatBid := range *seatBids {
		var bidderErrs []string
		bidderName, isValid := openrtb_ext.BidderMap[seatBid.Seat]
		if !isValid {
			// Invalid bidder name in SeatBid.Seat. Skip to the next seat.
			continue
		}
		adapterExtraCopy := *adapterExtra
		newBids := make([]openrtb.Bid, 0, len(seatBid.Bid))

		for _, bid := range seatBid.Bid {
			// Fetch currency from bid
			bidCurrency, err := jsonparser.GetString(bid.Ext, "ad_server_currency")
			if err != nil || bidCurrency == "" {
				if requestCurrency == defaultAdServerCurrency {
					// If bidder did not provide bid currency but request currency was set to USD (default), use USD
					currencyToUse = defaultAdServerCurrency
					newBids = append(newBids, bid)
					continue
				}

				// Currency was set in request but bidder did not provide currency info in bid. Return error and drop the bid.
				bidderErrs = append(bidderErrs, errors.New(defaultCurrencyConversionError).Error())
				continue
			}

			if bidCurrency != requestCurrency {
				var convertedBid float64
				var successfulConversion bool
				// Currencies do not match for this bid. Need to convert the bid price based on converstion rates.
				// Check if conversion rates were provided in request first.
				if len(requestExt.Currency.Rates) > 0 {
					// If they are set in request, convert the currencies.
					if convertedBid, err = convertCurrencyWithRates(requestExt.Currency.Rates, requestCurrency, bidCurrency, bid.Price); err != nil {
						// Error converting currency with request rates. Try with latest rates.
						if convertedBid, successfulConversion = convertCurrencyWithCurrentRates(currencyRates, &bidderErrs, requestCurrency, bidCurrency, bid); successfulConversion == false {
							continue
						}
					}
				} else {
					// Conversion rates are not set in request. Use latest rates.
					if convertedBid, successfulConversion = convertCurrencyWithCurrentRates(currencyRates, &bidderErrs, requestCurrency, bidCurrency, bid); successfulConversion == false {
						continue
					}
				}

				// Rate conversion successful. Set new bid price.
				bid.Price = convertedBid
			}
			// Rate conversion was successful or no conversion was needed. Set new currency.
			currencyToUse = requestCurrency
			newBids = append(newBids, bid)
		}

		if len(newBids) > 0 {
			// Set new bids for seat
			seatBid.Bid = newBids
			newSeatBids = append(newSeatBids, seatBid)
		}

		for _, bidderErr := range bidderErrs {
			// Set errors for this seat
			adapterExtraCopy[bidderName].Errors = append(adapterExtraCopy[bidderName].Errors, bidderErr)
		}
		*adapterExtra = adapterExtraCopy
	}
	// Set the updated list of seatBids
	*seatBids = newSeatBids

	return currencyToUse
}

func convertCurrencyWithCurrentRates(currencyRates []byte, bidderErrs *[]string, requestCurrency string, bidCurrency string, bid openrtb.Bid) (float64, bool) {
	var convertedBid float64
	var successfulConversion bool

	if len(currencyRates) > 0 {
		// Convert the currencies
		var currData currencyData
		if err := json.Unmarshal(currencyRates, &currData); err != nil || currData.ConversionRates == nil {
			*bidderErrs = append(*bidderErrs, errors.New(defaultCurrencyConversionError).Error())
		} else {
			convertedBid, err = convertCurrencyWithRates(currData.ConversionRates, requestCurrency, bidCurrency, bid.Price)
			if err != nil {
				// Error converting currencies. Drop the bid.
				*bidderErrs = append(*bidderErrs, err.Error())
			} else {
				successfulConversion = true
			}
		}
	} else {
		// If converstion rates not available, return error and drop the bid.
		*bidderErrs = append(*bidderErrs, errors.New(defaultCurrencyConversionError).Error())
	}

	return convertedBid, successfulConversion
}

func convertCurrencyWithRates(rates openrtb_ext.ConversionRates, requestCurrency string, bidCurrency string, bidPrice float64) (float64, error) {
	var conversionRate float64

	if rates[bidCurrency] != nil {
		// Direct conversion is available
		ratesToUse := rates[bidCurrency]
		if ratesToUse[requestCurrency] == 0 {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		conversionRate = ratesToUse[requestCurrency]
	} else if rates[requestCurrency] != nil {
		// Using reciprocal of conversion rate
		ratesToUse := rates[requestCurrency]
		if ratesToUse[bidCurrency] == 0 {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		conversionRate = 1 / ratesToUse[bidCurrency]
	} else {
		// Using first currency as intermediary
		var firstCurrency string
		var toIntermediateConversionRate, fromIntermediateConversionRate float64

		for currency, _ := range rates {
			firstCurrency = currency
			// Break since we just want the first currency in the list of conversions
			break
		}

		// Check if bid currency is in intermediary currency
		if bidRate, bidRateFound := rates[firstCurrency][bidCurrency]; bidRateFound {
			toIntermediateConversionRate = 1 / bidRate
		} else {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		// Check if request currency is in intermediary currency
		if requestRate, requestRateFound := rates[firstCurrency][requestCurrency]; requestRateFound {
			fromIntermediateConversionRate = requestRate
		} else {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		conversionRate = toIntermediateConversionRate * fromIntermediateConversionRate
	}

	return strconv.ParseFloat(fmt.Sprintf("%.2f", bidPrice*conversionRate), 64)
}
