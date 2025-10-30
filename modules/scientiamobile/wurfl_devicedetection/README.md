## WURFL Device Enrichment Module

### Overview

The **WURFL Device Enrichment Module** for Prebid Server enhances the OpenRTB 2.x payload
with comprehensive device detection data powered by **ScientiaMobile**’s WURFL device detection framework.
Thanks to WURFL's device database, the module provides accurate and comprehensive device-related information,
enabling bidders to make better-informed targeting and optimization decisions.

### Key features

#### Device Field Enrichment

The WURFL module populates **missing or empty fields** in `ortb2.device` with the following data:

- **make**: Manufacturer of the device (e.g., "Apple", "Samsung").
- **model**: Device model (e.g., "iPhone 14", "Galaxy S22").
- **os**: Operating system (e.g., "iOS", "Android").
- **osv**: Operating system version (e.g., "16.0", "12.0").
- **h**: Screen height in pixels.
- **w**: Screen width in pixels.
- **ppi**: Screen pixels per inch (PPI).
- **pxratio**: Screen pixel density ratio.
- **devicetype**: Device type (e.g., mobile, tablet, desktop).
- **js**: Support for JavaScript, where 0 = no, 1 = yes

> **Note**: If these fields are already populated in the bid request, the module will not overwrite them.

#### Publisher-Specific Enrichment

Device enrichment is selectively enabled for publishers based on their **account ID**.
The module identifies publishers through the following fields:

- `site.publisher.id` (for web environments).
- `app.publisher.id` (for mobile app environments).
- `dooh.publisher.id` (for digital out-of-home environments).

### Build prerequisites

