// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	"context"

	a2a "github.com/a2aproject/a2a-go/v2/a2a"
	a2agopb "github.com/a2aproject/a2a-go/v2/a2apb/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	slim_bindings "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/slim-bindings-go/slimrpc"
	"google.golang.org/protobuf/types/known/emptypb"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb/v1"
)

// Handler implements ourpb.A2AServiceServer by delegating to an a2asrv.RequestHandler.
type Handler struct {
	ourpb.UnimplementedA2AServiceServer
	handler a2asrv.RequestHandler
	conv    handlerConverter
}

// NewHandler creates a new SLIM server handler wrapping the given RequestHandler.
func NewHandler(handler a2asrv.RequestHandler, opts ...HandlerOption) *Handler {
	h := &Handler{handler: handler, conv: defaultHandlerConverter{}}
	for _, o := range opts {
		o(h)
	}
	return h
}

// RegisterWith registers the A2A service with a SLIM server.
func (h *Handler) RegisterWith(s *slim_bindings.Server) {
	ourpb.RegisterA2AServiceServer(s, h)
}

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

func (h *Handler) CancelTask(
	ctx context.Context, req *a2agopb.CancelTaskRequest,
) (*a2agopb.Task, error) {
	params, err := h.conv.FromProtoCancelTaskRequest(req)
	if err != nil {
		return nil, err
	}
	task, err := h.handler.CancelTask(ctx, params)
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoTask(task)
}

func (h *Handler) SubscribeToTask(
	ctx context.Context,
	req *a2agopb.SubscribeToTaskRequest,
	stream slimrpc.RequestStream[*a2agopb.StreamResponse],
) error {
	params, err := h.conv.FromProtoSubscribeToTaskRequest(req)
	if err != nil {
		return err
	}
	for event, err := range h.handler.SubscribeToTask(ctx, params) {
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

func (h *Handler) CreateTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.TaskPushNotificationConfig,
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

func (h *Handler) ListTaskPushNotificationConfigs(
	ctx context.Context,
	req *a2agopb.ListTaskPushNotificationConfigsRequest,
) (*a2agopb.ListTaskPushNotificationConfigsResponse, error) {
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

func (h *Handler) GetExtendedAgentCard(
	ctx context.Context, req *a2agopb.GetExtendedAgentCardRequest,
) (*a2agopb.AgentCard, error) {
	params, err := h.conv.FromProtoGetExtendedAgentCardRequest(req)
	if err != nil {
		return nil, err
	}
	card, err := h.handler.GetExtendedAgentCard(ctx, params)
	if err != nil {
		return nil, err
	}
	return h.conv.ToProtoAgentCard(card)
}

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
