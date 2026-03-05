// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command factory-client sends a text message to the echo agent over SLIM RPC
// using the a2a-go client factory and [slima2aclient.WithSLIMRPCTransport].
//
// Usage:
//
//	go run ./examples/echo_agent/cmd/factory-client --text "hello world"
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/a2aproject/a2a-go/a2a"
	a2aclient "github.com/a2aproject/a2a-go/a2aclient"
	slima2aclient "github.com/agntcy/slim-a2a-go/a2aclient"
	slim_bindings "github.com/agntcy/slim-bindings-go"
)

func main() {
	endpoint := flag.String("slim-endpoint", "http://127.0.0.1:46357", "SLIM node endpoint")
	agentName := flag.String("agent-name", "agntcy/demo/echo_agent", "SLIM name of the target agent")
	text := flag.String("text", "hello", "Text message to send to the echo agent")
	flag.Parse()

	if err := run(*endpoint, *agentName, *text); err != nil {
		slog.Error("client error", "err", err)
		os.Exit(1)
	}
}

func run(endpoint, agentName, text string) error {
	// Initialise the SLIM runtime.
	slim_bindings.InitializeWithDefaults()
	svc := slim_bindings.GetGlobalService()

	// Create the client's own SLIM identity.
	localName := slim_bindings.NewName("agntcy", "demo", "factory_client")
	app, err := svc.CreateAppWithSecret(localName, "my_shared_secret_for_testing_purposes_only")
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	// Connect to the SLIM node.
	connID, err := svc.Connect(slim_bindings.NewInsecureClientConfig(endpoint))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// Subscribe so this app can receive replies.
	if err := app.Subscribe(localName, &connID); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	// Build the a2a-go client factory with SLIM RPC as the sole transport.
	// WithDefaultsDisabled suppresses the built-in JSON-RPC and gRPC transports.
	factory := a2aclient.NewFactory(
		a2aclient.WithDefaultsDisabled(),
		slima2aclient.WithSLIMRPCTransport(app, &connID),
	)

	// Construct an agent card using the agent's SLIM name as the service URL.
	// WithSLIMRPCTransport calls slim_bindings.NameFromString on the URL to
	// derive the remote slim_bindings.Name when the channel is created.
	card := &a2a.AgentCard{
		URL:                agentName,
		PreferredTransport: slima2aclient.SLIMProtocol,
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
	}

	// Create the A2A client from the card.
	client, err := factory.CreateFromCard(context.Background(), card)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer client.Destroy() //nolint:errcheck

	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			ID:   a2a.NewMessageID(),
			Role: a2a.MessageRoleUser,
			Parts: []a2a.Part{
				a2a.TextPart{Text: text},
			},
		},
	}

	slog.Info("sending message", "text", text)
	result, err := client.SendMessage(context.Background(), params)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	fmt.Printf("> %s\n", text)
	fmt.Println(extractResult(result))
	return nil
}

// extractResult returns a human-readable string from a SendMessageResult.
func extractResult(result a2a.SendMessageResult) string {
	switch r := result.(type) {
	case *a2a.Message:
		return extractText(r)
	case *a2a.Task:
		for _, artifact := range r.Artifacts {
			for _, part := range artifact.Parts {
				if tp, ok := part.(a2a.TextPart); ok {
					return tp.Text
				}
			}
		}
		return fmt.Sprintf("(task %s in state %s)", r.ID, r.Status.State)
	default:
		return fmt.Sprintf("(unexpected result type %T)", result)
	}
}

// extractText returns the concatenated text of all TextParts in msg.
func extractText(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	out := ""
	for _, part := range msg.Parts {
		if tp, ok := part.(a2a.TextPart); ok {
			out += tp.Text
		}
	}
	return out
}
