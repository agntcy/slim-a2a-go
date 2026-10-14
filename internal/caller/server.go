// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package caller

import (
	"context"
	"fmt"

	slim_rpc "github.com/agntcy/slim-bindings-go/v3/slim_rpc"
	"google.golang.org/protobuf/proto"
)

// The generated slimrpc stubs build the handler's context.Context with
// slim_rpc.ContextFromRpcContext, which keeps the deadline, session ID and
// metadata but drops the peer identity. Server and the adapters below replace
// the generated per-method handlers with equivalents that also store the
// verified Caller in the context and drop the SLIM session-layer metadata keys. Unmarshalling, error mapping and stream
// closing match the generated code.

// Server wraps a slim_rpc.ServerInterface for the generated
// Register<Service>Server function. Each unary-unary and unary-stream method
// it registers is served by the adapter of the same name in Unary or Stream
// instead of the generated handler, so the service and method names still come
// from the stubs. It panics if the stubs register a method with no adapter, so
// a regenerated service with new methods fails at registration, not silently.
type Server struct {
	slim_rpc.ServerInterface
	Unary  map[string]slim_rpc.UnaryUnaryHandler
	Stream map[string]slim_rpc.UnaryStreamHandler
}

// RegisterUnaryUnary registers the adapter for method in place of handler.
func (s Server) RegisterUnaryUnary(service, method string, _ slim_rpc.UnaryUnaryHandler) {
	h, ok := s.Unary[method]
	if !ok {
		panic(fmt.Sprintf("a2aslimrpc: no caller-aware handler for unary method %s/%s", service, method))
	}
	s.ServerInterface.RegisterUnaryUnary(service, method, h)
}

// RegisterUnaryStream registers the adapter for method in place of handler.
func (s Server) RegisterUnaryStream(service, method string, _ slim_rpc.UnaryStreamHandler) {
	h, ok := s.Stream[method]
	if !ok {
		panic(fmt.Sprintf("a2aslimrpc: no caller-aware handler for streaming method %s/%s", service, method))
	}
	s.ServerInterface.RegisterUnaryStream(service, method, h)
}

// sessionMetadataKeys are added to every message's metadata by the SLIM 3.4
// session layer: the verified sender ("slim-peer-subject", set after MLS
// decryption) and the payload signature. They are transport internals, not
// call metadata, so they are removed from the metadata the handler sees; the
// verified identity is available through the Caller instead.
var sessionMetadataKeys = []string{"slim-peer-subject", "slim-payload-sig"}

// handlerContext converts rc to a context.Context like the generated stubs do,
// removes the SLIM session-layer keys from its metadata, and adds the verified
// Caller when SLIM reports one.
func handlerContext(rc *slim_rpc.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := slim_rpc.ContextFromRpcContext(rc)
	if md, ok := slim_rpc.MetadataFromContext(ctx); ok {
		if filtered, changed := withoutSessionKeys(md); changed {
			ctx = slim_rpc.WithMetadata(ctx, filtered)
		}
	}
	if c, ok := FromRPCContext(rc); ok {
		ctx = NewContext(ctx, c)
	}
	return ctx, cancel
}

// withoutSessionKeys returns a copy of md without sessionMetadataKeys, and
// whether any were present. md itself is not modified.
func withoutSessionKeys(md map[string]string) (map[string]string, bool) {
	changed := false
	for _, k := range sessionMetadataKeys {
		if _, ok := md[k]; ok {
			changed = true
			break
		}
	}
	if !changed {
		return md, false
	}
	out := make(map[string]string, len(md))
	for k, v := range md {
		out[k] = v
	}
	for _, k := range sessionMetadataKeys {
		delete(out, k)
	}
	return out, true
}

func toRPCError(err error, code slim_rpc.RpcCode) *slim_rpc.RpcError {
	if rpcErr, ok := err.(*slim_rpc.RpcError); ok {
		return rpcErr
	}
	return slim_rpc.NewRpcErrorRpc(code, err.Error(), nil)
}

func newMessage[T proto.Message]() T {
	var zero T
	return zero.ProtoReflect().New().Interface().(T)
}

type unary[Req, Resp proto.Message] func(context.Context, Req) (Resp, error)

// Unary adapts a unary-unary method to slim_rpc.UnaryUnaryHandler.
func Unary[Req, Resp proto.Message](fn func(context.Context, Req) (Resp, error)) slim_rpc.UnaryUnaryHandler {
	return unary[Req, Resp](fn)
}

func (fn unary[Req, Resp]) Handle(request []byte, rc *slim_rpc.Context) ([]byte, error) {
	req := newMessage[Req]()
	if err := proto.Unmarshal(request, req); err != nil {
		return nil, toRPCError(err, slim_rpc.RpcCodeInvalidArgument)
	}
	ctx, cancel := handlerContext(rc)
	defer cancel()

	resp, err := fn(ctx, req)
	if err != nil {
		return nil, toRPCError(err, slim_rpc.RpcCodeInternal)
	}
	b, err := proto.Marshal(resp)
	if err != nil {
		return nil, toRPCError(err, slim_rpc.RpcCodeInternal)
	}
	return b, nil
}

type unaryStream[Req, Resp proto.Message] func(context.Context, Req, slim_rpc.ServerStream[Resp]) error

// UnaryStream adapts a unary-stream method to slim_rpc.UnaryStreamHandler.
func UnaryStream[Req, Resp proto.Message](
	fn func(context.Context, Req, slim_rpc.ServerStream[Resp]) error,
) slim_rpc.UnaryStreamHandler {
	return unaryStream[Req, Resp](fn)
}

func (fn unaryStream[Req, Resp]) Handle(request []byte, rc *slim_rpc.Context, sink *slim_rpc.ResponseSink) error {
	req := newMessage[Req]()
	if err := proto.Unmarshal(request, req); err != nil {
		rpcErr := toRPCError(err, slim_rpc.RpcCodeInvalidArgument)
		_ = sink.SendErrorAsync(rpcErr)
		return rpcErr
	}
	ctx, cancel := handlerContext(rc)
	defer cancel()

	err := fn(ctx, req, slim_rpc.NewServerRequestStream[Resp](sink))
	closeErr := sink.CloseAsync()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		rpcErr := toRPCError(err, slim_rpc.RpcCodeInternal)
		_ = sink.SendErrorAsync(rpcErr)
		return rpcErr
	}
	return nil
}
