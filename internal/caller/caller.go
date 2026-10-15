// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package caller carries the verified SLIM identity of an RPC caller in a
// context.Context. It is shared by the v0 and v1 a2aslimrpc packages so that
// both expose the same type and context key.
package caller

import (
	"context"

	slim_rpc "github.com/agntcy/slim-bindings-go/v3/slim_rpc"
)

// Caller is the verified identity of the peer that sent a SLIM RPC request.
//
// The values come from the identity token SLIM verified for the session, never
// from request metadata, so they cannot be set by the caller.
type Caller struct {
	// Subject is the verified subject (`sub` claim) of the peer: for example a
	// did:key, a SPIFFE ID, or the name bound to a shared secret.
	Subject string
	// Issuer is the verified issuer (`iss` claim), or "" when the peer's token
	// has none.
	Issuer string
}

type contextKey struct{}

// NewContext returns a copy of ctx that carries c.
func NewContext(ctx context.Context, c Caller) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}

// FromContext returns the Caller stored in ctx by NewContext, and whether one
// was present.
func FromContext(ctx context.Context) (Caller, bool) {
	c, ok := ctx.Value(contextKey{}).(Caller)
	return c, ok
}

// rpcContext is the subset of *slim_rpc.Context needed to read the peer
// identity; an interface so tests can supply a fake.
type rpcContext interface {
	PeerSubject() *string
	PeerIssuer() *string
}

var _ rpcContext = (*slim_rpc.Context)(nil)

// FromRPCContext reads the verified peer identity from a SLIM RPC context. It
// returns false when SLIM reports no subject for the peer.
func FromRPCContext(rc rpcContext) (Caller, bool) {
	sub := rc.PeerSubject()
	if sub == nil || *sub == "" {
		return Caller{}, false
	}
	c := Caller{Subject: *sub}
	if iss := rc.PeerIssuer(); iss != nil {
		c.Issuer = *iss
	}
	return c, true
}
