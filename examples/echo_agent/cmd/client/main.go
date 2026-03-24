// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command client sends a text message to the echo agent over SLIM RPC and
// prints the response.
//
// Usage:
//
//	go run ./examples/echo_agent/cmd/client --text "hello world"
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/a2aproject/a2a-go/v2/a2a"
	slima2aclient "github.com/agntcy/slim-a2a-go/a2aclient"
	slim_bindings "github.com/agntcy/slim-bindings-go"
)

func main() {
	endpoint := flag.String("slim-endpoint", "http://127.0.0.1:46357", "SLIM node endpoint")
	text := flag.String("text", "hello", "Text message to send to the echo agent")
	flag.Parse()

	if err := run(*endpoint, *text); err != nil {
		slog.Error("client error", "err", err)
		os.Exit(1)
	}
}

func run(endpoint, text string) error {
	// Initialise the SLIM runtime.
	slim_bindings.InitializeWithDefaults()
	svc := slim_bindings.GetGlobalService()

	// Create the client's own SLIM identity.
	localName := slim_bindings.NewName("agntcy", "demo", "client")
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

	// Open a channel to the echo agent.
	remoteName := slim_bindings.NewName("agntcy", "demo", "echo_agent")
	channel := slim_bindings.ChannelNewWithConnection(app, remoteName, &connID)
	defer channel.Destroy()

	// Create the SLIM transport and send the message.
	transport := slima2aclient.NewTransport(channel)
	defer transport.Destroy() //nolint:errcheck

	req := &a2a.SendMessageRequest{
		Message: &a2a.Message{
			ID:   a2a.NewMessageID(),
			Role: a2a.MessageRoleUser,
			Parts: a2a.ContentParts{
				a2a.NewTextPart(text),
			},
		},
	}

	slog.Info("sending message", "text", text)
	result, err := transport.SendMessage(context.Background(), nil, req)
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
				if text, ok := part.Content.(a2a.Text); ok {
					return string(text)
				}
			}
		}
		return fmt.Sprintf("(task %s in state %s)", r.ID, r.Status.State)
	default:
		return fmt.Sprintf("(unexpected result type %T)", result)
	}
}

// extractText returns the concatenated text of all text parts in msg.
func extractText(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	out := ""
	for _, part := range msg.Parts {
		if text, ok := part.Content.(a2a.Text); ok {
			out += string(text)
		}
	}
	return out
}
