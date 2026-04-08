// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package a2aslimrpc provides a SLIM RPC Handler and Transport for the A2A
// protocol v0.3.x wire format.
//
// Handler implements the SLIM RPC server side: it receives protobuf-encoded A2A
// requests from SLIM, deserialises them using the v0.3.x proto schema, and
// delegates to an [github.com/a2aproject/a2a-go/v2/a2asrv.RequestHandler].
//
// Transport implements [github.com/a2aproject/a2a-go/v2/a2aclient.Transport] so
// that A2A clients can send requests over SLIM RPC instead of HTTP or gRPC,
// using the same v0.3.x wire format.
//
// Both types rely on the generated stubs in
// [github.com/agntcy/slim-a2a-go/a2apb/v0] and are structurally identical to
// their v1 counterparts in [github.com/agntcy/slim-a2a-go/a2aslimrpc/v1] — only
// the proto schema and RPC service name differ on the wire.
//
// For agents speaking the A2A v1.0+ spec, use [github.com/agntcy/slim-a2a-go/a2aslimrpc/v1]
// instead.
package a2aslimrpc
