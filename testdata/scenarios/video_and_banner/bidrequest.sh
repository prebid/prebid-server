#!/bin/bash

TPE_PREBID_SERVER_HOST=${TPE_PREBID_SERVER_HOST:=localhost:8000}

curl --data @../../requests/video_and_banner.json "http://${TPE_PREBID_SERVER_HOST}/openrtb2/auction" -v
