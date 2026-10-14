# Echo Agent

A minimal A2A echo agent that returns whatever text it receives, communicating over SLIM RPC.
This is the Go equivalent of the `echo_agent` example from [slim-a2a-python](https://github.com/agntcy/slim-a2a-python).

## Prerequisites

### 1. Set up slim-bindings-go (one-time)

The SLIM transport uses a pre-compiled Rust library. Run the setup tool once to download it:

```bash
go run github.com/agntcy/slim-bindings-go/v2/cmd/slim-bindings-setup
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
- Identity and node-connection flags: see [Identity and node connection](#identity-and-node-connection)

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
- Identity and node-connection flags: see [Identity and node connection](#identity-and-node-connection)

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
- Identity and node-connection flags: see [Identity and node connection](#identity-and-node-connection)

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

## Identity and node connection

All three commands share the same flags for the SLIM app identity and the
connection to the SLIM node. The defaults (shared secret, plain-text gRPC to
`http://127.0.0.1:46357`) only suit a local node: anyone with the shared secret
can claim any SLIM name. All members of a session need a compatible identity
type, so start the server and the clients with the same `--identity`.

| Flag | Description |
|---|---|
| `--identity` | `shared-secret` (default, local testing only), `jwt` or `spire` |
| `--shared-secret` | Secret for `shared-secret` (default: `$SLIM_SHARED_SECRET`, else a built-in test value) |
| `--jwt-key` | PEM private key that signs this app's JWTs |
| `--jwt-algorithm` | `EdDSA` (default), `ES256`, `ES384`, `RS256`, `RS384`, `RS512`, `PS256`, `PS384`, `PS512` |
| `--jwt-subject` | `sub` claim, e.g. the agent's `did:key` |
| `--jwt-issuer` | `iss` claim (default: `--jwt-subject`; SLIM requires one) |
| `--jwt-audience` | Comma-separated audiences to set and verify |
| `--jwt-duration` | Validity of each signed token (default `1h`) |
| `--jwt-verify-key` | Public key(s) trusted to verify peers |
| `--jwt-verify-format` | `jwks` (default), `jwk` or `pem` |
| `--jwt-verify-issuer` | Only accept peers with this `iss` (default: any issuer) |
| `--spire-socket` | SPIFFE Workload API socket (default: `$SPIFFE_ENDPOINT_SOCKET`) |
| `--spire-target-spiffe-id`, `--spire-audience` | JWT-SVID target and audiences |
| `--slim-endpoint` | SLIM node address (default `http://127.0.0.1:46357`) |
| `--tls` | Use TLS (implied by an `https://` or `wss://` endpoint) |
| `--tls-ca` | PEM CA bundle for the node certificate (system roots are used too) |
| `--tls-cert`, `--tls-key` | Client certificate for mTLS |
| `--tls-insecure-skip-verify` | Skip node certificate verification (testing only) |
| `--node-auth` | `none` (default), `bearer` (with `--node-token-file`) or `jwt` (signs with the `--jwt-*` key) |
| `--node-token-file` | Bearer token file; re-read when it changes |

### JWT identity with `did:key`

Each agent has an Ed25519 key (PKCS#8 PEM) and a `did:key` derived from its
public key. `trusted.jwks` lists the public key of every agent allowed to talk
to this one, each with `kid` set to that agent's `did:key`:

```bash
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/server \
  --identity jwt \
  --jwt-key server-ed25519.pem \
  --jwt-subject did:key:z6MkServer... \
  --jwt-audience agntcy-a2a \
  --jwt-verify-key trusted.jwks

GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/client \
  --identity jwt \
  --jwt-key client-ed25519.pem \
  --jwt-subject did:key:z6MkClient... \
  --jwt-audience agntcy-a2a \
  --jwt-verify-key trusted.jwks \
  --text "hello over a JWT identity"
```

Each token is self-issued (`iss` = `sub`). A client whose key is not in the
server's `trusted.jwks`, or that uses a shared secret, fails the session
handshake.

### SPIRE identity

```bash
export SPIFFE_ENDPOINT_SOCKET=unix:///run/spire/agent/public/api.sock
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/server \
  --identity spire --spire-audience agntcy-a2a
```

### TLS node with a bearer token

```bash
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn go run ./examples/echo_agent/cmd/factory-client \
  --slim-endpoint https://slim.example.com:443 \
  --node-auth bearer --node-token-file ./node-token \
  --identity jwt --jwt-key client-ed25519.pem --jwt-subject did:key:z6MkClient... \
  --jwt-verify-key trusted.jwks \
  --text "hello over TLS"
```

Add `--tls-ca ca.pem` for a private CA, or `--tls-cert`/`--tls-key` for mTLS.
