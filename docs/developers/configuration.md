# Configuration

Prebid Server is built using [Viper](https://github.com/spf13/viper) for configuration. Viper supports JSON, TOML, YAML, HCL, INI, envfile or Java properties formats. YAML, JSON, and Environment Variables are the most popular formats and are used as examples in this guide. 

Configuration is logged to standard out as Prebid Server starts up. If a validation error is detected, the application will immediately exit and report the problem.

For development, it's easiest to define your config inside a `pbs.yaml` file in the project root. This file is marked to be ignored by `.gitignore` and will not be automatically included in commits.

# We're Working On It

As we build this guide, please refer to [the contract classes](../../config/config.go) in code for a complete defintion of the configuration options.

# Privacy

## GDPR

### Default Value
String value that determines whether GDPR is enabled when no regulatory signal is available in the request. A value of `"0"` disables it by default and a value of `"1"` enabled it.
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
