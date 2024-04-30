## Overview

51Degrees module enriches an incoming OpenRTB request [51Degrees Device Data](https://51degrees.com/documentation/_device_detection__overview.html).

51Degrees module sets the following fields of the device object: `make`, `model`, `os`, `osv`, `h`, `w`, `ppi`, `pxratio` - interested bidder adapters may use these fields as needed.  In addition the module sets `device.ext.fiftyonedegrees_deviceId` to a permanent device ID which can be rapidly looked up in on premise data exposing over 250 properties including the device age, chip set, codec support, and price, operating system and app/browser versions, age, and embedded features.

## Setup

The 51Degrees module operates using a data file. You can get started with a free Lite data file that can be downloaded [here](https://github.com/51Degrees/device-detection-data/blob/main/51Degrees-LiteV4.1.hash). The Lite file is capable of detecting limited device information, so if you need in-depth device data, please [contact 51Degrees](https://51degrees.com/contact-us?ContactReason=Free%20Trial) to obtain a license.

## Configuration

To use this module set the module's enable flag to true and add 

```fiftyone-devicedetection-entrypoint-hook``` 

and 

```fiftyone-devicedetection-raw-auction-request-hook``` 

into hooks execution plan inside your yaml file:

```json
{
  "hooks": {
    "modules": {
      "fiftyone_degrees": {
        "device_detection": {
          "enabled": true,
          "data_file": {
            "path": "path/to/51Degrees-LiteV4.1.hash",
            "update": {
              "auto": true,
              "url": "https://my.datafile.com/datafile.gz",
              "polling_interval": 3600,
              "license_key": "your_license_key",
              "product": "V4Enterprise"
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
                    "timeout": 100,
                    "hook_sequence": [
                      {
                        "module_code": "fiftyone-devicedetection",
                        "hook_impl_code": "fiftyone-devicedetection-entrypoint-hook"
                      }
                    ]
                  }
                ]
              },
              "raw_auction_request": {
                "groups": [
                  {
                    "timeout": 100,
                    "hook_sequence": [
                      {
                        "module_code": "fiftyone-devicedetection",
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

Note that at a minimum you need to enable the module and specify a path to the data file in the configuration.

## Other configuration options

```yaml
hooks:
  modules:
    fiftyone_degrees:
      device_detection:
        account_filter:
          allow_list: [] # string, list of account ids for enabled publishers, or empty for all
        data_file:
          path: ~ # string, REQUIRED
          update:
            auto: ~ # boolean
            url: ~ # string
            polling_interval: ~ # int
            license_key: ~ # string
            product: ~ # string
        performance:
          profile: ~ # string, one of [default,low_memory,balanced_temp,balanced,high_performance, in_memory]
          concurrency: ~ # int
          difference: ~ # int
          allow-unmatched: ~ # boolean
          drift: ~ # int
      

```

Minimal sample (only required):

```yaml
  modules:
    fiftyone-devicedetection:
      data-file:
        path: "51Degrees-LiteV4.1.hash" # string, REQUIRED, download the sample from https://github.com/51Degrees/device-detection-data/blob/main/51Degrees-LiteV4.1.hash or Enterprise from https://51degrees.com/pricing
```

``account_filter``
 * ``allow-list`` - (list of strings) - A list of account IDs that this module will be applied for.  If empty, it will apply to all accounts. Full-string match is performed (whitespaces and capitalization matter). Defaults to empty.

``data-file``
 * ``path`` - (string, REQUIRED) - The full path to the device detection data file. Lite data file can be downloaded from [data repo on GitHub].

 * ``update``
   * ``auto`` - (boolean) - Enable/Disable auto update. Defaults to enabled. If enabled, the auto update system will automatically download and apply new data files for device detection.
   * ``url`` - (string) - Configure the engine to use the specified URL when looking for an updated data file. Default is the 51Degrees update URL.
   * ``license-key`` - (string) - Set the License Key used when checking for new device detection data files. A License Key is exclusive to the 51Degrees paid service. Defaults to null.
   * ``product`` - (string) - Set the Product used when checking for new device detection data files. A Product is exclusive to the 51Degrees paid service. By default it is `V4Enterprise`.  Please see options [here](https://51degrees.com/documentation/_info__distributor.html).
   * ``polling-interval`` - (int, seconds) - Set the time between checks for a new data file made by the DataUpdateService in seconds. Default = 30 minutes.

``performance``
  * ``profile`` - (string) - Set the performance profile for the device detection engine. Must be one of: LowMemory, MaxPerformance, HighPerformance, Balanced, BalancedTemp. Defaults to Balanced.
  * `concurrency` - _(int)_ - Set the expected number of concurrent operations using the engine. This sets the concurrency of the internal caches to avoid excessive locking. Default: 10.
  * `difference` - _(int)_ - Set the maximum difference to allow when processing HTTP headers. The meaning of difference depends on the Device Detection API being used. The difference is the difference in hash value between the hash that was found, and the hash that is being searched for. By default this is 0. For more information see [51Degrees documentation](https://51degrees.com/documentation/_device_detection__hash.html).
  * `allow-unmatched` - _(boolean)_ - If set to false, a non-matching User-Agent will result in properties without set values.
  If set to true, a non-matching User-Agent will cause the 'default profiles' to be returned. This means that properties will always have values (i.e. no need to check .hasValue) but some may be inaccurate. By default, this is false.
  * `drift` - _(int)_ - Set the maximum drift to allow when matching hashes. If the drift is exceeded, the result is considered invalid and values will not be returned. By default this is 0. For more information see [51Degrees documentation](https://51degrees.com/documentation/_device_detection__hash.html).

## Running the demo

1. Download dependencies:
```bash
go mod download
```

2. Replace the original config file `pbs.json` (placed in the repository root or in `/etc/config`) with the sample [config file](sample/pbs.json): 
```
cp modules/fiftyone_degrees/device_detection/sample/pbs.json pbs.json
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
--data @modules/fiftyone_degrees/device_detection/sample/request_data.json
```

7. Observe the `device` object get enriched with `devicetype`, `os`, `osv`, `w`, `h` and `ext.fiftyonedegrees_deviceId`.

## Maintainer contacts

Any suggestions or questions can be directed to [support@51degrees.com](support@51degrees.com) e-mail.

Or just open new [issue](https://github.com/prebid/prebid-server/issues/new) or [pull request](https://github.com/prebid/prebid-server/pulls) in this repository.