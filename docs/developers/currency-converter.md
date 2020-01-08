**For the time being, currency conversion is not enabled, feature is still under dev (check #280).**

# Currency Converter Mechanics

Prebid server supports currency conversions when receiving bids.

## Default currency

The default currency is `USD`. It means that any bids coming without an explicit currency will be interpreted as being `USD`.

## Setup

By default, the currency converter uses https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json for currency conversion. This data is updated every 24 hours on prebid.org side.
By default, currency conversions are updated from the endpoint every 30 minutes in prebid server.

Default configuration:
```
v.SetDefault("currency_converter.fetch_url", "https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json")
v.SetDefault("currency_converter.fetch_interval_seconds", 1800) // 30 minutes
```

This configuration can be changed:
- currency_converter.fetch_url can be any URL exposing currency using the following JSON schema:
  ```
  {
      "dataAsOf":"2018-09-12",
      "conversions":{
          "USD":{
              "GBP":0.77208
          },
          "GBP":{
              "USD":1.2952
          }
      }
  }
  ```
- currency_converter.fetch_interval_seconds can be anything from 0 to max int.
  **The currency conversion mechanism can be disable by setting it to 0, in this case, there will be no currency conversions at all and all bidders will need to provide bids as `USD`**

 ## Examples

 Here are couple examples showing the logic behind the currency converter:

| Bidder bid price | Currency      | Rate to USD   | Rate converter is active | Converted bid price (USD) | Valid bid |
| :--------------- | :------------ |:--------------| :------------------------| :-------------------------|:----------|
| 1                | USD           |             1 | YES                      |                         1 | YES       |
| 1                | N/A           |             1 | YES                      |                         1 | YES       |
| 1                | USD           |             1 | NO                       |                         1 | YES       |
| 1                | EUR           |          1.13 | YES                      |                      1.13 | YES       |
| 1                | EUR           |           N/A | YES                      |                       N/A | NO        |
| 1                | EUR           |          1.13 | NO                       |                       N/A | NO        |

## Debug

A dedicated endpoint will allow you to see what's happening within the currency converter.
See [currency rates endpoint](../endpoints/currency_rates.md) for more details.
