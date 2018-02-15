# Analytics module 

This document describes how to add a new analytics module to Prebid Server. 

The PBS Analytics creates _loggable objects_ for a transaction at each endpoint (for example, an `AuctionObject` for the `/openrtb2/auction` endpoint, `CookieSyncObject` for the  `/cookiesync` endpoint, `SetUIDObject` for the `/setuid` endpoint, and an `AmpObject` for the `/amp` endpoint.)

## Steps to add new module

### In analytics package,

1. Create a file for the new module.
2. Have the new module implement `PBSAnalyticsModule` interface that allows it to extract and log necessary information from the loggable objects. 
3. Write a `NewLogger(config) (PBSAnalyticsModule, error)` method that creates, initializes and returns the Logger.

### In config package,

1. Create a struct for the module that allows its config to be read from pbs.json.
2. Add this struct to `Analytics` in `config.Configuration`.

