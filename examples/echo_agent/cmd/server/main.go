// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command server runs an A2A echo agent over SLIM RPC.
// It echoes back whatever text message it receives.
//
// Usage:
//
//	go run ./examples/echo_agent/cmd/server --slim-endpoint http://127.0.0.1:46357
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/agntcy/slim-a2a-go/a2aserver"
	slim_bindings "github.com/agntcy/slim-bindings-go"
)

func main() {
	endpoint := flag.String("slim-endpoint", "http://127.0.0.1:46357", "SLIM node endpoint")
	flag.Parse()

	slog.Info("starting echo agent server", "endpoint", *endpoint)

	if err := run(*endpoint); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func run(endpoint string) error {
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

	// Wrap it in the SLIM server adapter and register with the SLIM server.
	slimHandler := a2aserver.NewHandler(requestHandler)
	server := slim_bindings.ServerNewWithConnection(app, name, &connID)
	slimHandler.RegisterWith(server)

	slog.Info("echo agent ready", "slim_name", "agntcy/demo/echo_agent")
	return server.Serve()
}

// echoExecutor implements a2asrv.AgentExecutor.
// It echoes the first text part of the incoming message back as a task artifact.
type echoExecutor struct{}

var _ a2asrv.AgentExecutor = (*echoExecutor)(nil)

func (e *echoExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	// Emit submitted state for new tasks.
	if reqCtx.StoredTask == nil {
		submitted := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateSubmitted, nil)
		if err := queue.Write(ctx, submitted); err != nil {
			return fmt.Errorf("write submitted: %w", err)
		}
	}

	working := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)
	if err := queue.Write(ctx, working); err != nil {
		return fmt.Errorf("write working: %w", err)
	}

	// Extract text from the first part of the message.
	text := extractText(reqCtx.Message)

	// Echo it back as an artifact.
	artifact := a2a.NewArtifactEvent(reqCtx, a2a.TextPart{Text: text})
	if err := queue.Write(ctx, artifact); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}

	// Mark the task as complete.
	completed := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCompleted, nil)
	completed.Final = true
	if err := queue.Write(ctx, completed); err != nil {
		return fmt.Errorf("write completed: %w", err)
	}

	return nil
}

func (e *echoExecutor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	cancelled := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, nil)
	cancelled.Final = true
	return queue.Write(ctx, cancelled)
}

// extractText returns the text of the first TextPart in msg, or an empty string.
func extractText(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	for _, part := range msg.Parts {
		if tp, ok := part.(a2a.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}
