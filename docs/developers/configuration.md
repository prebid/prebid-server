# Configuration

Configuration is handled by [Viper](https://github.com/spf13/viper), which supports [many ways](https://github.com/spf13/viper#why-viper) of setting config values.

As a general rule, Prebid Server will log its resolved config values on startup and exit immediately if they're not valid.

For development, it's easiest to define your config inside a `pbs.yaml` file in the project root.

## Available options

For now, see [the contract classes](../../config/config.go) in the code.

Also note that `Viper` will also read environment variables for config values. Prebid Server will look for the prefix `PBS_` on the environment variables, and map underscores (`_`)
to periods. For example, to set `host_cookie.ttl_days` via an environment variable, set `PBS_HOST_COOKIE_TTL_DAYS` to the desired value.
