// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aclient

import (
	"context"
	"fmt"
	"iter"

	a2a "github.com/a2aproject/a2a-go/v2/a2a"
	a2agoClient "github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2apb/v1/pbconv"
	slim_bindings "github.com/agntcy/slim-bindings-go"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb"
)

// SLIMProtocol is the transport protocol identifier for SLIM RPC.
const SLIMProtocol a2a.TransportProtocol = "slimrpc"

// Transport implements a2aclient.Transport over SLIM RPC.
type Transport struct {
	client  ourpb.A2AServiceClient
	channel *slim_bindings.Channel
}

// Verify Transport implements a2aclient.Transport at compile time.
var _ a2agoClient.Transport = (*Transport)(nil)

// NewTransport creates a new SLIM transport wrapping the provided channel.
func NewTransport(channel *slim_bindings.Channel) *Transport {
	return &Transport{
		client:  ourpb.NewA2AServiceClient(channel),
		channel: channel,
	}
}

// WithSLIMRPCTransport returns an [a2aclient.FactoryOption] that registers the
// SLIM RPC transport with the client factory.
//
// app and connID are the pre-established SLIM app and connection — set these
// up with slim_bindings before calling [a2aclient.NewFactory]. The url
// supplied by the factory (iface.URL) is parsed into a [slim_bindings.Name]
// via [slim_bindings.NameFromString], so callers should use the SLIM agent
// name (e.g. "agntcy/demo/echo_agent") as the interface URL.
func WithSLIMRPCTransport(app *slim_bindings.App, connID *uint64) a2agoClient.FactoryOption {
	return a2agoClient.WithTransport(
		SLIMProtocol,
		a2agoClient.TransportFactoryFn(func(ctx context.Context, card *a2a.AgentCard, iface *a2a.AgentInterface) (a2agoClient.Transport, error) {
			remoteName, err := slim_bindings.NameFromString(iface.URL)
			if err != nil {
				return nil, fmt.Errorf("invalid SLIM agent name %q: %w", iface.URL, err)
			}
			channel := slim_bindings.ChannelNewWithConnection(app, remoteName, connID)
			return NewTransport(channel), nil
		}),
	)
}

func (t *Transport) SendMessage(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.SendMessageRequest,
) (a2a.SendMessageResult, error) {
	pbReq, err := pbconv.ToProtoSendMessageRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.SendMessage(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoSendMessageResponse(resp)
}

func (t *Transport) SendStreamingMessage(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.SendMessageRequest,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		pbReq, err := pbconv.ToProtoSendMessageRequest(req)
		if err != nil {
			yield(nil, err)
			return
		}
		stream, err := t.client.SendStreamingMessage(ctx, pbReq)
		if err != nil {
			yield(nil, err)
			return
		}
		for {
			resp, err := stream.Recv()
			if err != nil {
				yield(nil, err)
				return
			}
			if resp == nil {
				return
			}
			event, err := pbconv.FromProtoStreamResponse(resp)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(event, nil) {
				return
			}
		}
	}
}

func (t *Transport) GetTask(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.GetTaskRequest,
) (*a2a.Task, error) {
	pbReq, err := pbconv.ToProtoGetTaskRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.GetTask(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTask(resp)
}

func (t *Transport) ListTasks(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.ListTasksRequest,
) (*a2a.ListTasksResponse, error) {
	pbReq, err := pbconv.ToProtoListTasksRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.ListTasks(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoListTasksResponse(resp)
}

func (t *Transport) CancelTask(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.CancelTaskRequest,
) (*a2a.Task, error) {
	pbReq, err := pbconv.ToProtoCancelTaskRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.CancelTask(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTask(resp)
}

func (t *Transport) SubscribeToTask(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.SubscribeToTaskRequest,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		pbReq, err := pbconv.ToProtoSubscribeToTaskRequest(req)
		if err != nil {
			yield(nil, err)
			return
		}
		stream, err := t.client.SubscribeToTask(ctx, pbReq)
		if err != nil {
			yield(nil, err)
			return
		}
		for {
			resp, err := stream.Recv()
			if err != nil {
				yield(nil, err)
				return
			}
			if resp == nil {
				return
			}
			event, err := pbconv.FromProtoStreamResponse(resp)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(event, nil) {
				return
			}
		}
	}
}

func (t *Transport) GetTaskPushConfig(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.GetTaskPushConfigRequest,
) (*a2a.TaskPushConfig, error) {
	pbReq, err := pbconv.ToProtoGetTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.GetTaskPushNotificationConfig(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTaskPushConfig(resp)
}

func (t *Transport) ListTaskPushConfigs(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.ListTaskPushConfigRequest,
) ([]*a2a.TaskPushConfig, error) {
	pbReq, err := pbconv.ToProtoListTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.ListTaskPushNotificationConfigs(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	domainResp, err := pbconv.FromProtoListTaskPushConfigResponse(resp)
	if err != nil {
		return nil, err
	}
	return domainResp.Configs, nil
}

func (t *Transport) CreateTaskPushConfig(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.CreateTaskPushConfigRequest,
) (*a2a.TaskPushConfig, error) {
	pbReq, err := pbconv.ToProtoCreateTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.CreateTaskPushNotificationConfig(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTaskPushConfig(resp)
}

func (t *Transport) DeleteTaskPushConfig(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.DeleteTaskPushConfigRequest,
) error {
	pbReq, err := pbconv.ToProtoDeleteTaskPushConfigRequest(req)
	if err != nil {
		return err
	}
	_, err = t.client.DeleteTaskPushNotificationConfig(ctx, pbReq)
	return err
}

func (t *Transport) GetExtendedAgentCard(
	ctx context.Context, _ a2agoClient.ServiceParams, req *a2a.GetExtendedAgentCardRequest,
) (*a2a.AgentCard, error) {
	pbReq, err := pbconv.ToProtoGetExtendedAgentCardRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.GetExtendedAgentCard(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoAgentCard(resp)
}

func (t *Transport) Destroy() error {
	t.channel.Destroy()
	return nil
}
