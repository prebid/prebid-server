## Overview

The 51Degrees module enriches an incoming OpenRTB request with [51Degrees Device Data](https://51degrees.com/documentation/_device_detection__overview.html).

The module sets the following fields of the device object: `make`, `model`, `os`, `osv`, `h`, `w`, `ppi`, `pxratio` - interested bidder adapters may use these fields as needed.  In addition the module sets `device.ext.fiftyonedegrees_deviceId` to a permanent device ID which can be rapidly looked up in on premise data exposing over 250 properties including the device age, chip set, codec support, and price, operating system and app/browser versions, age, and embedded features.

## Operation Details

### Evidence

The module uses `device.ua` (User Agent) and `device.sua` (Structured User Agent) provided in the oRTB request payload as input (or 'evidence' in 51Degrees terminology).  There is a fallback to the corresponding HTTP request headers if any of these are not present in the oRTB payload - in particular: `User-Agent` and `Sec-CH-UA-*` (aka User-Agent Client Hints).  To make sure Prebid.js sends Structured User Agent in the oRTB payload - we strongly advice publishers to enable [First Party Data Enrichment module](dev-docs/modules/enrichmentFpdModule.html) for their wrappers and specify

```js
pbjs.setConfig({
    firstPartyData: {
        uaHints: [
          'architecture',
          'model',
          'platform',
          'platformVersion',
          'fullVersionList',
        ]
    }
})
```

### Data File Updates

The module operates **fully autonomously and does not make any requests to any cloud services in real time to do device detection**. This is an [on-premise data](https://51degrees.com/developers/deployment-options/on-premise-data) deployment in 51Degrees terminology. The module operates using a local data file that is loaded into memory fully or partially during operation. The data file is occasionally updated to accomodate new devices, so it is recommended to enable automatic data updates in the module configuration. Alternatively `watch_file_system` option can be used and the file may be downloaded and replaced on disk manually. See the configuration options below.

## Setup

The 51Degrees module operates using a data file. You can get started with a free Lite data file that can be downloaded here: [51Degrees-LiteV4.1.hash](https://github.com/51Degrees/device-detection-data/blob/main/51Degrees-LiteV4.1.hash).  The Lite file is capable of detecting limited device information, so if you need in-depth device data, please contact 51Degrees to obtain a license: [https://51degrees.com/contact-us](https://51degrees.com/contact-us?ContactReason=Free%20Trial).

Put the data file in a file system location writable by the system account that is running the Prebid Server module and specify that directory location in the configuration parameters. The location needs to be writable if you would like to enable [automatic data file updates](https://51degrees.com/documentation/_features__automatic_datafile_updates.html).

### Execution Plan

This module supports running at two stages:

* entrypoint: this is where incoming requests are parsed and device detection evidences are extracted.
* raw-auction-request: this is where outgoing auction requests to each bidder are enriched with the device detection data

We recommend defining the execution plan right in the account config
so the module is only invoked for specific accounts. See below for an example.

### Global Config

There is no host-company level config for this module.

### Account-Level Config

To start using current module in PBS-Go you have to enable module and add `fiftyone-devicedetection-entrypoint-hook` and `fiftyone-devicedetection-raw-auction-request-hook` into hooks execution plan inside your config file:
Here's a general template for the account config used in PBS-Go:

```json
{
  "hooks": {
    "enabled":true,
    "modules": {
      "fiftyonedegrees": {
        "devicedetection": {
          "enabled": true,
          "make_temp_copy": true,
          "data_file": {
            "path": "path/to/51Degrees-LiteV4.1.hash",
            "update": {
              "auto": true,
              "url": "<optional custom URL>",
              "polling_interval": 1800,
              "license_key": "<your_license_key>",
              "product": "V4Enterprise",
              "watch_file_system": "true",
              "on_startup": true
            }
          }
        }
      },
      "host_execution_plan": {
        "endpoints": {
          "/openrtb2/auction": {
            "stages": {
              "entrypoint": {
                "groups": [
                  {
                    "timeout": 10,
                    "hook_sequence": [
                      {
                        "module_code": "fiftyonedegrees.devicedetection",
                        "hook_impl_code": "fiftyone-devicedetection-entrypoint-hook"
                      }
                    ]
                  }
                ]
              },
              "raw_auction_request": {
                "groups": [
                  {
                    "timeout": 10,
                    "hook_sequence": [
                      {
                        "module_code": "fiftyonedegrees.devicedetection",
                        "hook_impl_code": "fiftyone-devicedetection-raw-auction-request-hook"
                      }
                    ]
                  }
                ]
              }
            }
          }
        }
      }
    }
  }
}
```

The same config in YAML format:
```yaml
hooks:
  enabled: true
  modules:
    fiftyonedegrees:
      devicedetection:
        enabled: true
        make_temp_copy: true
        data_file:
          path: path/to/51Degrees-LiteV4.1.hash
          update:
            auto: true
            url: "<optional custom URL>"
            polling_interval: 1800
            license_key: "<your_license_key>"
            product: V4Enterprise
            watch_file_system: 'true'
    host_execution_plan:
      endpoints:
        "/openrtb2/auction":
          stages:
            entrypoint:
              groups:
                - timeout: 10
                  hook_sequence:
                    - module_code: fiftyonedegrees.devicedetection
                      hook_impl_code: fiftyone-devicedetection-entrypoint-hook
            raw_auction_request:
              groups:
                - timeout: 10
                  hook_sequence:
                    - module_code: fiftyonedegrees.devicedetection
                      hook_impl_code: fiftyone-devicedetection-raw-auction-request-hook
```

Note that at a minimum (besides adding to the host_execution_plan) you need to enable the module and specify a path to the data file in the configuration.
Sample module enablement configuration in JSON and YAML formats:

```json
{
  "modules": {
    "fiftyonedegrees": {
      "devicedetection": {
        "enabled": true,
        "data_file": {
          "path": "path/to/51Degrees-LiteV4.1.hash"
        }
      }
    }
  }
}
```

```yaml
  modules:
    fiftyonedegrees:
      devicedetection: 
        enabled: true
        data_file:
          path: "/path/to/51Degrees-LiteV4.1.hash"
```

## Module Configuration Parameters

The parameter names are specified with full path using dot-notation.  F.e. `section_name` .`sub_section` .`param_name` would result in this nesting in the JSON configuration:

```json
{
  "section_name": {
    "sub_section": {
      "param_name": "param-value"
    }
  }
}
```

| Param Name | Required| Type | Default  value | Description |
|:-------|:------|:------|:------|:---------------------------------------|
| `account_filter` .`allow_list`  |  No | list of strings | [] (empty list) | A list of account IDs that are allowed to use this module - only relevant if enabled globally for the host. If empty, all accounts are allowed. Full-string match is performed (whitespaces and capitalization matter). |
| `data_file` .`path`  |  **Yes** | string | null |The full path to the device detection data file. Sample file can be downloaded from [data repo on GitHub](https://github.com/51Degrees/device-detection-data/blob/main/51Degrees-LiteV4.1.hash), or get an Enterprise data file [here](https://51degrees.com/pricing). |
| `data_file` .`make_temp_copy` | No | boolean | true | If true, the engine will create a temporary copy of the data file rather than using the data file directly. |
| `data_file` .`update` .`auto` | No | boolean | true | If enabled, the engine will periodically (at predefined time intervals - see `polling-interval` parameter) check if new data file is available. When the new data file is available engine downloads it and switches to it for device detection. If custom `url` is not specified `license_key` param is required. |
| `data_file` .`update` .`on_startup` | No | boolean | false | If enabled, engine will check for the updated data file right away without waiting for the defined time interval. |
| `data_file` .`update` .`url` | No | string | null | Configure the engine to check the specified URL for the availability of the updated data file. If not specified the [51Degrees distributor service](https://51degrees.com/documentation/4.4/_info__distributor.html) URL will be used, which requires a License Key. |
| `data_file` .`update` .`license_key` | No | string | null | Required if `auto` is true and custom `url` is not specified. Allows to download the data file from the [51Degrees distributor service](https://51degrees.com/documentation/4.4/_info__distributor.html). |
| `data_file` .`update` .`watch_file_system` | No | boolean | true | If enabled the engine will watch the data file path for any changes, and automatically reload the data file from disk once it is updated. |
| `data_file` .`update` .`polling_interval` | No | int | 1800 | The time interval in seconds between consequent attempts to download an updated data file. Default = 1800 seconds = 30 minutes. |
| `data_file` .`update` .`product`| No | string | `V4Enterprise` | Set the Product used when checking for new device detection data files. A Product is exclusive to the 51Degrees paid service. Please see options [here](https://51degrees.com/documentation/_info__distributor.html). |
| `performance` .`profile` | No | string | `Balanced` | `performance.*` parameters are related to the tradeoffs between speed of device detection and RAM consumption or accuracy. `profile` dictates the proportion between the use of the RAM (the more RAM used - the faster is the device detection) and reads from disk (less RAM but slower device detection). Must be one of: `LowMemory`, `MaxPerformance`, `HighPerformance`, `Balanced`, `BalancedTemp`, `InMemory`. Defaults to `Balanced`.  |
| `performance` .`concurrency` | No | int | 10 |  Specify the expected number of concurrent operations that engine does. This sets the concurrency of the internal caches to avoid excessive locking. Default: 10.  |
| `performance` .`difference` | No | int | 0 |  Set the maximum difference to allow when processing evidence (HTTP headers). The meaning is the difference in hash value between the hash that was found, and the hash that is being searched for. By default this is 0. For more information see [51Degrees documentation](https://51degrees.com/documentation/_device_detection__hash.html).  |
| `performance` .`drift` | No | int | 0 |  Set the maximum drift to allow when matching hashes. If the drift is exceeded, the result is considered invalid and values will not be returned. By default this is 0. For more information see [51Degrees documentation](https://51degrees.com/documentation/_device_detection__hash.html).  |
| `performance` .`allow_unmatched` | No | boolean | false |  If set to false, a non-matching evidence will result in properties with no values set. If set to true, a non-matching evidence will cause the 'default profiles' to be returned. This means that properties will always have values (i.e. no need to check .hasValue) but some may be inaccurate. By default, this is false. |

## Running the demo

1. Download dependencies:
```bash
go mod download
```

2. Replace the original config file `pbs.json` (placed in the repository root or in `/etc/config`) with the sample [config file](sample/pbs.json):
```
cp modules/fiftyonedegrees/devicedetection/sample/pbs.json pbs.json
```

3. Download `51Degrees-LiteV4.1.hash` from [[GitHub](https://github.com/51Degrees/device-detection-data/blob/main/51Degrees-LiteV4.1.hash)] and put it in the project root directory.

```bash
curl -o 51Degrees-LiteV4.1.hash -L https://github.com/51Degrees/device-detection-data/raw/main/51Degrees-LiteV4.1.hash
```

4. Create a directory for sample stored requests (needed for the server to run):
```bash
mkdir -p sample/stored
```

5. Start the server:
```bash
go run main.go
```

6. Run sample request:
```bash
curl \
--header "Content-Type: application/json" \
http://localhost:8000/openrtb2/auction \
--data @modules/fiftyonedegrees/devicedetection/sample/request_data.json
```

7. Observe the `device` object get enriched with `devicetype`, `os`, `osv`, `w`, `h` and `ext.fiftyonedegrees_deviceId`.

## Maintainer contacts

Any suggestions or questions can be directed to [support@51degrees.com](support@51degrees.com) e-mail.

Or just open new [issue](https://github.com/prebid/prebid-server/issues/new) or [pull request](https://github.com/prebid/prebid-server/pulls) in this repository.
