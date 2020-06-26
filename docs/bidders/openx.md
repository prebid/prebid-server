# OpenX Bidder

OpenX supports the following parameters:

| property | type | required? | description | example |
|----------|------|-----------|-------------|---------|
| unit | string | required | The ad unit id | "10092842" |
| delDomain | string | required | The delivery domain for the customer | "sademo-d.openx.net" |
| customFloor | number | optional | The minimum CPM price in USD | 1.50 - sets a $1.50 floor |
| customParams | object | optional | User-defined targeting key-value pairs | {key1: "v1", key2: ["v2","v3"]} |

If you have any questions regarding setting up, please reach out to your account manager or 
<support@openx.com>

## Test Request

### App Impression Object
```
{
  "id": "test-impression-id",
  "banner": {
    "format": [
      {
        "w": 480,
        "h": 300
      },
      {
        "w": 480,
        "h": 320
      }
    ]
  },
  "ext": {
    "openx": {
      "delDomain": "mobile-d.openx.net",
      "unit": "541028953"
    }
  }
}
```


### Web
```
{
  "id": "div1",
  "banner": {
    "format": [
      {
        "w": 728,
        "h": 90
      }
    ]
  },
  "ext": {
    "openx": {
      "unit": "540949380",
      "delDomain": "sademo-d.openx.net"
    },
  }
}
```