# slim-a2a-go

[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/agntcy/slim-a2a-go/badge)](https://scorecard.dev/viewer/?uri=github.com/agntcy/slim-a2a-go)

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
go run github.com/agntcy/slim-bindings-go/v2/cmd/slim-bindings-setup
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
    slim_bindings "github.com/agntcy/slim-bindings-go/v2"
    slim_rpc "github.com/agntcy/slim-bindings-go/v2/slim_rpc"
    "github.com/a2aproject/a2a-go/v2/a2asrv"
)

slim_bindings.InitializeWithDefaults()
svc := slim_bindings.GetGlobalService()

name := slim_bindings.NewName("org", "project", "my_agent")
// Shared secret and plain-text gRPC: local testing only. See
// "Identity and secure node connections" below for JWT, SPIRE and TLS.
app, _ := svc.CreateAppWithSecret(name, secret)
connID, _ := svc.Connect(slim_bindings.NewInsecureClientConfig(slimNodeURL))
app.Subscribe(name, &connID)

server := slim_rpc.ServerNewWithConnection(app, name, &connID)
a2aslimrpc.NewHandler(a2asrv.NewHandler(myExecutor)).RegisterWith(server)
server.ServeBlocking()
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
channel := slim_rpc.ChannelNewWithConnection(app, remoteName, &connID)
transport := a2aslimrpc.NewTransport(channel)
defer transport.Destroy()
result, _ := transport.SendMessage(ctx, nil, req)
```

## Identity and secure node connections

`slim-a2a-go` does not create the SLIM app or the node connection: the caller
passes them in (`slim_rpc.ServerNewWithConnection(app, name, &connID)` on the
server, `WithSLIMRPCTransport(app, &connID)` on the client). So the app identity
and the node connection are whatever the caller configures.

The quickstarts use `CreateAppWithSecret` and `NewInsecureClientConfig`, which
are for a local node only. With a shared secret, anyone who holds the secret
can claim **any** SLIM name and join any MLS session. Every member of a session
must use a compatible identity type, so pick one for the whole mesh.

### JWT identity (EdDSA / `did:key`)

Each agent signs its own JWTs with an Ed25519 private key (PKCS#8 PEM) and
verifies peers against a JWKS file. The JWKS acts as the allowlist: only peers
whose public key is in it are accepted. For a `did:key` mesh set the subject to
the agent's `did:key` and give each JWKS entry `kid` = that `did:key`.

```go
audience := []string{"agntcy-a2a"}
did := "did:key:z6Mk..." // this agent's did:key

provider := slim_bindings.IdentityProviderConfigJwt{Config: slim_bindings.ClientJwtAuth{
    Key: slim_bindings.JwtKeyTypeEncoding{Key: slim_bindings.JwtKeyConfig{
        Algorithm: slim_bindings.JwtAlgorithmEdDsa,
        Format:    slim_bindings.JwtKeyFormatPem,
        Key:       slim_bindings.JwtKeyDataFile{Path: "/etc/agent/ed25519.pem"},
    }},
    Subject:  &did,
    Issuer:   &did,
    Audience: &audience,
    Duration: time.Hour,
}}
verifier := slim_bindings.IdentityVerifierConfigJwt{Config: slim_bindings.JwtAuth{
    Key: slim_bindings.JwtKeyTypeDecoding{Key: slim_bindings.JwtKeyConfig{
        Algorithm: slim_bindings.JwtAlgorithmEdDsa,
        Format:    slim_bindings.JwtKeyFormatJwks, // or JwtKeyFormatJwk / JwtKeyFormatPem
        Key:       slim_bindings.JwtKeyDataFile{Path: "/etc/agent/trusted-peers.jwks"},
    }},
    Audience: &audience,
    Duration: time.Hour,
}}
app, err := svc.CreateApp(name, provider, verifier)
```

SLIM verifiers reject tokens without an `iss` claim, so always set `Issuer` on
the provider; a self-issued token (`iss` = `sub` = `did:key`) is fine. Leave
`Issuer` unset on the verifier to accept peers with different issuers, or set
it to pin one. A pre-issued token can be used instead of a signing key with
`IdentityProviderConfigStaticJwt{Config: slim_bindings.StaticJwtAuth{TokenFile: path}}`.

### SPIRE identity

With a SPIRE agent on the host, the app gets its identity from the SPIFFE
Workload API and verifies peers against the trust bundle:

```go
spire := slim_bindings.SpireConfig{
    SocketPath:   nil, // nil uses SPIFFE_ENDPOINT_SOCKET
    JwtAudiences: []string{"agntcy-a2a"},
    TrustDomains: []string{},
}
app, err := svc.CreateApp(name,
    slim_bindings.IdentityProviderConfigSpire{Config: spire},
    slim_bindings.IdentityVerifierConfigSpire{Config: spire})
```

### TLS and bearer/JWT auth to the node

Start from `NewSecureClientConfig` (TLS on, system CA roots) and add a CA
bundle, a client certificate for mTLS, or node authentication as needed. A
public node that requires TLS plus a bearer token looks like this:

```go
cfg := slim_bindings.NewSecureClientConfig("https://slim.example.com:443")
cfg.Tls.CaSource = slim_bindings.CaSourceFile{Path: "/etc/agent/ca.pem"} // optional
// cfg.Tls.Source = slim_bindings.TlsSourceFile{Cert: "client.pem", Key: "client.key"} // mTLS

// Bearer token read from a file and re-read when it changes.
var auth slim_bindings.ClientAuthenticationConfig = slim_bindings.ClientAuthenticationConfigStaticJwt{
    Config: slim_bindings.StaticJwtAuth{TokenFile: "/run/agent/node-token", Duration: time.Hour},
}
// Or sign a JWT per connection: slim_bindings.ClientAuthenticationConfigJwt{Config: clientJwtAuth}.
cfg.Auth = &auth

connID, err := svc.Connect(cfg)
```

The echo agent example exposes all of this as flags (`--identity`, `--jwt-*`,
`--spire-*`, `--tls*`, `--node-auth`); see
[`examples/echo_agent`](examples/echo_agent/README.md#identity-and-node-connection).

### Caller identity in the handler

The handler does not yet tell the A2A executor which SLIM identity sent a
request. In slim-bindings-go v2.1.0 the server-side `slim_rpc.Context` exposes
only the deadline, the session ID and the request metadata, and the generated
stubs build the handler's `context.Context` from those. Metadata is set by the
sender, so do not use it as an identity. Surfacing the verified caller
(`peer_subject` / `peer_issuer`) needs SLIM 3.3 bindings and will follow the
slim-bindings-go upgrade.

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
