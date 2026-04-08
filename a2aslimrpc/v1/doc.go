// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package a2aslimrpc provides a SLIM RPC Handler and Transport for the A2A
// protocol v1.0+ wire format.
//
// Handler implements the SLIM RPC server side: it receives protobuf-encoded A2A
// requests from SLIM, deserialises them using the v1 proto schema, and delegates
// to an [github.com/a2aproject/a2a-go/v2/a2asrv.RequestHandler].
//
// Transport implements [github.com/a2aproject/a2a-go/v2/a2aclient.Transport] so
// that A2A clients can send requests over SLIM RPC instead of HTTP or gRPC,
// using the A2A v1.0 wire format.
//
// Both types rely on the generated stubs in
// [github.com/agntcy/slim-a2a-go/a2apb/v1] and are structurally identical to
// their v0 counterparts in [github.com/agntcy/slim-a2a-go/a2aslimrpc/v0] — only
// the proto schema and RPC service name differ on the wire.
//
// For backward compatibility with agents speaking the A2A v0.3.x spec, use
// [github.com/agntcy/slim-a2a-go/a2aslimrpc/v0] instead.
package a2aslimrpc
