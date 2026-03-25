// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command client sends a text message to the echo agent over SLIM RPC and
// prints the response.
//
// Usage:
//
//	go run ./examples/echo_agent/cmd/client --text "hello world"
//	go run ./examples/echo_agent/cmd/client --a2a-version v0   # v0.3.x only
//	go run ./examples/echo_agent/cmd/client --a2a-version v1   # v1.0 only (default)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2aslimrpcv0 "github.com/agntcy/slim-a2a-go/a2aslimrpc/v0"
	a2aslimrpcv1 "github.com/agntcy/slim-a2a-go/a2aslimrpc/v1"
	slim_bindings "github.com/agntcy/slim-bindings-go"
)

func main() {
	endpoint := flag.String("slim-endpoint", "http://127.0.0.1:46357", "SLIM node endpoint")
	text := flag.String("text", "hello", "Text message to send to the echo agent")
	version := flag.String("a2a-version", "v1", "A2A protocol version to use: v0 or v1")
	flag.Parse()

	if *version != "v0" && *version != "v1" {
		slog.Error("invalid --a2a-version; must be v0 or v1", "value", *version)
		os.Exit(1)
	}

	slog.Info("starting client", "endpoint", *endpoint, "a2a-version", *version)

	if err := run(*endpoint, *text, *version); err != nil {
		slog.Error("client error", "err", err)
		os.Exit(1)
	}
}

func run(endpoint, text, version string) error {
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

	var result a2a.SendMessageResult
	if version == "v0" {
		t := a2aslimrpcv0.NewTransport(channel)
		defer t.Destroy() //nolint:errcheck
		result, err = t.SendMessage(context.Background(), nil, req)
	} else {
		t := a2aslimrpcv1.NewTransport(channel)
		defer t.Destroy() //nolint:errcheck
		result, err = t.SendMessage(context.Background(), nil, req)
	}
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
