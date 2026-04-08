# slim-a2a-go

SLIM RPC transport adapter for the [A2A protocol](https://a2a-protocol.org) in Go.

`slim-a2a-go` lets agents built with [a2a-go](https://github.com/a2aproject/a2a-go) communicate
over [SLIM](https://github.com/agntcy/slim) instead of HTTP or gRPC. It provides two components:

- **`Handler`** — a SLIM RPC server adapter (analogous to `a2agrpc.Handler`) that receives A2A
  requests arriving over SLIM and delegates them to an `a2asrv.RequestHandler`.
- **`Transport`** — an `a2aclient.Transport` implementation that sends A2A requests over SLIM
  RPC instead of HTTP/JSON-RPC.

Both components are version-agnostic from the application's perspective: the same
`a2asrv.RequestHandler` / `a2aclient.Transport` interfaces are used regardless of which
wire format is in use.

## Dual-version support

| Package | A2A spec | Wire format |
|---|---|---|
| `github.com/agntcy/slim-a2a-go/a2aslimrpc/v0` | ≤ 0.3.x | `a2a-go/a2apb` (v0.3 proto) |
| `github.com/agntcy/slim-a2a-go/a2aslimrpc/v1` | 1.0+ | `a2a-go/v2/a2apb/v1` (v1 proto) |

A server can register both handlers simultaneously so that v0 and v1 clients can connect to the
same agent (see `--a2a-version both` in the echo agent example).

## Prerequisites

**slim-bindings-go setup (one-time)** — downloads the pre-compiled Rust library:

```bash
go run github.com/agntcy/slim-bindings-go/cmd/slim-bindings-setup
```

**SLIM node** — download `slimctl` from the
[releases page](https://github.com/agntcy/slim/releases) and start a local node:

```bash
slimctl slim start --endpoint 127.0.0.1:46357
```

## Server quickstart

```go
import (
    a2aslimrpc "github.com/agntcy/slim-a2a-go/a2aslimrpc/v1"
    slim_bindings "github.com/agntcy/slim-bindings-go"
    "github.com/a2aproject/a2a-go/v2/a2asrv"
)

slim_bindings.InitializeWithDefaults()
svc := slim_bindings.GetGlobalService()

name := slim_bindings.NewName("org", "project", "my_agent")
app, _ := svc.CreateAppWithSecret(name, secret)
connID, _ := svc.Connect(slim_bindings.NewInsecureClientConfig(slimNodeURL))
app.Subscribe(name, &connID)

server := slim_bindings.ServerNewWithConnection(app, name, &connID)
a2aslimrpc.NewHandler(a2asrv.NewHandler(myExecutor)).RegisterWith(server)
server.Serve()
```

## Client quickstart

Using the `a2aclient` factory with `WithSLIMRPCTransport`:

```go
import (
    a2aslimrpc "github.com/agntcy/slim-a2a-go/a2aslimrpc/v1"
    a2aclient "github.com/a2aproject/a2a-go/v2/a2aclient"
)

factory := a2aclient.NewFactory(
    a2aclient.WithDefaultsDisabled(),
    a2aslimrpc.WithSLIMRPCTransport(app, &connID),
)

card := &a2a.AgentCard{
    SupportedInterfaces: []*a2a.AgentInterface{
        a2a.NewAgentInterface("org/project/my_agent", a2aslimrpc.SLIMProtocol),
    },
}

client, _ := factory.CreateFromCard(ctx, card)
defer client.Destroy()
result, _ := client.SendMessage(ctx, req)
```

Or construct a `Transport` directly without the factory:

```go
channel := slim_bindings.ChannelNewWithConnection(app, remoteName, &connID)
transport := a2aslimrpc.NewTransport(channel)
defer transport.Destroy()
result, _ := transport.SendMessage(ctx, nil, req)
```

## Packages

| Import path | Description |
|---|---|
| `github.com/agntcy/slim-a2a-go/a2aslimrpc/v0` | Handler and Transport for A2A spec ≤ 0.3.x |
| `github.com/agntcy/slim-a2a-go/a2aslimrpc/v1` | Handler and Transport for A2A spec 1.0+ |
| `github.com/agntcy/slim-a2a-go/a2apb/v0` | Generated protobuf types for A2A v0.3.x (SLIM wire format) |
| `github.com/agntcy/slim-a2a-go/a2apb/v1` | Generated protobuf types for A2A v1.0 (SLIM wire format) |

## Development

Tasks are run via [Task](https://taskfile.dev):

```bash
task lint          # golangci-lint ./...
task fmt           # goimports -w .
task test          # go test -v ./...
task generate      # regenerate protobuf stubs (buf generate)
```

## Example

See [`examples/echo_agent`](examples/echo_agent/README.md) for a complete working example
that demonstrates the server, direct client, and factory-client patterns.

## API reference

[![pkg.go.dev](https://pkg.go.dev/badge/github.com/agntcy/slim-a2a-go.svg)](https://pkg.go.dev/github.com/agntcy/slim-a2a-go)
