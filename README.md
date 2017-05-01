[![Build Status](https://travis-ci.org/prebid/prebid-server.svg?branch=master)](https://travis-ci.org/prebid/prebid-server)

# prebid-server
Server side component to offload prebid processing to the cloud

# How it works
The client (typically prebid.js) sends a JSON request to Prebid Server at `/auction`. See static/pbs_request.json for the format.
Prebid Server forms OpenRTB requests, sends them to the appropriate adapters, concatenates the responses, and returns them
to the client.

A few key points:
 * No ranking or decisioning is performed by Prebid Server. It just proxies requests.
 * No ad quality management (malware, viruses, deceptive creatives) is performed by Prebid Server
 * Prebid Server does no fraud scanning and does nothing to prevent bad traffic.

# User synching
Prebid Server provides a `/setuid` endpoint that allows adapters to push in their user IDs. These are stored in a cookie named,
creatively, `uids`. To see stored cookies, call `/getuids`. To set an optout cookie, call `/optout`. When an adapter doesn't
have a synched cookie, a `no_cookie` response is returned with a usersync URL that the client should call via asynchronous pixel
or equivalent. If Prebid Server doesn't have a cookie set, a preemptive `no_cookie` response is returned to allow the client
to ask for user consent and drop a cookie.

# Logging
Prebid Server does no server-side logging. It can stream metrics to an InfluxDB endpoint, which are aggregated as a time series.
Prebid Server has no user profiling or user-data collection capabilities.

# Data integration
Prebid Server has three primary data objects that it needs to manage:
 * Accounts represent publishers, and are used for metrics aggregation and terms of service adherence. Requests without an
 active account will be rejected.
 * Domains are compared to the HTTP Referer header; all unknown/unapproved domains will be rejected.
 * Configs are used for server-side configuration of adapters, primarily for use with mobile apps where managing configs
 client-side is ineffective.

# Up Next
 * Refine adapters and openrtb protocol (support burl? validation of responses?)
 * Support Prebid mobile SDK
 * Video and native
