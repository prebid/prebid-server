# Configuration

Configuration is handled by [Viper](https://github.com/spf13/viper), which supports [many ways](https://github.com/spf13/viper#why-viper) to set config values.

As a general rule, Prebid Server will log its resolved config values on startup and exit immediately if they're not valid.

For development, it's easiest to define config inside a `pbs.yaml` file in the project root.

## Available options

For now, see [the contract classes](../../config/config.go) in the code.
