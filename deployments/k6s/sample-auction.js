import http from "k6/http";

const url = "https://prebid.sandbox.adthrive.com/openrtb2/auction";
const jsonBody = {
  imp: [
    {
      ext: {
        data: {
          hvp: 80,
          pos: "atf",
          adserver: {
            name: "gam",
            adslot: "/18190176/AdThrive_Footer_1/60a7fa14d53602489a3692c6",
          },
          pbadslot: "/18190176/AdThrive_Footer_1/60a7fa14d53602489a3692c6",
        },
        tid: "aa470d66-187e-4ee4-8228-bc8c873f28c1",
        gpid: "/18190176/AdThrive_Footer_1/60a7fa14d53602489a3692c6",
        prebid: {
          bidder: {
            tripl_ss: { inventoryCode: "adthrive_footer_1_hdx_pbs2s" },
            grid: { uid: 367 },
            opnx_ss: {
              delDomain: "cafemedia-d.openx.net",
              unit: "558246006",
              customParams: {
                sens: [
                  "alc",
                  "ast",
                  "cbd",
                  "conl",
                  "cosm",
                  "dat",
                  "drg",
                  "gamc",
                  "gamv",
                  "pol",
                  "rel",
                  "sst",
                  "ssr",
                  "srh",
                  "tob",
                  "wtl",
                ],
                bucket: ["727858c:ovrd"],
              },
            },
            pubm_ss: {
              publisherId: "157347",
              adSlot: "Footer1_XandrS2S@728x90",
              pmzoneid:
                "alc,ast,cbd,conl,cosm,dat,drg,gamc,gamv,pol,rel,sst,ssr,srh,tob,wtl",
            },
            yah_ss: {
              dcn: "8a9698af01888852cc626cda97350021",
              pos: "8a9691d801888852d23d6cdc9f220029",
            },
            yieldmo: { placementId: "2626064683911553041" },
            conversant: { tag_id: "1b2dec1", site_id: "203587" },
            "33across": { siteId: "aMbGkS_Lur6ikXaKkGJozW", productId: "siab" },
            unruly: { siteId: 249232 },
            col_ss: { groupId: "466" },
            improve_ss: { publisherId: 2250, placementId: 22983142 },
          },
          adunitcode: "AdThrive_Footer_1_tablet",
          floors: { floorMin: 1.8663 },
        },
      },
      id: "AdThrive_Footer_1_tablet",
      banner: {
        topframe: 1,
        format: [
          { w: 728, h: 90 },
          { w: 320, h: 50 },
          { w: 300, h: 50 },
          { w: 320, h: 100 },
          { w: 468, h: 60 },
          { w: 1, h: 1 },
        ],
        pos: 1,
      },
      bidfloor: 1.8663,
      bidfloorcur: "USD",
      secure: 1,
    },
  ],
  cur: ["USD"],
  at: 1,
  device: {
    w: 919,
    h: 931,
    dnt: 0,
    ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
    language: "en",
    deviceType: 2,
    js: 1,
    sua: {
      source: 1,
      platform: { brand: "Windows" },
      browsers: [
        { brand: "Google Chrome", version: ["117"] },
        { brand: "Not;A=Brand", version: ["8"] },
        { brand: "Chromium", version: ["117"] },
      ],
      mobile: 0,
    },
  },
  site: {
    domain: "disneydining.com",
    publisher: { domain: "disneydining.com", id: "1234" },
    page: "https://www.disneydining.com/?commit=727858c",
    content: { language: "en" },
    cat: ["IAB20"],
    pagecat: ["IAB20"],
    privacypolicy: 1,
    mobile: 1,
  },
  user: {
    data: [
      { name: "cafemedia.com", segment: [{ id: "719" }], ext: { segtax: 4 } },
    ],
    ext: {
      eids: [
        {
          source: "yahoo.com",
          uids: [
            {
              id: "eClrZfEA3dKjTu-I6V540P7QjTiq7-W6iWbAerf6HEJDRn1Z45u9O0UuNVLlXFwZqaf8wZaGh1Xp05QgJNkJjw",
              atype: 3,
            },
          ],
        },
        {
          source: "criteo.com",
          uids: [
            {
              id: "38lz4V82cyUyRndpN3p6cldxWmlpNk9IaFFCWWN3ajhQZHdGJTJGcVBkSlREZDZ3YU52QnV3WGhOS05PMVc4VmtMV3lLQ1l4cVowNGN6dlJLTlc2YklVVU9aQ2duMmowN1hocnFJJTJCeFRRcXFOQUtMNDVmTWxiTTdNYVZBRER6U3JpVVUzb05CNg",
              atype: 1,
            },
          ],
        },
        {
          source: "liveramp.com",
          uids: [
            {
              id: "AnJj72q5uIjshQSJhY_zmUxZxLAvbauLQ_IVvHX40uvIJqSRG0qcYobAN3eXLsGKFQzXRPF5voSFDZqP0A3zg5KIWkXnrKwLzsFJ",
              atype: 3,
            },
          ],
        },
        {
          source: "pubcid.org",
          uids: [{ id: "a3446f63-b768-4c36-92e5-0e3805175be7", atype: 1 }],
        },
        {
          source: "adserver.org",
          uids: [
            {
              id: "58b8d741-d28e-44c3-b73f-e3de33e9583a",
              atype: 1,
              ext: { rtiPartner: "TDID" },
            },
          ],
        },
        {
          source: "neustar.biz",
          uids: [
            {
              id: "E1:PZr1HqIfOLPm_2YNZ7UsPz_JZFQv9veqg2Bsd-UzBdmscbsMxEdsn6tKVFrZ1Nhso3DienVw-fmsEOMBmK05p_W09CjXF2MooE0xnhiprGiAniko7KXgPBXrHkM4dLno",
              atype: 1,
            },
          ],
        },
        {
          source: "flashtalking.com",
          uids: [
            {
              id: "eb54fa440d4246438d4d31a7fd44d510",
              atype: 1,
              ext: {
                HHID: "c795f5e209d84d5987e49a4c876930b3",
                DeviceID: "eb54fa440d4246438d4d31a7fd44d510",
                SingleDeviceID: "56b7c3fd293b4aa5bee2b24499579cf2",
              },
            },
          ],
        },
        {
          source: "liveintent.com",
          uids: [{ id: "4KjsbM-t8lQyjtQbnPTFIL4iT0VqXsAuW-0Bew", atype: 3 }],
        },
        {
          source: "bidswitch.net",
          uids: [
            {
              id: "d2da4bc2-d678-4c19-b38e-72b90d105b97",
              atype: 3,
              ext: { provider: "liveintent.com" },
            },
          ],
        },
        {
          source: "media.net",
          uids: [
            {
              id: "3359086558895554000V10",
              atype: 3,
              ext: { provider: "liveintent.com" },
            },
          ],
        },
        {
          source: "rubiconproject.com",
          uids: [
            {
              id: "L89FM1GU-23-MG2S",
              atype: 3,
              ext: { provider: "liveintent.com" },
            },
          ],
        },
        {
          source: "liveintent.indexexchange.com",
          uids: [
            {
              id: "ZOe2Lrc9A420xM3ViRdKmQAA&5592",
              atype: 3,
              ext: { provider: "liveintent.com" },
            },
          ],
        },
        {
          source: "crwdcntrl.net",
          uids: [
            {
              id: "1fd3242ccbe24323abf262201d174945a702129d27cf8035843522d4a0947fd3",
              atype: 1,
            },
          ],
        },
      ],
    },
  },
  source: {
    fd: 0,
    ext: {
      schain: {
        ver: "1.0",
        complete: 1,
        nodes: [
          { asi: "cafemedia.com", sid: "60a7fa14d53602489a3692c6", hp: 1 },
        ],
      },
    },
  },
  regs: {
    gpp: "DBABzw~1YNY~BVQqAAAAAgA",
    gpp_sid: [6, 7],
    ext: { us_privacy: "1YNY" },
  },
  ext: {
    prebid: {
      auctiontimestamp: 1695654373561,
      targeting: { includewinners: true, includebidderkeys: false },
      aliases: {
        improve_ss: "improvedigital",
        tripl_ss: "triplelift",
        opnx_ss: "openx",
        pubm_ss: "pubmatic",
        yah_ss: "yahoossp",
        col_ss: "colossus",
      },
      floors: { enabled: false, floorMin: 1.8663, floorMinCur: "USD" },
      bidderconfig: [
        {
          bidders: ["tripl_ss"],
          config: {
            ortb2: {
              site: {
                ext: {
                  data: {
                    sens: [
                      "alc",
                      "ast",
                      "cbd",
                      "conl",
                      "cosm",
                      "dat",
                      "drg",
                      "gamc",
                      "gamv",
                      "pol",
                      "rel",
                      "sst",
                      "ssr",
                      "srh",
                      "tob",
                      "wtl",
                    ],
                  },
                },
              },
            },
          },
        },
      ],
      schains: [
        {
          bidders: [
            "tripl_ss",
            "grid",
            "opnx_ss",
            "pubm_ss",
            "yah_ss",
            "yieldmo",
            "conversant",
            "33across",
            "unruly",
            "col_ss",
            "improve_ss",
            "rubi_ss",
          ],
          schain: {
            ver: "1.0",
            complete: 1,
            nodes: [
              { asi: "cafemedia.com", sid: "60a7fa14d53602489a3692c6", hp: 1 },
            ],
          },
        },
      ],
      channel: { name: "pbjs", version: "v8.5.0" },
    },
  },
  id: "98108d5f-960b-427d-9e7e-af708acc2281",
  test: 1,
  tmax: 2300,
};

export const options = {
  ext: {
    loadimpact: {
      // Project: Prebid Server
      projectID: 3658473,
      // Test runs with the same name groups test runs together
      name: "Basic Prebid Request",
    },
  },
  scenarios: {
    contacts: {
      executor: "ramping-arrival-rate",

      // Start iterations per `timeUnit`
      startRate: 6500,

      // Start `startRate` iterations per minute
      timeUnit: "1s",

      // Pre-allocate necessary VUs.
      preAllocatedVUs: 10000,

      stages: [
        // Linearly ramp-up to starting 600 iterations per `timeUnit` over the following two minutes.
        { target: 6500, duration: "3m" },
      ],
    },
  },
};

export default function () {
  // Using a JSON string as body
  let res = http.post(url, JSON.stringify(jsonBody), {
    headers: { "Content-Type": "application/json" },
  });
}
