# Pubnative Bidder

## Prerequisite
Before adding PubNative as a new bidder, there are 3 prerequisites:
- As a Publisher, you need to have Prebid Mobile SDK integrated.
- You need a configured Prebid Server (either self-hosted or hosted by 3rd party).
- You need to be integrated with Ad Server SDK (e.g. Mopub) or internal product which communicates with Prebid Mobile SDK.

## Configuration 

- app_auth_token is unique per publisher app. Please contact our account manager to obtain yours.
zone_id should be always set to 1 (unless special use case)
- bidder should be always set to "pubnative"

## Testing

Please consult with our Account Manager for testing. 
We need to confirm that your ad request is correctly received by our system.

The following test parameters can be used to verify that Prebid Server is working properly with the 
Appnexus adapter.

The following json can be used to do a request to prebid server for verifying it's integration with Pubnative adapter.

```json
{
    "id": "some-impression-id",
    "site": {
      "page": "https://good.site/url"
    },
    "imp": [
      {
        "id": "test-imp-id",
        "banner": {
          "format": [
            {
              "w": 300,
              "h": 250
            }
          ]
        },
        "ext": {
          "pubnative": {
            "zone_id": 1,
            "app_auth_token": "b620e282f3c74787beedda34336a4821"
          }
        }
      }
    ],
    "device": {
      "os": "android",
      "h": 700,
      "w": 375
    },
    "tmax": 500,
    "test": 1
}
```