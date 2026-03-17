## hawkeye run

Run hawkeye

### Synopsis

Hawk-Eye will be started with the provided configuration

```
hawkeye run [flags]
```

### Options

```
      --apiAddress string               api: The address the server is listening on (default ":8080")
  -h, --help                            help for run
      --loaderFilePath string           file loader: The path to the file to read the runtime config from (default "config.yaml")
      --loaderHttpRetryCount int        http loader: Amount of retries trying to load the configuration (default 3)
      --loaderHttpRetryDelay duration   http loader: The initial delay between retries in seconds (default 1s)
      --loaderHttpTimeout duration      http loader: The timeout for the http request in seconds (default 30s)
      --loaderHttpToken string          http loader: Bearer token to authenticate the http endpoint
      --loaderHttpUrl string            http loader: The url where to get the remote configuration
      --loaderInterval duration         defines the interval the loader reloads the configuration in seconds (default 5m0s)
  -l, --loaderType string               Defines the loader type that will load the checks configuration during the runtime. The fallback is the fileLoader (default "http")
      --hawkeyeName string              The DNS name of the hawkeye
```

### Options inherited from parent commands

```
  -c, --config string   config file (default is $HOME/.hawkeye.yaml)
```

### SEE ALSO

* [hawkeye](hawkeye.md)	 - Hawk-Eye, the infrastructure monitoring agent

