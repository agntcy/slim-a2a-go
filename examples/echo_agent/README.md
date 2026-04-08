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
- `--a2a-version` — A2A protocol version(s) to serve: `v0`, `v1`, or `both` (default: `both`)

With `--a2a-version both` (the default) the server registers both the v0 and v1 RPC service names
simultaneously, so a v0 or v1 client can both connect to the same running agent.

## Run the direct client

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
- `--a2a-version` — A2A protocol version to use: `v0` or `v1` (default: `v1`)

The direct client constructs a `Transport` and `Channel` manually without using the
`a2aclient.Factory`.

## Run the factory client

```bash
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/factory-client --text "hi from factory"
```

Expected output:
```
> hi from factory
hi from factory
```

Optional flags:
- `--text` — message to send (default: `hello`)
- `--slim-endpoint` — SLIM node address (default: `http://127.0.0.1:46357`)
- `--agent-name` — SLIM name of the target agent (default: `agntcy/demo/echo_agent`)
- `--a2a-version` — A2A protocol version to use: `v0` or `v1` (default: `v1`)

The factory client demonstrates the `a2aclient.Factory` pattern: it calls
`WithSLIMRPCTransport` to register the transport, then uses `factory.CreateFromCard` with an
`AgentCard` to create the client. This is the recommended approach for production code because
it decouples transport selection from the rest of the application.

## Cross-version example

Start the server with both protocols enabled (the default), then run a v0 and a v1 client side
by side:

```bash
# Terminal 1 — server (serves both v0 and v1)
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/server

# Terminal 2 — v1 client (default)
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/client --text "hello from v1"

# Terminal 3 — v0 client
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/client --a2a-version v0 --text "hello from v0"
```
