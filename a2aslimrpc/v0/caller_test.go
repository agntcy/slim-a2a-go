// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	"context"
	"iter"
	"strings"
	"sync"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	slim_bindings "github.com/agntcy/slim-bindings-go/v3"
	slim_rpc "github.com/agntcy/slim-bindings-go/v3/slim_rpc"
)

// callerExecutor records the Caller its Execute context carries, then
// completes the task.
type callerExecutor struct {
	mu       sync.Mutex
	caller   Caller
	found    bool
	metadata map[string]string
}

var _ a2asrv.AgentExecutor = (*callerExecutor)(nil)

func (e *callerExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	c, ok := CallerFromContext(ctx)
	md, _ := slim_rpc.MetadataFromContext(ctx)
	e.mu.Lock()
	e.caller, e.found, e.metadata = c, ok, md
	e.mu.Unlock()
	return func(yield func(a2a.Event, error) bool) {
		if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
			return
		}
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
	}
}

func (e *callerExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

// assertCaller checks the recorded Caller is the client app. A shared-secret
// identity's subject is the app's SLIM name followed by "_" and a random suffix.
func (e *callerExecutor) assertCaller(t *testing.T, clientApp *slim_bindings.App) {
	t.Helper()
	e.mu.Lock()
	got, ok, md := e.caller, e.found, e.metadata
	e.mu.Unlock()
	if !ok {
		t.Fatal("CallerFromContext: no caller in the AgentExecutor context")
	}
	if prefix := clientApp.Name().String() + "_"; !strings.HasPrefix(got.Subject, prefix) {
		t.Errorf("Caller.Subject = %q, want the client identity %q<suffix>", got.Subject, prefix)
	}
	if got.Issuer != "" {
		t.Errorf("Caller.Issuer = %q, want empty for a shared-secret identity", got.Issuer)
	}
	// SLIM 3.4 adds these to every message; they must not leak into the call
	// metadata, while the client's own keys still arrive.
	for _, k := range []string{"slim-peer-subject", "slim-payload-sig"} {
		if v, ok := md[k]; ok {
			t.Errorf("handler metadata has session-layer key %q = %q", k, v)
		}
	}
	if md["x-app-key"] != "app-value" {
		t.Errorf("handler metadata[x-app-key] = %q, want %q (metadata: %v)", md["x-app-key"], "app-value", md)
	}
}

// TestHandler_CallerIdentity checks, over a real in-process SLIM session, that
// the client's verified identity reaches the AgentExecutor through a2asrv for
// unary and streaming calls, that metadata sent by the client claiming
// another identity (including SLIM's own slim-peer-subject key) has no effect
// on it, and that SLIM session-layer keys are not passed on as call metadata.
func TestHandler_CallerIdentity(t *testing.T) {
	spoofed := map[string]string{
		"subject":           "spiffe://evil/admin",
		"caller":            "spiffe://evil/admin",
		"slim-peer-subject": "spiffe://evil/admin",
		"x-app-key":         "app-value",
	}

	newRequest := func(t *testing.T) (context.Context, *a2a.SendMessageRequest) {
		t.Helper()
		ctx := slim_rpc.WithMetadata(t.Context(), spoofed)
		return ctx, &a2a.SendMessageRequest{
			Message: a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("who am I?")),
		}
	}

	t.Run("unary", func(t *testing.T) {
		exec := &callerExecutor{}
		client, clientApp := startTestServerWithClientApp(t, a2asrv.NewHandler(exec))
		ctx, req := newRequest(t)
		pbReq, err := defaultConverter{}.ToProtoSendMessageRequest(req)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.SendMessage(ctx, pbReq); err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
		exec.assertCaller(t, clientApp)
	})

	t.Run("streaming", func(t *testing.T) {
		exec := &callerExecutor{}
		client, clientApp := startTestServerWithClientApp(t, a2asrv.NewHandler(exec))
		ctx, req := newRequest(t)
		pbReq, err := defaultConverter{}.ToProtoSendMessageRequest(req)
		if err != nil {
			t.Fatal(err)
		}
		stream, err := client.SendStreamingMessage(ctx, pbReq)
		if err != nil {
			t.Fatalf("SendStreamingMessage: %v", err)
		}
		for {
			ev, err := stream.Recv()
			if err != nil {
				t.Fatalf("Recv: %v", err)
			}
			if ev == nil {
				break
			}
		}
		exec.assertCaller(t, clientApp)
	})
}

func TestCallerFromContext_NoCaller(t *testing.T) {
	if c, ok := CallerFromContext(context.Background()); ok {
		t.Fatalf("CallerFromContext(context.Background()) = %+v, true; want false", c)
	}
}
