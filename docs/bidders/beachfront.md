# Beachfront bidder

To use the beachfront bidder you will need an appId from an exchange 
account on [https://platform.beachfront.io](platform.beachfront.io).

For further information, please contact adops@beachfront.com.

## Some notes on mixed format requests
Internally, the Beachfront adapter has to make display vs. video requests to two separate servers.
A given request to the beachfront adapter can therefore return only display
ads or only video ads. Also, a beachfront AppId can only be configured on
[https://platform.beachfront.io](platform.beachfront.io) for one or the other. 
This can be worked around by using aliases.

In the example adUnit setup below, there is a placement for a banner ad
and a placement for a video ad. 

```javascript
    var VIDEO_APPID = "11bc5dd5-7421-4dd8-c926-40fa653bec76";
    var BANNER_APPID = "3b16770b-17af-4d22-daff-9606bdf2c9c3";

    pbjs.aliasBidder("beachfront","beachfrontBanner");
    pbjs.aliasBidder("beachfront","beachfrontVideo");
    var adUnits = [
        {
            code: "bannerImp",
            sizes: [[728, 90], [468, 60]],

            bids: [
                {
                    bidder: "beachfrontBanner",
                    params: {
                        bidfloor: 3.01,
                        appId : BANNER_APPID
                    }
                } ]
        }, {
            code: "videoImp",
            sizes: [500,380],
            mediaTypes: {
                video: {
                    mimes: ["video/mp4"],
                    w: 500,
                    h: 380
                }
            },
            bids: [
            {
                bidder: "beachfrontVideo",
                params: {
                    bidfloor: 0.06,
                    appId: VIDEO_APPID
                }
            } ]
        }
    ];


```

It is also possible to request both for a single placement slot. Notice the use of the "forceBanner" 
flag on the banner request. If that were skipped, an attempt would be made on the Beachfront backend
to get a video ad for that bid using a banner AppId, which will fail. The set up below will
return both a banner and a video ad, each with it's correct "mediaType", which can then be handled 
 programmatically by what ever criteria is appropriate. :

```javascript
    var VIDEO_APPID = "11bc5dd5-7421-4dd8-c926-40fa653bec76";
    var BANNER_APPID = "3b16770b-17af-4d22-daff-9606bdf2c9c3";

    pbjs.aliasBidder("beachfront","beachfrontBanner");
    pbjs.aliasBidder("beachfront","beachfrontVideo");
    var adUnit = {
        code: 'mmTest',
        mediaTypes: {
            video: {
                mimes: ["video/mp4"],
                context : "instream"
            },
            banner: {
                sizes: sizes
            }
        },

        bids: [
            {
                bidder: 'beachfrontBanner',
                params: {
                    bidfloor: 0.03,
                    appId : BANNER_APPID,
                    forceBanner : true
                }
            }, {
                bidder: "beachfrontVideo",
                params: {
                    bidfloor: 0.01,
                    appId: VIDEO_APPID
                }
            }
        ]
    };

```