package exchange

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

const latestConversionRatesUrl = "http://currency.prebid.org/latest.json"
const defaultCurrencyConversionError = "No currency conversion available."

type currencyData struct {
	DataAsOfDate string      `json:"dataAsOf"`
	Conversions  conversions `json:"conversions"`
}

type conversions map[string]rates

type rates map[string]float64

func getCurrency(seatBids *[]openrtb.SeatBid, bidRequest *openrtb.BidRequest, adapterExtra *map[openrtb_ext.BidderName]*seatResponseExtra) string {
	currencyToUse := defaultAdServerCurrency
	var requestExt openrtb_ext.ExtRequest

	if bidRequest.Ext != nil {
		if err := json.Unmarshal(bidRequest.Ext, &requestExt); err != nil {
			// Error unmarshalling bidRequest.Ext. Treat as if currency was not set in request.
			return currencyToUse
		}

		requestCurrency := strings.ToUpper(requestExt.Currency.AdServerCurrency)
		// First check if currency was set in request
		if requestCurrency != "" {
			newSeatBids := make([]openrtb.SeatBid, 0, len(*seatBids))

			// Iterate through bidder currencies and make sure they match with request currency
			for _, seatBid := range *seatBids {
				var bidderErrs []interface{}
				bidderName, ok := openrtb_ext.GetBidderName(seatBid.Seat)
				if ok == false {
					// Invalid bidder name in SeatBid.Seat. Skip to the next seat.
					continue
				}
				adapterExtraCopy := *adapterExtra
				newBids := make([]openrtb.Bid, 0, len(seatBid.Bid))

				for _, bid := range seatBid.Bid {
					// Fetch currency from bid
					bidCurrency, err := jsonparser.GetString(bid.Ext, "ad_server_currency")
					if err != nil || bidCurrency == "" {
						if err == nil && requestCurrency == defaultAdServerCurrency {
							// If bidder did not provide bid currency but request currency was set to USD (default), use USD
							currencyToUse = defaultAdServerCurrency
							newBids = append(newBids, bid)
							continue
						}

						// Currency was set in request but bidder did not provide currency info in bid. Return error and drop the bid.
						bidderErrs = append(bidderErrs, errors.New(defaultCurrencyConversionError).Error())
						continue
					}

					bidCurrency = strings.ToUpper(bidCurrency)

					if bidCurrency != requestCurrency {
						var convertedBid float64
						var successfulConversion bool
						// Currencies do not match for this bid. Need to convert the bid price based on converstion rates.
						// Check if conversion rates were provided in request first.
						if requestExt.Currency.Rates != nil {
							// If they are set in request, convert the currencies.
							var convns conversions
							if err := json.Unmarshal(bytes.ToUpper(requestExt.Currency.Rates), &convns); err != nil {
								bidderErrs = append(bidderErrs, errors.New(defaultCurrencyConversionError).Error())
								continue
							}

							if convertedBid, err = convertCurrency(convns, requestCurrency, bidCurrency, bid.Price); err != nil {
								// Error converting currency with request rates. Try with latest rates.
								if convertedBid, successfulConversion = convertCurrencyWithCurrentRates(&bidderErrs, requestCurrency, bidCurrency, bid); successfulConversion == false {
									continue
								}
							}
						} else {
							// Conversion rates are not set in request. Use latest rates.
							if convertedBid, successfulConversion = convertCurrencyWithCurrentRates(&bidderErrs, requestCurrency, bidCurrency, bid); successfulConversion == false {
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
					adapterExtraCopy[bidderName].Errors = append(adapterExtraCopy[bidderName].Errors, fmt.Sprintf("%s", bidderErr))
				}
				*adapterExtra = adapterExtraCopy
			}
			// Set the updated list of seatBids
			*seatBids = newSeatBids
		}
	}

	return currencyToUse
}

func convertCurrencyWithCurrentRates(bidderErrs *[]interface{}, requestCurrency string, bidCurrency string, bid openrtb.Bid) (float64, bool) {
	var convertedBid float64
	var successfulConversion bool
	conversionRates := fetchCurrencyConversionRates()

	if conversionRates == nil {
		// If converstion rates not available, return error and drop the bid.
		*bidderErrs = append(*bidderErrs, errors.New(defaultCurrencyConversionError).Error())
	} else {
		// Convert the currencies
		var currData currencyData
		if err := json.Unmarshal(conversionRates, &currData); err != nil {
			*bidderErrs = append(*bidderErrs, errors.New(defaultCurrencyConversionError).Error())
		} else {
			convertedBid, err = convertCurrency(currData.Conversions, requestCurrency, bidCurrency, bid.Price)
			if err != nil {
				// Error converting currencies. Drop the bid.
				*bidderErrs = append(*bidderErrs, err.Error())
			} else {
				successfulConversion = true
			}
		}
	}

	return convertedBid, successfulConversion
}

func fetchCurrencyConversionRates() []byte {
	var rates []byte
	// Fetching latest currency conversion rates
	resp, err := http.Get(latestConversionRatesUrl)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	rates, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return rates
}

func convertCurrency(conversionRates conversions, requestCurrency string, bidCurrency string, bidPrice float64) (float64, error) {
	var conversionRate float64

	if conversionRates[bidCurrency] != nil {
		// Direct conversion is available
		rates := conversionRates[bidCurrency]
		if rates[requestCurrency] == 0 {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		conversionRate = rates[requestCurrency]
	} else if conversionRates[requestCurrency] != nil {
		// Using reciprocal of conversion rate
		rates := conversionRates[requestCurrency]
		if rates[bidCurrency] == 0 {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		conversionRate = 1 / rates[bidCurrency]
	} else {
		// Using first currency as intermediary
		var firstCurrency string
		var toIntermediateConversionRate, fromIntermediateConversionRate float64

		for currency, _ := range conversionRates {
			firstCurrency = currency
			// Break since we just want the first currency in the list of conversions
			break
		}

		// Check if bid currency is in intermediary currency
		if bidRate, bidRateFound := conversionRates[firstCurrency][bidCurrency]; bidRateFound {
			toIntermediateConversionRate = 1 / bidRate
		} else {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		// Check if request currency is in intermediary currency
		if requestRate, requestRateFound := conversionRates[firstCurrency][requestCurrency]; requestRateFound {
			fromIntermediateConversionRate = requestRate
		} else {
			// Currency not supported
			return 0, errors.New(defaultCurrencyConversionError)
		}

		conversionRate = toIntermediateConversionRate * fromIntermediateConversionRate
	}

	return strconv.ParseFloat(fmt.Sprintf("%.2f", bidPrice*conversionRate), 64)
}
