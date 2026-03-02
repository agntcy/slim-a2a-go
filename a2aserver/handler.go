// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aserver

import (
	"context"

	a2a "github.com/a2aproject/a2a-go/a2a"
	a2agopb "github.com/a2aproject/a2a-go/a2apb"
	"github.com/a2aproject/a2a-go/a2apb/pbconv"
	"github.com/a2aproject/a2a-go/a2asrv"
	slim_bindings "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/slim-bindings-go/slimrpc"
	"google.golang.org/protobuf/types/known/emptypb"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb"
)

// Handler implements ourpb.A2AServiceServer by delegating to an a2asrv.RequestHandler.
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
	result, err := h.handler.OnSendMessage(ctx, params)
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
	for event, err := range h.handler.OnSendMessageStream(ctx, params) {
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
	task, err := h.handler.OnGetTask(ctx, params)
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoTask(task)
}

// fromProtoTaskState converts a proto TaskState enum value to the domain a2a.TaskState string.
func fromProtoTaskState(state a2agopb.TaskState) a2a.TaskState {
	switch state {
	case a2agopb.TaskState_TASK_STATE_AUTH_REQUIRED:
		return a2a.TaskStateAuthRequired
	case a2agopb.TaskState_TASK_STATE_CANCELLED: //nolint:misspell // proto-generated enum name
		return a2a.TaskStateCanceled
	case a2agopb.TaskState_TASK_STATE_COMPLETED:
		return a2a.TaskStateCompleted
	case a2agopb.TaskState_TASK_STATE_FAILED:
		return a2a.TaskStateFailed
	case a2agopb.TaskState_TASK_STATE_INPUT_REQUIRED:
		return a2a.TaskStateInputRequired
	case a2agopb.TaskState_TASK_STATE_REJECTED:
		return a2a.TaskStateRejected
	case a2agopb.TaskState_TASK_STATE_SUBMITTED:
		return a2a.TaskStateSubmitted
	case a2agopb.TaskState_TASK_STATE_WORKING:
		return a2a.TaskStateWorking
	default:
		return a2a.TaskStateUnspecified
	}
}

func (h *Handler) ListTasks(
	ctx context.Context, req *a2agopb.ListTasksRequest,
) (*a2agopb.ListTasksResponse, error) {
	// Convert proto request to domain type manually (pbconv.FromProtoListTasksRequest does not exist).
	domainReq := &a2a.ListTasksRequest{
		ContextID:        req.GetContextId(),
		PageSize:         int(req.GetPageSize()),
		PageToken:        req.GetPageToken(),
		HistoryLength:    int(req.GetHistoryLength()),
		IncludeArtifacts: req.GetIncludeArtifacts(),
	}
	if req.GetStatus() != a2agopb.TaskState_TASK_STATE_UNSPECIFIED {
		domainReq.Status = fromProtoTaskState(req.GetStatus())
	}
	if req.GetLastUpdatedTime() != nil {
		t := req.GetLastUpdatedTime().AsTime()
		domainReq.LastUpdatedAfter = &t
	}

	resp, err := h.handler.OnListTasks(ctx, domainReq)
	if err != nil {
		return nil, err
	}

	// Build proto response manually (pbconv.ToProtoListTasksResponse does not exist).
	protoTasks := make([]*a2agopb.Task, len(resp.Tasks))
	for i, task := range resp.Tasks {
		pt, err := pbconv.ToProtoTask(task)
		if err != nil {
			return nil, err
		}
		protoTasks[i] = pt
	}
	return &a2agopb.ListTasksResponse{
		Tasks:         protoTasks,
		NextPageToken: resp.NextPageToken,
		TotalSize:     int32(resp.TotalSize), //nolint:gosec // page size fits in int32
	}, nil
}

func (h *Handler) CancelTask(
	ctx context.Context, req *a2agopb.CancelTaskRequest,
) (*a2agopb.Task, error) {
	taskID, err := pbconv.ExtractTaskID(req.GetName())
	if err != nil {
		return nil, err
	}
	task, err := h.handler.OnCancelTask(ctx, &a2a.TaskIDParams{ID: taskID})
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
	for event, err := range h.handler.OnResubscribeToTask(ctx, &a2a.TaskIDParams{ID: taskID}) {
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
	result, err := h.handler.OnSetTaskPushConfig(ctx, params)
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
	result, err := h.handler.OnGetTaskPushConfig(ctx, params)
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoTaskPushConfig(result)
}

func (h *Handler) ListTaskPushNotificationConfig(
	ctx context.Context,
	req *a2agopb.ListTaskPushNotificationConfigRequest,
) (*a2agopb.ListTaskPushNotificationConfigResponse, error) {
	taskID, err := pbconv.ExtractTaskID(req.GetParent())
	if err != nil {
		return nil, err
	}
	results, err := h.handler.OnListTaskPushConfig(ctx, &a2a.ListTaskPushConfigParams{TaskID: taskID})
	if err != nil {
		return nil, err
	}
	return pbconv.ToProtoListTaskPushConfig(results)
}

func (h *Handler) GetAgentCard(ctx context.Context, _ *a2agopb.GetAgentCardRequest) (*a2agopb.AgentCard, error) {
	card, err := h.handler.OnGetExtendedAgentCard(ctx)
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
	if err := h.handler.OnDeleteTaskPushConfig(ctx, params); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
