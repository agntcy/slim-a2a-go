// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	"context"

	a2agopb "github.com/a2aproject/a2a-go/a2apb"
	a2a "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2apb/v0/pbconv"
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
}

// NewHandler creates a new SLIM server handler wrapping the given RequestHandler.
func NewHandler(handler a2asrv.RequestHandler) *Handler {
	return &Handler{handler: handler}
}

// RegisterWith registers the A2A service with a SLIM server.
func (h *Handler) RegisterWith(s *slim_bindings.Server) {
	ourpb.RegisterA2AServiceServer(s, h)
}

func (h *Handler) SendMessage(
	ctx context.Context, req *a2agopb.SendMessageRequest,
) (*a2agopb.SendMessageResponse, error) {
	params, err := pbconv.FromProtoSendMessageRequest(req)
	if err != nil {
		return nil, err
	}
	result, err := h.handler.SendMessage(ctx, params)
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoSendMessageResponse(result)
}

func (h *Handler) SendStreamingMessage(
	ctx context.Context,
	req *a2agopb.SendMessageRequest,
	stream slimrpc.RequestStream[*a2agopb.StreamResponse],
) error {
	params, err := pbconv.FromProtoSendMessageRequest(req)
	if err != nil {
		return err
	}
	for event, err := range h.handler.SendStreamingMessage(ctx, params) {
		if err != nil {
			return err
		}
		pbEvt, err := pbconv.ToProtoStreamResponse(event)
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
	params, err := pbconv.FromProtoGetTaskRequest(req)
	if err != nil {
		return nil, err
	}
	task, err := h.handler.GetTask(ctx, params)
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoTask(task)
}

func (h *Handler) ListTasks(
	ctx context.Context, req *a2agopb.ListTasksRequest,
) (*a2agopb.ListTasksResponse, error) {
	params, err := pbconv.FromProtoListTasksRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := h.handler.ListTasks(ctx, params)
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoListTasksResponse(resp)
}

func (h *Handler) CancelTask(
	ctx context.Context, req *a2agopb.CancelTaskRequest,
) (*a2agopb.Task, error) {
	taskID, err := pbconv.ExtractTaskID(req.GetName())
	if err != nil {
		return nil, err
	}
	task, err := h.handler.CancelTask(ctx, &a2a.CancelTaskRequest{ID: taskID})
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoTask(task)
}

func (h *Handler) TaskSubscription(
	ctx context.Context,
	req *a2agopb.TaskSubscriptionRequest,
	stream slimrpc.RequestStream[*a2agopb.StreamResponse],
) error {
	taskID, err := pbconv.ExtractTaskID(req.GetName())
	if err != nil {
		return err
	}
	for event, err := range h.handler.SubscribeToTask(ctx, &a2a.SubscribeToTaskRequest{ID: taskID}) {
		if err != nil {
			return err
		}
		pbEvt, err := pbconv.ToProtoStreamResponse(event)
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
	req *a2agopb.CreateTaskPushNotificationConfigRequest,
) (*a2agopb.TaskPushNotificationConfig, error) {
	params, err := pbconv.FromProtoCreateTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	result, err := h.handler.CreateTaskPushConfig(ctx, params)
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoTaskPushConfig(result)
}

func (h *Handler) GetTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.GetTaskPushNotificationConfigRequest,
) (*a2agopb.TaskPushNotificationConfig, error) {
	params, err := pbconv.FromProtoGetTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	result, err := h.handler.GetTaskPushConfig(ctx, params)
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoTaskPushConfig(result)
}

func (h *Handler) ListTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.ListTaskPushNotificationConfigRequest,
) (*a2agopb.ListTaskPushNotificationConfigResponse, error) {
	params, err := pbconv.FromProtoListTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	configs, err := h.handler.ListTaskPushConfigs(ctx, params)
	if err != nil {
		return nil, err
	}
	resp := &a2a.ListTaskPushConfigResponse{Configs: configs}
	return pbconv.ToProtoListTaskPushConfigResponse(resp)
}

func (h *Handler) GetAgentCard(
	ctx context.Context, _ *a2agopb.GetAgentCardRequest,
) (*a2agopb.AgentCard, error) {
	card, err := h.handler.GetExtendedAgentCard(ctx, &a2a.GetExtendedAgentCardRequest{})
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoAgentCard(card)
}

func (h *Handler) DeleteTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.DeleteTaskPushNotificationConfigRequest,
) (*emptypb.Empty, error) {
	params, err := pbconv.FromProtoDeleteTaskPushConfigRequest(req)
	if err != nil {
		return nil, err
	}
	if err := h.handler.DeleteTaskPushConfig(ctx, params); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
