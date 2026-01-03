# Project Setup

## CLI

### cobra-cli

This CLI uses [cobra-cli](https://github.com/spf13/cobra) to create new commands.

To add a new command, run:

```sh
cobra-cli add <subcommand-name>
```

### mockgen

This CLI uses [mockgen](https://github.com/golang/mock) to generate mock interfaces.


To generate the mock for the interfaces, run:

```sh
mockgen -source=./pkg/cli/cli.go -destination=./pkg/cli/mock/mock_cli.go -package=mock
```

## Testing

### gotestsum

This CLI uses [gotestsum](https://github.com/gotestyourself/gotestsum) to run tests.

To run tests, run:

```sh
gotestsum --watch
```

### socat

This CLI uses [socat](http://www.dest-unreach.org/socat/) to proxy connections and debug.

## Sway WM integration

###  swayipc

This CLI uses [swayipc](https://github.com/nwg-piotr/swayipc) to integrate Sway with IPC.
Today it is hosted on [codeberg](https://codeberg.org/scip/swayipc).
`go get codeberg.org/scip/swayipc/v2`

