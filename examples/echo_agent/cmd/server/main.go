// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command server runs an A2A echo agent over SLIM RPC.
// It echoes back whatever text message it receives.
//
// Usage:
//
//	go run ./examples/echo_agent/cmd/server --slim-endpoint http://127.0.0.1:46357
//	go run ./examples/echo_agent/cmd/server --a2a-version v0   # v0.3.x only
//	go run ./examples/echo_agent/cmd/server --a2a-version v1   # v1.0 only
//	go run ./examples/echo_agent/cmd/server --a2a-version both # both (default)
package main

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"log/slog"
	"os"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	a2aslimrpcv0 "github.com/agntcy/slim-a2a-go/a2aslimrpc/v0"
	a2aslimrpcv1 "github.com/agntcy/slim-a2a-go/a2aslimrpc/v1"
	slim_bindings "github.com/agntcy/slim-bindings-go"
)

func main() {
	endpoint := flag.String("slim-endpoint", "http://127.0.0.1:46357", "SLIM node endpoint")
	version := flag.String("a2a-version", "both", "A2A protocol version to serve: v0, v1, or both")
	flag.Parse()

	if *version != "v0" && *version != "v1" && *version != "both" {
		slog.Error("invalid --a2a-version; must be v0, v1, or both", "value", *version)
		os.Exit(1)
	}

	slog.Info("starting echo agent server", "endpoint", *endpoint, "a2a-version", *version)

	if err := run(*endpoint, *version); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func run(endpoint, version string) error {
	// Initialise the SLIM runtime.
	slim_bindings.InitializeWithDefaults()
	svc := slim_bindings.GetGlobalService()

	// Create the agent's SLIM identity.
	name := slim_bindings.NewName("agntcy", "demo", "echo_agent")
	app, err := svc.CreateAppWithSecret(name, "my_shared_secret_for_testing_purposes_only")
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	// Connect to the SLIM node.
	connID, err := svc.Connect(slim_bindings.NewInsecureClientConfig(endpoint))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// Subscribe so this app receives messages addressed to its name.
	if err := app.Subscribe(name, &connID); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	// Build the A2A request handler wrapping our echo executor.
	requestHandler := a2asrv.NewHandler(&echoExecutor{})

	// Register protocol handler(s) with the SLIM server.
	// v0 and v1 use different service names on the wire so both can coexist.
	server := slim_bindings.ServerNewWithConnection(app, name, &connID)
	switch version {
	case "v0":
		a2aslimrpcv0.NewHandler(requestHandler).RegisterWith(server)
	case "v1":
		a2aslimrpcv1.NewHandler(requestHandler).RegisterWith(server)
	default: // "both"
		a2aslimrpcv0.NewHandler(requestHandler).RegisterWith(server)
		a2aslimrpcv1.NewHandler(requestHandler).RegisterWith(server)
	}

	slog.Info("echo agent ready", "slim_name", "agntcy/demo/echo_agent")
	return server.Serve()
}

// echoExecutor implements a2asrv.AgentExecutor.
// It echoes the first text part of the incoming message back as a task artifact.
type echoExecutor struct{}

var _ a2asrv.AgentExecutor = (*echoExecutor)(nil)

func (e *echoExecutor) Execute(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		// Emit submitted state for new tasks.
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}

		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}

		// Extract text from the first part of the message and echo it back as an artifact.
		text := extractText(execCtx.Message)
		if !yield(a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(text)), nil) {
			return
		}

		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
	}
}

func (e *echoExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

// extractText returns the text of the first text part in msg, or an empty string.
func extractText(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	for _, part := range msg.Parts {
		if text, ok := part.Content.(a2a.Text); ok {
			return string(text)
		}
	}
	return ""
}
