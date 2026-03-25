# Echo Agent

A minimal A2A echo agent that returns whatever text it receives, communicating over SLIM RPC.
This is the Go equivalent of the `echo_agent` example from [slim-a2a-python](https://github.com/agntcy/slim-a2a-python).

## Prerequisites

### 1. Set up slim-bindings-go (one-time)

The SLIM transport uses a pre-compiled Rust library. Run the setup tool once to download it:

```bash
go run github.com/agntcy/slim-bindings-go/cmd/slim-bindings-setup
```

### 2. Start a SLIM node

Download `slimctl` from the [release page](https://github.com/agntcy/slim/releases) and start a local node:

```bash
slimctl slim start --endpoint 127.0.0.1:46357
```

## Run the echo agent server

```bash
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/server
```

Optional flags:
- `--slim-endpoint` — SLIM node address (default: `http://127.0.0.1:46357`)

## Run the echo agent client

```bash
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/client --text "hi, this is a text message"
```

Expected output:
```
> hi, this is a text message
hi, this is a text message
```

Optional flags:
- `--text` — message to send (default: `hello`)
- `--slim-endpoint` — SLIM node address (default: `http://127.0.0.1:46357`)
