# Configuration

Prebid Server can be configured uaing environment variables and supports several configuration file formats. The most commonly used formats are `yaml` and `json`, which are used as examples in this guide. Other supported formats include `toml`, `hcl`, `tfvars`, `ini`, `properities` (Java), and `env`.

- describe hunt path and file name



Configuration is logged to standard out as Prebid Server starts up. If a validation error is detected, the application will immediately exit and report the problem.

# Contents
> [!IMPORTANT]
> As we are still developing this guide, please refer to the [configuration structures in code](../../config/config.go) for a complete definition of the options.

- [Privacy](#privacy)
  - [GDPR](#gdpr)

# Privacy

## GDPR

### `gdpr.enabled`
Boolean value that determines if GDPR processing for TCF signals is enabled. Defaults to `true`.
<details>
  <summary>Example</summary>
  <p>

  YAML:
  ```
  gdpr:
    enabled: true
  ```

  JSON:
  ```
  {
    "gdpr": {
      "enabled": true
    }
  }
  ```

  Environment Variable:
  ```
  PBS_GDPR_ENABLED: true
  ```

  </p>
</details>


### `gdpr.default_value` (required)
String value that determines whether GDPR is enabled when no regulatory signal is available in the request. A value of `"0"` disables it by default and a value of `"1"` enables it. This is a required configuration value with no default.
<details>
  <summary>Example</summary>
  <p>

  YAML:
  ```
  gdpr:
    default_value: "0"
  ```

  JSON:
  ```
  {
    "gdpr": {
      "default_value": "0"
    }
  }
  ```

  Environment Variable:
  ```
  PBS_GDPR_DEFAULT_VALUE: 0
  ```

  </p>
</details>
