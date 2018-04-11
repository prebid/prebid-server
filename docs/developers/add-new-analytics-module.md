# Adding a New Analytics Module

This document describes how to add a new Analytics module to Prebid Server.

### 1. Config: 

The parameters needed to setup the analytics module are sent through `configuration.analytics.{module}` 
 
### 2. Create a struct

The new analytics module belongs in the `analytics` package in _analytics/{module}.go_ file and needs to implement the `PBSAnalyticsModule` interface from `analytics/core.go`. The body of each of the `Log{loggableObject}({loggableObject})` method extracts required information from the `{loggableObject}` and is responsible for completing the logging. 

### 3. Initializing method

This belongs in `analytics` package in `/analytics/{module}.go`. It should be able to use it's configuration from the `configuration.analytics.{module}`  and initialize the struct.  

### 4. Call the initializing method while setting up PBSAnalytics

In order to log to this module, it needs to initialized inside `NewPBSAnalytics(analytics *config.Analytics) ` method.

An example of such an analytics module is the `FileLogger` in `/analytics/file_module.go`



 