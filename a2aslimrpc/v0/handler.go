// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	"context"

	a2agopb "github.com/a2aproject/a2a-go/a2apb"
	a2a "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	slim_bindings "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/slim-bindings-go/slimrpc"
	"google.golang.org/protobuf/types/known/emptypb"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb/v0"
)

// Handler implements ourpb.A2AServiceServer by delegating to an a2asrv.RequestHandler.
// It translates between the A2A v0.3.x wire format and the v2 domain types.
type Handler struct {
	ourpb.UnimplementedA2AServiceServer
	handler a2asrv.RequestHandler
	conv    converter
}

// NewHandler creates a new SLIM server handler wrapping the given RequestHandler.
func NewHandler(handler a2asrv.RequestHandler, opts ...HandlerOption) *Handler {
	h := &Handler{handler: handler, conv: defaultConverter{}}
	for _, o := range opts {
		o(h)
	}
	return h
}

// RegisterWith registers the A2A service with a SLIM server.
func (h *Handler) RegisterWith(s *slim_bindings.Server) {
	ourpb.RegisterA2AServiceServer(s, h)
}

// SendMessage implements [ourpb.A2AServiceServer] — receives a message and returns a non-streaming response.
func (h *Handler) SendMessage(
	ctx context.Context, req *a2agopb.SendMessageRequest,
) (*a2agopb.SendMessageResponse, error) {
	params, err := h.conv.FromProtoSendMessageRequest(req)
	if err != nil {
		return nil, err
	}

	result, err := h.handler.SendMessage(ctx, params)
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoSendMessageResponse(result)
}

// SendStreamingMessage implements [ourpb.A2AServiceServer] — sends a message and streams events back to the caller.
func (h *Handler) SendStreamingMessage(
	ctx context.Context,
	req *a2agopb.SendMessageRequest,
	stream slimrpc.RequestStream[*a2agopb.StreamResponse],
) error {
	params, err := h.conv.FromProtoSendMessageRequest(req)
	if err != nil {
		return err
	}
	for event, err := range h.handler.SendStreamingMessage(ctx, params) {
		if err != nil {
			return err
		}
		pbEvt, err := h.conv.ToProtoStreamResponse(event)
		if err != nil {
			return err
		}
		if err := stream.Send(pbEvt); err != nil {
			return err
		}
	}
	return nil
}

// GetTask implements [ourpb.A2AServiceServer] — retrieves a task by ID.
func (h *Handler) GetTask(ctx context.Context, req *a2agopb.GetTaskRequest) (*a2agopb.Task, error) {
	params, err := h.conv.FromProtoGetTaskRequest(req)
	if err != nil {
		return nil, err
	}
	task, err := h.handler.GetTask(ctx, params)
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoTask(task)
}

// ListTasks implements [ourpb.A2AServiceServer] — lists tasks matching the request filters.
func (h *Handler) ListTasks(
	ctx context.Context, req *a2agopb.ListTasksRequest,
) (*a2agopb.ListTasksResponse, error) {
	params, err := h.conv.FromProtoListTasksRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := h.handler.ListTasks(ctx, params)
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoListTasksResponse(resp)
}

// CancelTask implements [ourpb.A2AServiceServer] — requests cancellation of a task.
func (h *Handler) CancelTask(
	ctx context.Context, req *a2agopb.CancelTaskRequest,
) (*a2agopb.Task, error) {
	taskID, err := h.conv.ExtractTaskID(req.GetName())
	if err != nil {
		return nil, err
	}
	task, err := h.handler.CancelTask(ctx, &a2a.CancelTaskRequest{ID: taskID})
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoTask(task)
}

// TaskSubscription implements [ourpb.A2AServiceServer] — (re)subscribes to a task's event stream.
func (h *Handler) TaskSubscription(
	ctx context.Context,
	req *a2agopb.TaskSubscriptionRequest,
	stream slimrpc.RequestStream[*a2agopb.StreamResponse],
) error {
	taskID, err := h.conv.ExtractTaskID(req.GetName())
	if err != nil {
		return err
	}
	for event, err := range h.handler.SubscribeToTask(ctx, &a2a.SubscribeToTaskRequest{ID: taskID}) {
		if err != nil {
			return err
		}
		pbEvt, err := h.conv.ToProtoStreamResponse(event)
		if err != nil {
			return err
		}
		if err := stream.Send(pbEvt); err != nil {
			return err
		}
	}
	return nil
}

// CreateTaskPushNotificationConfig implements [ourpb.A2AServiceServer] — creates a push notification config.
func (h *Handler) CreateTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.CreateTaskPushNotificationConfigRequest,
) (*a2agopb.TaskPushNotificationConfig, error) {
	params, err := h.conv.FromProtoCreateTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	result, err := h.handler.CreateTaskPushConfig(ctx, params)
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoTaskPushConfig(result)
}

// GetTaskPushNotificationConfig implements [ourpb.A2AServiceServer] — retrieves a push notification config.
func (h *Handler) GetTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.GetTaskPushNotificationConfigRequest,
) (*a2agopb.TaskPushNotificationConfig, error) {
	params, err := h.conv.FromProtoGetTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	result, err := h.handler.GetTaskPushConfig(ctx, params)
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoTaskPushConfig(result)
}

// ListTaskPushNotificationConfig implements [ourpb.A2AServiceServer] — lists push notification configs for a task.
func (h *Handler) ListTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.ListTaskPushNotificationConfigRequest,
) (*a2agopb.ListTaskPushNotificationConfigResponse, error) {
	params, err := h.conv.FromProtoListTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	configs, err := h.handler.ListTaskPushConfigs(ctx, params)
	if err != nil {
		return nil, err
	}
	resp := &a2a.ListTaskPushConfigResponse{Configs: configs}
	return h.conv.ToProtoListTaskPushConfigResponse(resp)
}

// GetAgentCard implements [ourpb.A2AServiceServer] — retrieves the agent's extended card.
func (h *Handler) GetAgentCard(
	ctx context.Context, _ *a2agopb.GetAgentCardRequest,
) (*a2agopb.AgentCard, error) {
	card, err := h.handler.GetExtendedAgentCard(ctx, &a2a.GetExtendedAgentCardRequest{})
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoAgentCard(card)
}

// DeleteTaskPushNotificationConfig implements [ourpb.A2AServiceServer] — deletes a push notification config.
func (h *Handler) DeleteTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.DeleteTaskPushNotificationConfigRequest,
) (*emptypb.Empty, error) {
	params, err := h.conv.FromProtoDeleteTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	if err := h.handler.DeleteTaskPushConfig(ctx, params); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