To build the WURFL module, you need to install the WURFL Infuze from ScientiaMobile.
For more details, visit: [ScientiaMobile WURFL Infuze](https://docs.scientiamobile.com/documentation/infuze/infuze-c-api-user-guide).

#### Note

The WURFL module requires CGO at compile time to link against the WURFL Infuze library.

To enable the WURFL module, the `wurfl` build tag must be specified:

```go
go build -tags wurfl .
```

If the `wurfl` tag is not provided, the module will compile a demo version that returns sample data,
allowing basic testing without an Infuze license.

### Configuring the WURFL Module

Below is a sample configuration for the WURFL module:

```json
{
  "adapters": [
    {
      "appnexus": {
        "enabled": true
      }
    }
  ],
  "gdpr": {
    "enabled": true,
    "default_value": 0,
    "timeouts_ms": {
      "active_vendorlist_fetch": 900000
    }
  },
  "hooks": {
    "enabled": true,
    "modules": {
      "scientiamobile": {
        "wurfl_devicedetection": {
          "enabled": true,
          "wurfl_file_path": "/path/to/wurfl.zip",
          "wurfl_snapshot_url": "<wurfl_snapshot_url>",
          "wurfl_cache_size": 200000,
          "allowed_publisher_ids": ["1","3"],
          "ext_caps": true
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
                      "module_code": "scientiamobile.wurfl_devicedetection",
                      "hook_impl_code": "scientiamobile-wurfl_devicedetection-entrypoint-hook"
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
                      "module_code": "scientiamobile.wurfl_devicedetection",
                      "hook_impl_code": "scientiamobile-wurfl_devicedetection-raw-auction-request-hook"
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
```

The same configuration in YAML format

```yaml
adapters:
  - appnexus:
      enabled: true
gdpr:
  enabled: true
  default_value: 0
  timeouts_ms:
    active_vendorlist_fetch: 900000
hooks:
  enabled: true
  modules:
    scientiamobile:
      wurfl_devicedetection:
        enabled: true
        wurfl_file_path: "/path/to/wurfl.zip"
        wurfl_snapshot_url: "<wurfl_snapshot_url>"
        wurfl_cache_size: 200000
        allowed_publisher_ids:
          - "1"
          - "3"
        ext_caps: true
  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          entrypoint:
            groups:
              - timeout: 10
                hook_sequence:
                  - module_code: "scientiamobile.wurfl_devicedetection"
                    hook_impl_code: "scientiamobile-wurfl_devicedetection-entrypoint-hook"
          raw_auction_request:
            groups:
              - timeout: 10
                hook_sequence:
                  - module_code: "scientiamobile.wurfl_devicedetection"
                    hook_impl_code: "scientiamobile-wurfl_devicedetection-raw-auction-request-hook"
```

### Configuration Options

| Parameter                 | Requirement | Description                                                                                           |
|---------------------------|-------------|-------------------------------------------------------------------------------------------------------|
| **`wurfl_file_path`** | Mandatory   | Path to the WURFL file (e.g. /path/to/wurfl.zip). |
| **`wurfl_snapshot_url`**  | Optional    | URL of the licensed WURFL snapshot. If set, it periodically updates the WURFL file in the `wurfl_file_path` directory, which must be writable. |
| **`wurfl_cache_size`**    | Optional    | Maximum number of devices stored in the WURFL cache. Defaults to the WURFL cache's standard size.    |
| **`ext_caps`**            | Optional    | If `true`, the module adds all licensed capabilities to the `device.ext` object.                     |
| **`allowed_publisher_ids`** | Optional  | List of publisher IDs permitted to use the module. Defaults to all publishers.                       |

A valid WURFL license must include all the required capabilities for device enrichment.

### Launching Prebid Server with the WURFL Module

1. Download dependencies

```bash
go mod download
```

2. Copy the sample [config file](modules/scientiamobile/wurfl_devicedetection/sample/pbs-example.json)

```bash
cp modules/scientiamobile/wurfl_devicedetection/sample/pbs-example.json pbs.json
```

3. Copy the WURFL file to `wurfl_file_path`

In order for the WURFL module to work, you need a WURFL file installed on your system. The WURFL Infuze package comes with a recent evaluation copy of the WURFL file called wurfl.zip.

Commercial licensees of WURFL are granted access to "The Customer Vault", a personal virtual space containing purchased software licenses and weekly updated versions of the WURFL repository.

For more details, visit: [ScientiaMobile WURFL Infuze](https://docs.scientiamobile.com/documentation/infuze/infuze-c-api-user-guide).

4. Start the server

  ```bash
  go run -tags wurfl .
```

When the server starts, it loads the WURFL file from `wurfl_file_path` into the module.
To enable automatic updates, ensure that `wurfl_snapshot_url` is correctly configured.

Sample request data for testing is available in the module's `sample` directory.
Using the `auction` endpoint, you can observe WURFL-enriched device data in the response.

#### Start in demo mode

To test the WURFL module without an Infuze license:

```bash
go run .
```

### Sample Response

Using the sample request data via `curl` when the module is configured with `ext_caps` set to `false` (or no value)

```bash
curl http://localhost:8000/openrtb2/auction --data @modules/scientiamobile/wurfl_devicedetection/sample/request_data.json
```

the device object in the response will include WURFL device detection data:

```json
"device": {
  "ua": "Mozilla/5.0 (Linux; Android 15; Pixel 9 Pro XL Build/AP3A.241005.015;) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36 EdgA/124.0.2478.64",
  "devicetype": 1,
  "make": "Google",
  "model": "Pixel 9 Pro XL",
  "os": "Android",
  "osv": "15",
  "h": 2992,
  "w": 1344,
  "ppi": 481,
  "pxratio": 2.55,
  "js": 1,
  "ext": {
    "wurfl": {
      "wurfl_id": "google_pixel_9_pro_xl_ver1_suban150"
    }
  }
}
```

When `ext_caps` is set to `true`, the response will include all licensed capabilities:

```json
"device":{
  "ua":"Mozilla/5.0 (Linux; Android 15; Pixel 9 Pro XL Build/AP3A.241005.015; ) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36 EdgA/124.0.2478.64",
  "devicetype":1,
  "make":"Google",
  "model":"Pixel 9 Pro XL",
  "os":"Android",
  "osv":"15",
  "h":2992,
  "w":1344,
  "ppi":481,
  "pxratio":2.55,
  "js":1,
  "ext":{
    "wurfl":{
      "wurfl_id":"google_pixel_9_pro_xl_ver1_suban150",
      "mobile_browser_version":"",
      "resolution_height":"2992",
      "resolution_width":"1344",
      "is_wireless_device":"true",
      "is_tablet":"false",
      "physical_form_factor":"phone_phablet",
      "ajax_support_javascript":"true",
      "preferred_markup":"html_web_4_0",
      "brand_name":"Google",
      "can_assign_phone_number":"true",
      "xhtml_support_level":"4",
      "ux_full_desktop":"false",
      "device_os":"Android",
      "physical_screen_width":"71",
      "is_connected_tv":"false",
      "is_smarttv":"false",
      "physical_screen_height":"158",
      "model_name":"Pixel 9 Pro XL",
      "is_ott":"false",
      "density_class":"2.55",
      "marketing_name":"",
      "device_os_version":"15.0",
      "mobile_browser":"Chrome Mobile",
      "pointing_method":"touchscreen",
      "is_app_webview":"false",
      "advertised_app_name":"Edge Browser",
      "is_smartphone":"true",
      "is_robot":"false",
      "advertised_device_os":"Android",
      "is_largescreen":"true",
      "is_android":"true",
      "is_xhtmlmp_preferred":"false",
      "device_name":"Google Pixel 9 Pro XL",
      "is_ios":"false",
      "is_touchscreen":"true",
      "is_wml_preferred":"false",
      "is_app":"false",
      "is_mobile":"true",
      "is_phone":"true",
      "is_full_desktop":"false",
      "is_generic":"false",
      "advertised_browser":"Edge",
      "complete_device_name":"Google Pixel 9 Pro XL",
      "advertised_browser_version":"124.0.2478.64",
      "is_html_preferred":"true",
      "is_windows_phone":"false",
      "pixel_density":"481",
      "form_factor":"Smartphone",
      "advertised_device_os_version":"15"
    }
  }
}
```

### Building Docker images with WURFL support

When building Prebid Server with the WURFL module using Docker, you need to install both compile-time and runtime WURFL dependencies.

To build a Docker image with WURFL support:

1. **Place the WURFL library package**: Copy the `libwurfl*.deb` package files into the `modules/scientiamobile/wurfl_devicedetection/libwurfl/` directory before building the Docker image.

2. **Install compile-time dependencies**: In the build stage of your Dockerfile, before the `go build` command, add:

   ```dockerfile
   # Installing WURFL compile-time dependencies if libwurfl package is present
   RUN if ls modules/scientiamobile/wurfl_devicedetection/libwurfl/libwurfl*.deb 1> /dev/null 2>&1; then \
         dpkg -i modules/scientiamobile/wurfl_devicedetection/libwurfl/libwurfl*.deb; \
       fi
   ```

   And build with the WURFL tag:

   ```dockerfile
   RUN go build -tags wurfl -mod=vendor -ldflags "-X github.com/prebid/prebid-server/v3/version.Ver=`git describe --tags | sed 's/^v//'` -X github.com/prebid/prebid-server/v3/version.Rev=`git rev-parse HEAD`" .
   ```

3. **Install runtime dependencies**: In the runtime stage, add the WURFL runtime dependencies:

   ```dockerfile
   # Installing WURFL runtime dependencies if libwurfl package is present
   COPY modules/scientiamobile/wurfl_devicedetection/libwurfl/ /tmp/wurfl
   RUN if ls /tmp/wurfl/libwurfl*.deb 1> /dev/null 2>&1; then \
         dpkg -i /tmp/wurfl/libwurfl*.deb; \
         apt-get update && \
         apt-get install -y curl && \
         apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
         rm -rf /tmp/wurfl; \
       fi
   ```

## Maintainer

<prebid@scientiamobile.com>
