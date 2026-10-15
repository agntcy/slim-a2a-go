// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	"context"

	"github.com/agntcy/slim-a2a-go/internal/caller"
)

// Caller is the verified SLIM identity of the peer that sent a request to a
// [Handler]: the subject and issuer of the identity token SLIM verified for the
// session (JWT, SPIRE or shared secret). It is never read from request
// metadata, which the caller controls.
//
// In a group session the identity is the MLS sender of the request.
//
// The v0 and v1 packages share this type and context key, so a request handler
// served over both reads the caller the same way.
type Caller = caller.Caller

// CallerFromContext returns the verified identity of the peer that sent the
// request being handled. The [Handler] stores it in the context passed to the
// a2asrv.RequestHandler, so an AgentExecutor can call this on the context it
// receives. It returns false when SLIM did not report a peer identity.
func CallerFromContext(ctx context.Context) (Caller, bool) {
	return caller.FromContext(ctx)
}
