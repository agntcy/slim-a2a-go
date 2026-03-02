// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aclient

import (
	"context"
	"iter"

	a2a "github.com/a2aproject/a2a-go/a2a"
	a2agoClient "github.com/a2aproject/a2a-go/a2aclient"
	a2agopb "github.com/a2aproject/a2a-go/a2apb"
	"github.com/a2aproject/a2a-go/a2apb/pbconv"
	slim_bindings "github.com/agntcy/slim-bindings-go"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb"
)

// SLIMProtocol is the transport protocol identifier for SLIM.
const SLIMProtocol a2a.TransportProtocol = "slim"

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

func (t *Transport) SendMessage(
	ctx context.Context, params *a2a.MessageSendParams,
) (a2a.SendMessageResult, error) {
	req, err := pbconv.ToProtoSendMessageRequest(params)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.SendMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoSendMessageResponse(resp)
}

func (t *Transport) SendStreamingMessage(
	ctx context.Context, params *a2a.MessageSendParams,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		req, err := pbconv.ToProtoSendMessageRequest(params)
		if err != nil {
			yield(nil, err)
			return
		}
		stream, err := t.client.SendStreamingMessage(ctx, req)
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
	ctx context.Context, query *a2a.TaskQueryParams,
) (*a2a.Task, error) {
	req, err := pbconv.ToProtoGetTaskRequest(query)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.GetTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTask(resp)
}

func (t *Transport) ListTasks(
	ctx context.Context, listReq *a2a.ListTasksRequest,
) (*a2a.ListTasksResponse, error) {
	req, err := pbconv.ToProtoListTasksRequest(listReq)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.ListTasks(ctx, req)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoListTasksResponse(resp)
}

func (t *Transport) CancelTask(
	ctx context.Context, id *a2a.TaskIDParams,
) (*a2a.Task, error) {
	req, err := pbconv.ToProtoCancelTaskRequest(id)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.CancelTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTask(resp)
}

func (t *Transport) ResubscribeToTask(
	ctx context.Context, id *a2a.TaskIDParams,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		req, err := pbconv.ToProtoTaskSubscriptionRequest(id)
		if err != nil {
			yield(nil, err)
			return
		}
		stream, err := t.client.TaskSubscription(ctx, req)
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
	ctx context.Context, params *a2a.GetTaskPushConfigParams,
) (*a2a.TaskPushConfig, error) {
	req, err := pbconv.ToProtoGetTaskPushConfigRequest(params)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.GetTaskPushNotificationConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTaskPushConfig(resp)
}

func (t *Transport) ListTaskPushConfig(
	ctx context.Context, params *a2a.ListTaskPushConfigParams,
) ([]*a2a.TaskPushConfig, error) {
	req, err := pbconv.ToProtoListTaskPushConfigRequest(params)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.ListTaskPushNotificationConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoListTaskPushConfig(resp)
}

func (t *Transport) SetTaskPushConfig(
	ctx context.Context, params *a2a.TaskPushConfig,
) (*a2a.TaskPushConfig, error) {
	req, err := pbconv.ToProtoCreateTaskPushConfigRequest(params)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.CreateTaskPushNotificationConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoTaskPushConfig(resp)
}

func (t *Transport) DeleteTaskPushConfig(
	ctx context.Context, params *a2a.DeleteTaskPushConfigParams,
) error {
	req, err := pbconv.ToProtoDeleteTaskPushConfigRequest(params)
	if err != nil {
		return err
	}
	_, err = t.client.DeleteTaskPushNotificationConfig(ctx, req)
	return err
}

func (t *Transport) GetAgentCard(ctx context.Context) (*a2a.AgentCard, error) {
	resp, err := t.client.GetAgentCard(ctx, &a2agopb.GetAgentCardRequest{})
	if err != nil {
		return nil, err
	}
	return pbconv.FromProtoAgentCard(resp)
}

func (t *Transport) Destroy() error {
	t.channel.Destroy()
	return nil
}
