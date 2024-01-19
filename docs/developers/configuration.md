# Configuration

Prebid Server is configured using environment variables, a `pbs.json` file, or a `pbs.yaml` file, in that order of precedence. Configuration files are read from either the application directory or `/etc/config`. 

Upon starting, Prebid Server logs the resolved configuration to standard out with passwords and secrets redacted. If there's an error with the configuration, the application will log the error and exit.

# Sections
> [!IMPORTANT]
> As we are still developing this guide, please refer to the [configuration structures in code](../../config/config.go) for a complete definition of the options.

- [General](#general)
- [Privacy](#privacy)
  - [GDPR](#gdpr)


# General

### `external_url`
String value that specifies the external url to reach your Prebid Server instance. It's used for event tracking and user sync callbacks, and is shared with bidders in outgoing requests at `req.ext.prebid.server.externalurl`. Defaults to empty.

<details>
  <summary>Example</summary>
  <p>

  JSON:
  ```
  {
    "external_url": "https://your-pbs-server.com"
  }
  ```

  YAML:
  ```
  external_url: https://your-pbs-server.com
  ```

  Environment Variable:
  ```
  PBS_EXTERNAL_URL: https://your-pbs-server.com
  ```

  </p>
</details>

### `host`
String value that specifies the address the server will listen to for connections.  If the value is empty, Prebid Server will listen on all available addresses, which is a common configuration. This value is also used for the Prometheus endpoint, if enabled. Defaults to empty.

<details>
  <summary>Example</summary>
  <p>

  JSON:
  ```
  {
    "host": "127.0.0.1"
  }
  ```

  YAML:
  ```
  host: 127.0.0.1
  ```

  Environment Variable:
  ```
  PBS_HOST: 127.0.0.1
  ```

  </p>
</details>

### `port`
Integer value that specifies the port the server will listen to for connections. Defaults to `8000`.

<details>
  <summary>Example</summary>
  <p>

  JSON:
  ```
  {
    "port": 8000
  }
  ```

  YAML:
  ```
  port: 8000
  ```

  Environment Variable:
  ```
  PBS_PORT: 8000
  ```

  </p>
</details>

# Privacy

## GDPR

### `gdpr.enabled`
Boolean value that determines if GDPR processing for TCF signals is enabled. Defaults to `true`.
<details>
  <summary>Example</summary>
  <p>

  JSON:
  ```
  {
    "gdpr": {
      "enabled": true
    }
  }
  ```

  YAML:
  ```
  gdpr:
    enabled: true
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

  JSON:
  ```
  {
    "gdpr": {
      "default_value": "0"
    }
  }
  ```

  YAML:
  ```
  gdpr:
    default_value: "0"
  ```

  Environment Variable:
  ```
  PBS_GDPR_DEFAULT_VALUE: 0
  ```

  </p>
</details>
