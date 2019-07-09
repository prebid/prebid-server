# Beachfront bidder

To use the beachfront bidder you will need an appId from an exchange 
account on [https://platform.beachfront.io](platform.beachfront.io).

For further information, please contact adops@beachfront.com.

The beachfront bidder is capable of delivering banner and video bids, but
to get back both banner and video requires setting up some simple aliases. 
In the example adUnit setup below, there is a placement for a banner ad
and a placement for a video ad. It would also be possible to request
both for a single placement slot.
```javascript
    pbjs.aliasBidder("beachfront","beachfrontVideo");
    var adUnits = [
        {
            code: "banner",
            sizes: [[728, 90], [468, 60]],

            bids: [
                {
                    bidder: "beachfront",
                    params: {
                        bidfloor: 3.01,
                        appId : BANNER_APPID
                    }
                }, {
                    bidder: "appnexus",
                    params: {
                       placementId: "13144370"
                    }
                } ]
        }, {
            code: "video",
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
            }, {
                bidder: "appnexusAst",
                params: {
                  placementId: '123456',
                  video: {
                    id: 123,
                    skipppable: true,
                    playback_method: ['auto_play_sound_off']
                  }
                }
            } ]
        }
    ];


```


