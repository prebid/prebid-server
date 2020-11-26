Please contact <admin@we-are-adot.com> if you would like to build and deploy Prebid server and use it with Adot.

---
layout: bidder
title: Adot
description: Prebid Adot Bidder Adaptor
biddercode: adot
media_types: banner, video, native, parallax
gdpr_supported: true
prebid_member: true
userIds: none
schain_supported: true
coppa_supported: true
usp_supported: true
tcf2_supported: true
pbjs: true
pbs: true
gvl_id: 32
---


### Table of Contents

- [Bid Params](#adot-bid-params)
- [Video Object](#adot-video-object)

<a name="adot-bid-params" />

#### Bid Params
| Name                | Scope    | Description                                                                                                                                                                   | Example                                               | Type             |
|---------------------|----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------|------------------|
| `placementId`       | optional | An ID which identifies this placement of the impression.                                   | `adot_placement_224521`                                            | `string`         |
| `position`          | optional | Specify the position of the ad as a relative measure of visibility or prominence. Allowed values: Unknown: 0 (default); Above the fold: 1 ; Below the fold: 3.                                                                                                                    | `'0'`                                             | `integer`         |
| `parallax`          | optional | Specify if the wanted advertising's creative is a parallax.                                                                        | `true/false` | `boolean`         |               
| `video`             | optional | Object containing video targeting parameters.  See [Video Object](#adot-video-object) for details.                                                                        | `video: { playback_method: ['auto_play_sound_off'] }` | `object`         |                                                                                             | `'video: { mimes: ['video/mp4'] }'`                  | `object`         |

<a name="adot-video-object" />

#### Video Object

| Name              | Scope                                     | Description                                                                                                                                                                                                                                  | Type             |
|-------------------|-------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------|
| `mimes`           | required                                  | Array of strings listing the content MIME types supported, e.g., `['video/mp4']`.                                                                                                                                                            | `Array<string>`  |
| `minduration`     | optional                                  | Integer that defines the minimum video ad duration in seconds.                                                                                                                                                                               | `integer`        |
| `maxduration`     | optional                                  | Integer that defines the maximum video ad duration in seconds.                                                                                                                                                                               | `integer`        |
| `protocols`       | required                                  | Array of supported video protocols, e.g., `[2, 3]`                                                                                                                                                                                           | `Array<integer>` |
| `container`       | optional                                  | Selector used for finding the element in which the video player will be displayed, e.g., `#div-1`. The `ad unit code` will be used if no `container` is provided.                                                                            | `string`         |
| `instreamContext` | required if `video.context` is `instream` | String used to define the type of instream video. Allowed values: Pre-roll: `pre-roll`; Mid-roll: `mid-roll` ; Post-roll: `post-roll`.                                                                                                       | `string`         |


#### Adot Server Test Request

The following test parameters can be used to verify that Prebid Server is working properly with the 
server-side adot adapter. This is a mobile Bid-request example.

```
	{
  "id": "b967c495-adeb-4cf3-8f0a-0d86fa17aeb2",
  "app": {
    "id": "0",
    "name": "test-adot-integration",
    "publisher": {
      "id": "1",
      "name": "Test",
      "domain": "test.com"
    },
    "bundle": "com.example.app",
    "paid": 0,
    "domain": "test.com",
    "page": "https:\/\/www.test.com\/",
    "cat": [
      "IAB1",
      "IAB6",
      "IAB8",
      "IAB9",
      "IAB10",
      "IAB16",
      "IAB18",
      "IAB19",
      "IAB22"
    ]
  },
  "device": {
    "ua": "Mozilla\/5.0 (Linux; Android 7.0; SM-G925F Build\/NRD90M; wv) AppleWebKit\/537.36 (KHTML, like Gecko) Version\/4.0 Chrome\/80.0.3987.132 Mobile Safari\/537.36",
    "make": "phone-make",
    "model": "phone-model",
    "os": "os",
    "osv": "osv",
    "ip": "0.0.0.0",
    "ifa": "IDFA",
    "carrier": "WIFI",
    "language": "English",
    "geo": {
      "zip": "75001",
      "country": "FRA",
      "type": 2,
      "lon": 48.2,
      "lat": 2.32,
      "accuracy": 100
    },
    "ext": {
      "is_app": 1
    },
    "connectiontype": 2,
    "devicetype": 4
  },
  "user": {
    "id": "IDFA",
    "buyeruid": ""
  },
  "imp": [
    {
      "id": "dec4147e-a63f-4d25-9fff-da9bfd05bd02",
      "banner": {
        "w": 320,
        "h": 50,
        "format": [
          {
            "w": 320,
            "h": 50
          }
        ],
        "api": [
          1,
          2,
          5,
          6,
          7
        ]
      },
      "bidfloorcur": "USD",
      "bidfloor": 0.1,
      "instl": 0,
      "ext": {
        "adot": {
        }
      }
    }
  ],
  "cur": [
    "USD"
  ],
  "regs": {
    "ext": {
      "gdpr": 1
    }
  },
  "at": 1
}
```