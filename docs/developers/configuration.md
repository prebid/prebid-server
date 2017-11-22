# Configuration

Configuration is handled by [Viper](https://github.com/spf13/viper), which has [many ways to set config values](https://github.com/spf13/viper#why-viper).

As a general rule, Prebid Server will log its resolved config values on startup and exit immediately if they're not valid.

For development, it's easiest to define a `pbs.yaml` file in the project root.

## Available options

For now, see [the contract classes](../../config/config.go) in the code.
