// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"reflect"
	"sort"
	"testing"
	"time"

	a2agopb "github.com/a2aproject/a2a-go/a2apb"
	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2apb/v0/pbconv"
	slim_bindings "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/slim-bindings-go/slimrpc"
	"google.golang.org/protobuf/types/known/emptypb"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb/v0"
)

// mockProtoServer implements ourpb.A2AServiceServer with configurable function fields.
// Unset fields fall through to UnimplementedA2AServiceServer which returns an error.
type mockProtoServer struct {
	ourpb.UnimplementedA2AServiceServer
	sendMessageFn                      func(context.Context, *a2agopb.SendMessageRequest) (*a2agopb.SendMessageResponse, error)
	sendStreamingMessageFn             func(context.Context, *a2agopb.SendMessageRequest, slimrpc.RequestStream[*a2agopb.StreamResponse]) error
	getTaskFn                          func(context.Context, *a2agopb.GetTaskRequest) (*a2agopb.Task, error)
	listTasksFn                        func(context.Context, *a2agopb.ListTasksRequest) (*a2agopb.ListTasksResponse, error)
	cancelTaskFn                       func(context.Context, *a2agopb.CancelTaskRequest) (*a2agopb.Task, error)
	taskSubscriptionFn                 func(context.Context, *a2agopb.TaskSubscriptionRequest, slimrpc.RequestStream[*a2agopb.StreamResponse]) error
	createTaskPushNotificationConfigFn func(context.Context, *a2agopb.CreateTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error)
	getTaskPushNotificationConfigFn    func(context.Context, *a2agopb.GetTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error)
	listTaskPushNotificationConfigFn   func(context.Context, *a2agopb.ListTaskPushNotificationConfigRequest) (*a2agopb.ListTaskPushNotificationConfigResponse, error)
	getAgentCardFn                     func(context.Context, *a2agopb.GetAgentCardRequest) (*a2agopb.AgentCard, error)
	deleteTaskPushNotificationConfigFn func(context.Context, *a2agopb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error)
}

var _ ourpb.A2AServiceServer = (*mockProtoServer)(nil)

func (m *mockProtoServer) SendMessage(ctx context.Context, req *a2agopb.SendMessageRequest) (*a2agopb.SendMessageResponse, error) {
	if m.sendMessageFn != nil {
		return m.sendMessageFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.SendMessage(ctx, req)
}

func (m *mockProtoServer) SendStreamingMessage(ctx context.Context, req *a2agopb.SendMessageRequest, stream slimrpc.RequestStream[*a2agopb.StreamResponse]) error {
	if m.sendStreamingMessageFn != nil {
		return m.sendStreamingMessageFn(ctx, req, stream)
	}
	return m.UnimplementedA2AServiceServer.SendStreamingMessage(ctx, req, stream)
}

func (m *mockProtoServer) GetTask(ctx context.Context, req *a2agopb.GetTaskRequest) (*a2agopb.Task, error) {
	if m.getTaskFn != nil {
		return m.getTaskFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.GetTask(ctx, req)
}

func (m *mockProtoServer) ListTasks(ctx context.Context, req *a2agopb.ListTasksRequest) (*a2agopb.ListTasksResponse, error) {
	if m.listTasksFn != nil {
		return m.listTasksFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.ListTasks(ctx, req)
}

func (m *mockProtoServer) CancelTask(ctx context.Context, req *a2agopb.CancelTaskRequest) (*a2agopb.Task, error) {
	if m.cancelTaskFn != nil {
		return m.cancelTaskFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.CancelTask(ctx, req)
}

func (m *mockProtoServer) TaskSubscription(ctx context.Context, req *a2agopb.TaskSubscriptionRequest, stream slimrpc.RequestStream[*a2agopb.StreamResponse]) error {
	if m.taskSubscriptionFn != nil {
		return m.taskSubscriptionFn(ctx, req, stream)
	}
	return m.UnimplementedA2AServiceServer.TaskSubscription(ctx, req, stream)
}

func (m *mockProtoServer) CreateTaskPushNotificationConfig(ctx context.Context, req *a2agopb.CreateTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error) {
	if m.createTaskPushNotificationConfigFn != nil {
		return m.createTaskPushNotificationConfigFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.CreateTaskPushNotificationConfig(ctx, req)
}

func (m *mockProtoServer) GetTaskPushNotificationConfig(ctx context.Context, req *a2agopb.GetTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error) {
	if m.getTaskPushNotificationConfigFn != nil {
		return m.getTaskPushNotificationConfigFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.GetTaskPushNotificationConfig(ctx, req)
}

func (m *mockProtoServer) ListTaskPushNotificationConfig(ctx context.Context, req *a2agopb.ListTaskPushNotificationConfigRequest) (*a2agopb.ListTaskPushNotificationConfigResponse, error) {
	if m.listTaskPushNotificationConfigFn != nil {
		return m.listTaskPushNotificationConfigFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.ListTaskPushNotificationConfig(ctx, req)
}

func (m *mockProtoServer) GetAgentCard(ctx context.Context, req *a2agopb.GetAgentCardRequest) (*a2agopb.AgentCard, error) {
	if m.getAgentCardFn != nil {
		return m.getAgentCardFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.GetAgentCard(ctx, req)
}

func (m *mockProtoServer) DeleteTaskPushNotificationConfig(ctx context.Context, req *a2agopb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	if m.deleteTaskPushNotificationConfigFn != nil {
		return m.deleteTaskPushNotificationConfigFn(ctx, req)
	}
	return m.UnimplementedA2AServiceServer.DeleteTaskPushNotificationConfig(ctx, req)
}

// startTestTransport registers srv directly with a real in-process SLIM server
// (bypassing NewHandler) and returns a Transport wrapping a paired client channel.
// The server is shut down and the channel destroyed via t.Cleanup.
func startTestTransport(t *testing.T, srv ourpb.A2AServiceServer, opts ...TransportOption) *Transport {
	t.Helper()
	svc := slim_bindings.GetGlobalService()

	serverName := slim_bindings.NewName("agntcy", t.Name(), "srv")
	serverApp, err := svc.CreateAppWithSecret(serverName, "test-secret-for-slim-unit-tests-only")
	if err != nil {
		t.Fatalf("create server app: %v", err)
	}

	server := slim_bindings.NewServer(serverApp, serverName)
	ourpb.RegisterA2AServiceServer(server, srv)
	go func() {
		if err := server.Serve(); err != nil {
			t.Logf("server.Serve() exited with error: %v", err)
		}
	}()

	// Give the server a moment to start before the client tries to connect.
	time.Sleep(10 * time.Millisecond)

	clientName := slim_bindings.NewName("agntcy", t.Name(), "cli")
	clientApp, err := svc.CreateAppWithSecret(clientName, "test-secret-for-slim-unit-tests-only")
	if err != nil {
		t.Fatalf("create client app: %v", err)
	}

	channel := slim_bindings.NewChannel(clientApp, serverName)
	transport := NewTransport(channel, opts...)

	t.Cleanup(func() {
		if err := transport.Destroy(); err != nil {
			t.Logf("transport.Destroy: %v", err)
		}
		server.Shutdown()
	})
	return transport
}

// collectStreamEvents ranges over an iter.Seq2[a2a.Event, error] and collects
// all yielded events. Returns the events and any first error encountered.
func collectStreamEvents(seq iter.Seq2[a2a.Event, error]) ([]a2a.Event, error) {
	var events []a2a.Event
	var streamErr error
	for event, err := range seq {
		if err != nil {
			streamErr = err
			break
		}
		events = append(events, event)
	}
	return events, streamErr
}

func TestTransport_SendMessage(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("transport-task-1")
	msgID := "transport-msg-1"

	mock := &mockProtoServer{
		sendMessageFn: func(_ context.Context, req *a2agopb.SendMessageRequest) (*a2agopb.SendMessageResponse, error) {
			if req.GetRequest().GetMessageId() == "trigger-error" {
				return nil, errors.New("mock server error")
			}
			return &a2agopb.SendMessageResponse{
				Payload: &a2agopb.SendMessageResponse_Msg{
					Msg: &a2agopb.Message{
						MessageId: req.GetRequest().GetMessageId() + "-response",
						TaskId:    req.GetRequest().GetTaskId(),
						Role:      a2agopb.Role_ROLE_AGENT,
						Parts:     req.GetRequest().GetParts(),
					},
				},
			}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.SendMessageRequest
		want    a2a.SendMessageResult
		wantErr bool
	}{
		{
			name: "success",
			req: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:     msgID,
					TaskID: taskID,
					Role:   a2a.MessageRoleUser,
					Parts:  a2a.ContentParts{a2a.NewTextPart("hello")},
				},
			},
			want: &a2a.Message{
				ID:     msgID + "-response",
				TaskID: taskID,
				Role:   a2a.MessageRoleAgent,
				Parts:  a2a.ContentParts{a2a.NewTextPart("hello")},
			},
		},
		{
			name: "handler error",
			req: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:    "trigger-error",
					Role:  a2a.MessageRoleUser,
					Parts: a2a.ContentParts{a2a.NewTextPart("trigger")},
				},
			},
			wantErr: true,
		},
		{
			name: "to proto error",
			req: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:       "bad-meta",
					Role:     a2a.MessageRoleUser,
					Parts:    a2a.ContentParts{a2a.NewTextPart("trigger")},
					Metadata: map[string]any{"bad": struct{}{}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transport.SendMessage(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SendMessage() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("SendMessage() got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SendMessage() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_SendStreamingMessage(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("stream-task-1")
	contextID := "stream-ctx-1"
	msgID := "stream-msg-1"

	mock := &mockProtoServer{
		sendStreamingMessageFn: func(_ context.Context, req *a2agopb.SendMessageRequest, stream slimrpc.RequestStream[*a2agopb.StreamResponse]) error {
			if req.GetRequest().GetMessageId() == "trigger-error" {
				return errors.New("mock server stream error")
			}
			if req.GetRequest().GetMessageId() == "nil-payload" {
				// Send a bare StreamResponse (no payload) to trigger FromProtoStreamResponse error.
				return stream.Send(&a2agopb.StreamResponse{})
			}
			tID := req.GetRequest().GetTaskId()
			cID := req.GetRequest().GetContextId()
			responses := []*a2agopb.StreamResponse{
				{
					Payload: &a2agopb.StreamResponse_Task{
						Task: &a2agopb.Task{
							Id:        tID,
							ContextId: cID,
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED},
						},
					},
				},
				{
					Payload: &a2agopb.StreamResponse_StatusUpdate{
						StatusUpdate: &a2agopb.TaskStatusUpdateEvent{
							TaskId:    tID,
							ContextId: cID,
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING},
						},
					},
				},
				{
					Payload: &a2agopb.StreamResponse_Msg{
						Msg: &a2agopb.Message{
							MessageId: req.GetRequest().GetMessageId() + "-response",
							TaskId:    tID,
							Role:      a2agopb.Role_ROLE_AGENT,
							Parts:     req.GetRequest().GetParts(),
						},
					},
				},
			}
			for _, r := range responses {
				if err := stream.Send(r); err != nil {
					return err
				}
			}
			return nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name      string
		req       *a2a.SendMessageRequest
		wantCount int
		wantErr   bool
	}{
		{
			name: "success",
			req: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:        msgID,
					TaskID:    taskID,
					ContextID: contextID,
					Role:      a2a.MessageRoleUser,
					Parts:     a2a.ContentParts{a2a.NewTextPart("streaming hello")},
				},
			},
			wantCount: 3,
		},
		{
			name: "handler error",
			req: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:    "trigger-error",
					Role:  a2a.MessageRoleUser,
					Parts: a2a.ContentParts{a2a.NewTextPart("trigger")},
				},
			},
			wantErr: true,
		},
		{
			name: "nil payload response",
			req: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:    "nil-payload",
					Role:  a2a.MessageRoleUser,
					Parts: a2a.ContentParts{a2a.NewTextPart("trigger")},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := transport.SendStreamingMessage(ctx, nil, tt.req)
			events, err := collectStreamEvents(seq)
			if tt.wantErr {
				// SLIM streaming: errors may propagate via iterator error or result in 0 events.
				if err != nil || len(events) == 0 {
					return
				}
				t.Fatalf("SendStreamingMessage() expected error or empty stream, got %d events", len(events))
			}
			if err != nil {
				t.Fatalf("SendStreamingMessage() got unexpected error: %v", err)
			}
			if len(events) != tt.wantCount {
				t.Fatalf("SendStreamingMessage() got %d events, want %d", len(events), tt.wantCount)
			}
			// Verify event types in order: Task, TaskStatusUpdateEvent, Message.
			if _, ok := events[0].(*a2a.Task); !ok {
				t.Errorf("event[0] got type %T, want *a2a.Task", events[0])
			}
			if _, ok := events[1].(*a2a.TaskStatusUpdateEvent); !ok {
				t.Errorf("event[1] got type %T, want *a2a.TaskStatusUpdateEvent", events[1])
			}
			if _, ok := events[2].(*a2a.Message); !ok {
				t.Errorf("event[2] got type %T, want *a2a.Message", events[2])
			}
		})
	}

	t.Run("consumer stops early", func(t *testing.T) {
		seq := transport.SendStreamingMessage(ctx, nil, &a2a.SendMessageRequest{
			Message: &a2a.Message{
				ID:        msgID,
				TaskID:    taskID,
				ContextID: contextID,
				Role:      a2a.MessageRoleUser,
				Parts:     a2a.ContentParts{a2a.NewTextPart("streaming hello")},
			},
		})
		var count int
		for _, err := range seq {
			if err != nil {
				t.Fatalf("unexpected error before break: %v", err)
			}
			count++
			break // stop early to exercise the !yield branch in the iterator
		}
		if count == 0 {
			t.Fatal("consumer stops early: expected at least 1 event before break")
		}
	})
}

func TestTransport_GetTask(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("gt-task-1")
	historyLen := 5

	mock := &mockProtoServer{
		getTaskFn: func(_ context.Context, req *a2agopb.GetTaskRequest) (*a2agopb.Task, error) {
			if req.GetName() == "tasks/handler-error" {
				return nil, errors.New("task not found")
			}
			// v0 Name format: "tasks/<id>" — extract id after the slash
			id := req.GetName()[len("tasks/"):]
			return &a2agopb.Task{
				Id:        id,
				ContextId: "test-ctx",
				Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED},
			}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.GetTaskRequest
		want    *a2a.Task
		wantErr bool
	}{
		{
			name: "success",
			req:  &a2a.GetTaskRequest{ID: taskID},
			want: &a2a.Task{
				ID:        taskID,
				ContextID: "test-ctx",
				Status:    a2a.TaskStatus{State: a2a.TaskStateSubmitted},
				Artifacts: []*a2a.Artifact{},
				History:   []*a2a.Message{},
			},
		},
		{
			name: "success with history",
			req:  &a2a.GetTaskRequest{ID: taskID, HistoryLength: &historyLen},
			want: &a2a.Task{
				ID:        taskID,
				ContextID: "test-ctx",
				Status:    a2a.TaskStatus{State: a2a.TaskStateSubmitted},
				Artifacts: []*a2a.Artifact{},
				History:   []*a2a.Message{},
			},
		},
		{
			name:    "handler error",
			req:     &a2a.GetTaskRequest{ID: "handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transport.GetTask(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetTask() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetTask() got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetTask() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_ListTasks(t *testing.T) {
	ctx := t.Context()
	taskID1 := a2a.TaskID("lt-task-1")
	taskID2 := a2a.TaskID("lt-task-2")

	mock := &mockProtoServer{
		listTasksFn: func(_ context.Context, req *a2agopb.ListTasksRequest) (*a2agopb.ListTasksResponse, error) {
			tasks := []*a2agopb.Task{
				{Id: string(taskID1), ContextId: "ctx-a", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED}},
				{Id: string(taskID2), ContextId: "ctx-b", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_COMPLETED}},
			}
			if req.GetContextId() != "" {
				var filtered []*a2agopb.Task
				for _, tsk := range tasks {
					if tsk.GetContextId() == req.GetContextId() {
						filtered = append(filtered, tsk)
					}
				}
				tasks = filtered
			}
			return &a2agopb.ListTasksResponse{
				Tasks:         tasks,
				TotalSize:     int32(len(tasks)),
				NextPageToken: "",
			}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.ListTasksRequest
		want    *a2a.ListTasksResponse
		wantErr bool
	}{
		{
			name: "all tasks",
			req:  &a2a.ListTasksRequest{},
			want: &a2a.ListTasksResponse{
				Tasks: []*a2a.Task{
					{ID: taskID1, ContextID: "ctx-a", Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted}, Artifacts: []*a2a.Artifact{}, History: []*a2a.Message{}},
					{ID: taskID2, ContextID: "ctx-b", Status: a2a.TaskStatus{State: a2a.TaskStateCompleted}, Artifacts: []*a2a.Artifact{}, History: []*a2a.Message{}},
				},
				TotalSize: 2,
				PageSize:  2, // v0 pbconv sets PageSize = len(tasks)
			},
		},
		{
			name: "context filter",
			req:  &a2a.ListTasksRequest{ContextID: "ctx-a"},
			want: &a2a.ListTasksResponse{
				Tasks: []*a2a.Task{
					{ID: taskID1, ContextID: "ctx-a", Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted}, Artifacts: []*a2a.Artifact{}, History: []*a2a.Message{}},
				},
				TotalSize: 1,
				PageSize:  1, // v0 pbconv sets PageSize = len(tasks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transport.ListTasks(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ListTasks() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ListTasks() got unexpected error: %v", err)
			}
			sort.Slice(got.Tasks, func(i, j int) bool { return got.Tasks[i].ID < got.Tasks[j].ID })
			sort.Slice(tt.want.Tasks, func(i, j int) bool { return tt.want.Tasks[i].ID < tt.want.Tasks[j].ID })
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ListTasks() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_CancelTask(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("cancel-task-1")

	mock := &mockProtoServer{
		cancelTaskFn: func(_ context.Context, req *a2agopb.CancelTaskRequest) (*a2agopb.Task, error) {
			if req.GetName() == "tasks/handler-error" {
				return nil, errors.New("task not found")
			}
			id := req.GetName()[len("tasks/"):]
			return &a2agopb.Task{
				Id:        id,
				ContextId: "test-ctx",
				Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_CANCELLED},
			}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.CancelTaskRequest
		want    *a2a.Task
		wantErr bool
	}{
		{
			name: "success",
			req:  &a2a.CancelTaskRequest{ID: taskID},
			want: &a2a.Task{
				ID:        taskID,
				ContextID: "test-ctx",
				Status:    a2a.TaskStatus{State: a2a.TaskStateCanceled},
				Artifacts: []*a2a.Artifact{},
				History:   []*a2a.Message{},
			},
		},
		{
			name:    "handler error",
			req:     &a2a.CancelTaskRequest{ID: "handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transport.CancelTask(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CancelTask() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CancelTask() got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CancelTask() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_SubscribeToTask(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("sub-task-1")

	mock := &mockProtoServer{
		taskSubscriptionFn: func(_ context.Context, req *a2agopb.TaskSubscriptionRequest, stream slimrpc.RequestStream[*a2agopb.StreamResponse]) error {
			if req.GetName() == "tasks/handler-error" {
				return errors.New("subscribe error")
			}
			if req.GetName() == "tasks/nil-payload" {
				// Send a bare StreamResponse (no payload) to trigger FromProtoStreamResponse error.
				return stream.Send(&a2agopb.StreamResponse{})
			}
			id := req.GetName()[len("tasks/"):]
			responses := []*a2agopb.StreamResponse{
				{
					Payload: &a2agopb.StreamResponse_Task{
						Task: &a2agopb.Task{
							Id:        id,
							ContextId: "sub-ctx",
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING},
						},
					},
				},
				{
					Payload: &a2agopb.StreamResponse_StatusUpdate{
						StatusUpdate: &a2agopb.TaskStatusUpdateEvent{
							TaskId:    id,
							ContextId: "sub-ctx",
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_COMPLETED},
						},
					},
				},
			}
			for _, r := range responses {
				if err := stream.Send(r); err != nil {
					return err
				}
			}
			return nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name      string
		req       *a2a.SubscribeToTaskRequest
		wantCount int
		wantErr   bool
	}{
		{
			name:      "success",
			req:       &a2a.SubscribeToTaskRequest{ID: taskID},
			wantCount: 2,
		},
		{
			name:    "handler error",
			req:     &a2a.SubscribeToTaskRequest{ID: "handler-error"},
			wantErr: true,
		},
		{
			name:    "nil payload response",
			req:     &a2a.SubscribeToTaskRequest{ID: "nil-payload"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := transport.SubscribeToTask(ctx, nil, tt.req)
			events, err := collectStreamEvents(seq)
			if tt.wantErr {
				// SLIM streaming: errors may propagate via iterator error or result in 0 events.
				if err != nil || len(events) == 0 {
					return
				}
				t.Fatalf("SubscribeToTask() expected error or empty stream, got %d events", len(events))
			}
			if err != nil {
				t.Fatalf("SubscribeToTask() got unexpected error: %v", err)
			}
			if len(events) != tt.wantCount {
				t.Fatalf("SubscribeToTask() got %d events, want %d", len(events), tt.wantCount)
			}
			if _, ok := events[0].(*a2a.Task); !ok {
				t.Errorf("event[0] got type %T, want *a2a.Task", events[0])
			}
			if _, ok := events[1].(*a2a.TaskStatusUpdateEvent); !ok {
				t.Errorf("event[1] got type %T, want *a2a.TaskStatusUpdateEvent", events[1])
			}
		})
	}

	t.Run("consumer stops early", func(t *testing.T) {
		seq := transport.SubscribeToTask(ctx, nil, &a2a.SubscribeToTaskRequest{ID: taskID})
		var count int
		for _, err := range seq {
			if err != nil {
				t.Fatalf("unexpected error before break: %v", err)
			}
			count++
			break // stop early to exercise the !yield branch in the iterator
		}
		if count == 0 {
			t.Fatal("consumer stops early: expected at least 1 event before break")
		}
	})
}

func TestTransport_CreateTaskPushConfig(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("push-task-1")
	configID := "cfg-1"

	mock := &mockProtoServer{
		createTaskPushNotificationConfigFn: func(_ context.Context, req *a2agopb.CreateTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error) {
			if req.GetParent() == "tasks/handler-error" {
				return nil, errors.New("create config error")
			}
			cfgID := req.GetConfig().GetPushNotificationConfig().GetId()
			taskName := req.GetParent()
			return &a2agopb.TaskPushNotificationConfig{
				Name: fmt.Sprintf("%s/pushNotificationConfigs/%s", taskName, cfgID),
				PushNotificationConfig: &a2agopb.PushNotificationConfig{
					Id: cfgID,
				},
			}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.CreateTaskPushConfigRequest
		want    *a2a.TaskPushConfig
		wantErr bool
	}{
		{
			name: "success",
			req: &a2a.CreateTaskPushConfigRequest{
				TaskID: taskID,
				Config: a2a.PushConfig{ID: configID},
			},
			want: &a2a.TaskPushConfig{
				TaskID: taskID,
				Config: a2a.PushConfig{ID: configID},
			},
		},
		{
			name: "handler error",
			req: &a2a.CreateTaskPushConfigRequest{
				TaskID: "handler-error",
				Config: a2a.PushConfig{ID: configID},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transport.CreateTaskPushConfig(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CreateTaskPushConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateTaskPushConfig() got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CreateTaskPushConfig() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_GetTaskPushConfig(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("push-task-2")
	configID := "cfg-2"

	mock := &mockProtoServer{
		getTaskPushNotificationConfigFn: func(_ context.Context, req *a2agopb.GetTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error) {
			// v0 Name format: "tasks/<taskid>/pushNotificationConfigs/<cfgid>"
			if req.GetName() == "tasks/handler-error/pushNotificationConfigs/cfg-2" {
				return nil, errors.New("config not found")
			}
			return &a2agopb.TaskPushNotificationConfig{
				Name: req.GetName(),
				PushNotificationConfig: &a2agopb.PushNotificationConfig{
					Id: configID,
				},
			}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.GetTaskPushConfigRequest
		want    *a2a.TaskPushConfig
		wantErr bool
	}{
		{
			name: "success",
			req:  &a2a.GetTaskPushConfigRequest{TaskID: taskID, ID: configID},
			want: &a2a.TaskPushConfig{
				TaskID: taskID,
				Config: a2a.PushConfig{ID: configID},
			},
		},
		{
			name:    "handler error",
			req:     &a2a.GetTaskPushConfigRequest{TaskID: "handler-error", ID: configID},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transport.GetTaskPushConfig(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetTaskPushConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetTaskPushConfig() got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetTaskPushConfig() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_ListTaskPushConfigs(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("push-task-3")
	configID := "cfg"

	mock := &mockProtoServer{
		listTaskPushNotificationConfigFn: func(_ context.Context, req *a2agopb.ListTaskPushNotificationConfigRequest) (*a2agopb.ListTaskPushNotificationConfigResponse, error) {
			if req.GetParent() == "tasks/handler-error" {
				return nil, errors.New("list configs error")
			}
			if req.GetParent() == "tasks/trigger-conv-error" {
				// Return a config with an invalid Name to trigger FromProtoTaskPushConfig parsing error.
				return &a2agopb.ListTaskPushNotificationConfigResponse{
					Configs: []*a2agopb.TaskPushNotificationConfig{
						{Name: "invalid-name"},
					},
				}, nil
			}
			return &a2agopb.ListTaskPushNotificationConfigResponse{
				Configs: []*a2agopb.TaskPushNotificationConfig{
					{
						Name: fmt.Sprintf("%s/pushNotificationConfigs/%s-1", req.GetParent(), configID),
						PushNotificationConfig: &a2agopb.PushNotificationConfig{
							Id: fmt.Sprintf("%s-1", configID),
						},
					},
					{
						Name: fmt.Sprintf("%s/pushNotificationConfigs/%s-2", req.GetParent(), configID),
						PushNotificationConfig: &a2agopb.PushNotificationConfig{
							Id: fmt.Sprintf("%s-2", configID),
						},
					},
				},
			}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.ListTaskPushConfigRequest
		want    []*a2a.TaskPushConfig
		wantErr bool
	}{
		{
			name: "success",
			req:  &a2a.ListTaskPushConfigRequest{TaskID: taskID},
			want: []*a2a.TaskPushConfig{
				{TaskID: taskID, Config: a2a.PushConfig{ID: configID + "-1"}},
				{TaskID: taskID, Config: a2a.PushConfig{ID: configID + "-2"}},
			},
		},
		{
			name:    "handler error",
			req:     &a2a.ListTaskPushConfigRequest{TaskID: "handler-error"},
			wantErr: true,
		},
		{
			name:    "response conversion error",
			req:     &a2a.ListTaskPushConfigRequest{TaskID: "trigger-conv-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transport.ListTaskPushConfigs(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ListTaskPushConfigs() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ListTaskPushConfigs() got unexpected error: %v", err)
			}
			sort.Slice(got, func(i, j int) bool { return got[i].Config.ID < got[j].Config.ID })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].Config.ID < tt.want[j].Config.ID })
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ListTaskPushConfigs() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_DeleteTaskPushConfig(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("push-task-4")
	configID := "cfg-delete"

	mock := &mockProtoServer{
		deleteTaskPushNotificationConfigFn: func(_ context.Context, req *a2agopb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
			if req.GetName() == fmt.Sprintf("tasks/handler-error/pushNotificationConfigs/%s", configID) {
				return nil, errors.New("delete error")
			}
			return &emptypb.Empty{}, nil
		},
	}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		req     *a2a.DeleteTaskPushConfigRequest
		wantErr bool
	}{
		{
			name: "success",
			req:  &a2a.DeleteTaskPushConfigRequest{TaskID: taskID, ID: configID},
		},
		{
			name:    "handler error",
			req:     &a2a.DeleteTaskPushConfigRequest{TaskID: "handler-error", ID: configID},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transport.DeleteTaskPushConfig(ctx, nil, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DeleteTaskPushConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteTaskPushConfig() got unexpected error: %v", err)
			}
		})
	}
}

func TestTransport_GetExtendedAgentCard(t *testing.T) {
	ctx := t.Context()

	// Build the proto agent card that the mock will return, then compute the
	// expected domain value via pbconv to ensure consistent comparison.
	// In v0, the primary interface data lives directly on AgentCard (not in a SupportedInterfaces
	// slice). v0 FromProtoAgentCard has no field validation, so a minimal card is sufficient.
	pbCard := &a2agopb.AgentCard{
		Name:            "Test Agent v0",
		ProtocolVersion: "0.3",
	}
	wantCard, err := pbconv.FromProtoAgentCard(pbCard)
	if err != nil {
		t.Fatalf("setup: FromProtoAgentCard failed: %v", err)
	}

	mock := &mockProtoServer{}
	transport := startTestTransport(t, mock)

	tests := []struct {
		name    string
		cardFn  func(context.Context, *a2agopb.GetAgentCardRequest) (*a2agopb.AgentCard, error)
		want    *a2a.AgentCard
		wantErr bool
	}{
		{
			name: "success",
			cardFn: func(_ context.Context, _ *a2agopb.GetAgentCardRequest) (*a2agopb.AgentCard, error) {
				return pbCard, nil
			},
			want: wantCard,
		},
		{
			name: "handler error",
			cardFn: func(_ context.Context, _ *a2agopb.GetAgentCardRequest) (*a2agopb.AgentCard, error) {
				return nil, errors.New("agent card unavailable")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.getAgentCardFn = tt.cardFn
			got, err := transport.GetExtendedAgentCard(ctx, nil, &a2a.GetExtendedAgentCardRequest{})
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetExtendedAgentCard() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetExtendedAgentCard() got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetExtendedAgentCard() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTransport_Destroy(t *testing.T) {
	svc := slim_bindings.GetGlobalService()
	name := slim_bindings.NewName("agntcy", t.Name(), "cli")
	app, err := svc.CreateAppWithSecret(name, "test-secret-for-slim-unit-tests-only")
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	channel := slim_bindings.NewChannel(app, slim_bindings.NewName("agntcy", "destroy-test", "srv"))
	transport := NewTransport(channel)
	if err := transport.Destroy(); err != nil {
		t.Fatalf("Destroy() returned unexpected error: %v", err)
	}
}

// mockTransportConverter implements transportConverter with configurable function fields.
// Unset methods fall through to the real pbconv package.
type mockTransportConverter struct {
	toProtoSendMessageRequestFn           func(*a2a.SendMessageRequest) (*a2agopb.SendMessageRequest, error)
	toProtoGetTaskRequestFn               func(*a2a.GetTaskRequest) (*a2agopb.GetTaskRequest, error)
	toProtoCancelTaskRequestFn            func(*a2a.CancelTaskRequest) (*a2agopb.CancelTaskRequest, error)
	toProtoTaskSubscriptionRequestFn      func(*a2a.SubscribeToTaskRequest) (*a2agopb.TaskSubscriptionRequest, error)
	toProtoListTasksRequestFn             func(*a2a.ListTasksRequest) (*a2agopb.ListTasksRequest, error)
	toProtoCreateTaskPushConfigRequestFn  func(*a2a.CreateTaskPushConfigRequest) (*a2agopb.CreateTaskPushNotificationConfigRequest, error)
	toProtoGetTaskPushConfigRequestFn     func(*a2a.GetTaskPushConfigRequest) (*a2agopb.GetTaskPushNotificationConfigRequest, error)
	toProtoListTaskPushConfigRequestFn    func(*a2a.ListTaskPushConfigRequest) (*a2agopb.ListTaskPushNotificationConfigRequest, error)
	toProtoDeleteTaskPushConfigRequestFn  func(*a2a.DeleteTaskPushConfigRequest) (*a2agopb.DeleteTaskPushNotificationConfigRequest, error)
	fromProtoSendMessageResponseFn        func(*a2agopb.SendMessageResponse) (a2a.SendMessageResult, error)
	fromProtoStreamResponseFn             func(*a2agopb.StreamResponse) (a2a.Event, error)
	fromProtoTaskFn                       func(*a2agopb.Task) (*a2a.Task, error)
	fromProtoListTasksResponseFn          func(*a2agopb.ListTasksResponse) (*a2a.ListTasksResponse, error)
	fromProtoTaskPushConfigFn             func(*a2agopb.TaskPushNotificationConfig) (*a2a.TaskPushConfig, error)
	fromProtoListTaskPushConfigResponseFn func(*a2agopb.ListTaskPushNotificationConfigResponse) (*a2a.ListTaskPushConfigResponse, error)
	fromProtoAgentCardFn                  func(*a2agopb.AgentCard) (*a2a.AgentCard, error)
}

var _ transportConverter = (*mockTransportConverter)(nil)

func (m *mockTransportConverter) ToProtoSendMessageRequest(req *a2a.SendMessageRequest) (*a2agopb.SendMessageRequest, error) {
	if m.toProtoSendMessageRequestFn != nil {
		return m.toProtoSendMessageRequestFn(req)
	}
	return pbconv.ToProtoSendMessageRequest(req)
}

func (m *mockTransportConverter) ToProtoGetTaskRequest(req *a2a.GetTaskRequest) (*a2agopb.GetTaskRequest, error) {
	if m.toProtoGetTaskRequestFn != nil {
		return m.toProtoGetTaskRequestFn(req)
	}
	return pbconv.ToProtoGetTaskRequest(req)
}

func (m *mockTransportConverter) ToProtoCancelTaskRequest(req *a2a.CancelTaskRequest) (*a2agopb.CancelTaskRequest, error) {
	if m.toProtoCancelTaskRequestFn != nil {
		return m.toProtoCancelTaskRequestFn(req)
	}
	return pbconv.ToProtoCancelTaskRequest(req)
}

func (m *mockTransportConverter) ToProtoTaskSubscriptionRequest(req *a2a.SubscribeToTaskRequest) (*a2agopb.TaskSubscriptionRequest, error) {
	if m.toProtoTaskSubscriptionRequestFn != nil {
		return m.toProtoTaskSubscriptionRequestFn(req)
	}
	return pbconv.ToProtoTaskSubscriptionRequest(req)
}

func (m *mockTransportConverter) ToProtoListTasksRequest(req *a2a.ListTasksRequest) (*a2agopb.ListTasksRequest, error) {
	if m.toProtoListTasksRequestFn != nil {
		return m.toProtoListTasksRequestFn(req)
	}
	return pbconv.ToProtoListTasksRequest(req)
}

func (m *mockTransportConverter) ToProtoCreateTaskPushConfigRequest(req *a2a.CreateTaskPushConfigRequest) (*a2agopb.CreateTaskPushNotificationConfigRequest, error) {
	if m.toProtoCreateTaskPushConfigRequestFn != nil {
		return m.toProtoCreateTaskPushConfigRequestFn(req)
	}
	return pbconv.ToProtoCreateTaskPushConfigRequest(req)
}

func (m *mockTransportConverter) ToProtoGetTaskPushConfigRequest(req *a2a.GetTaskPushConfigRequest) (*a2agopb.GetTaskPushNotificationConfigRequest, error) {
	if m.toProtoGetTaskPushConfigRequestFn != nil {
		return m.toProtoGetTaskPushConfigRequestFn(req)
	}
	return pbconv.ToProtoGetTaskPushConfigRequest(req)
}

func (m *mockTransportConverter) ToProtoListTaskPushConfigRequest(req *a2a.ListTaskPushConfigRequest) (*a2agopb.ListTaskPushNotificationConfigRequest, error) {
	if m.toProtoListTaskPushConfigRequestFn != nil {
		return m.toProtoListTaskPushConfigRequestFn(req)
	}
	return pbconv.ToProtoListTaskPushConfigRequest(req)
}

func (m *mockTransportConverter) ToProtoDeleteTaskPushConfigRequest(req *a2a.DeleteTaskPushConfigRequest) (*a2agopb.DeleteTaskPushNotificationConfigRequest, error) {
	if m.toProtoDeleteTaskPushConfigRequestFn != nil {
		return m.toProtoDeleteTaskPushConfigRequestFn(req)
	}
	return pbconv.ToProtoDeleteTaskPushConfigRequest(req)
}

func (m *mockTransportConverter) FromProtoSendMessageResponse(resp *a2agopb.SendMessageResponse) (a2a.SendMessageResult, error) {
	if m.fromProtoSendMessageResponseFn != nil {
		return m.fromProtoSendMessageResponseFn(resp)
	}
	return pbconv.FromProtoSendMessageResponse(resp)
}

func (m *mockTransportConverter) FromProtoStreamResponse(resp *a2agopb.StreamResponse) (a2a.Event, error) {
	if m.fromProtoStreamResponseFn != nil {
		return m.fromProtoStreamResponseFn(resp)
	}
	return pbconv.FromProtoStreamResponse(resp)
}

func (m *mockTransportConverter) FromProtoTask(resp *a2agopb.Task) (*a2a.Task, error) {
	if m.fromProtoTaskFn != nil {
		return m.fromProtoTaskFn(resp)
	}
	return pbconv.FromProtoTask(resp)
}

func (m *mockTransportConverter) FromProtoListTasksResponse(resp *a2agopb.ListTasksResponse) (*a2a.ListTasksResponse, error) {
	if m.fromProtoListTasksResponseFn != nil {
		return m.fromProtoListTasksResponseFn(resp)
	}
	return pbconv.FromProtoListTasksResponse(resp)
}

func (m *mockTransportConverter) FromProtoTaskPushConfig(resp *a2agopb.TaskPushNotificationConfig) (*a2a.TaskPushConfig, error) {
	if m.fromProtoTaskPushConfigFn != nil {
		return m.fromProtoTaskPushConfigFn(resp)
	}
	return pbconv.FromProtoTaskPushConfig(resp)
}

func (m *mockTransportConverter) FromProtoListTaskPushConfigResponse(resp *a2agopb.ListTaskPushNotificationConfigResponse) (*a2a.ListTaskPushConfigResponse, error) {
	if m.fromProtoListTaskPushConfigResponseFn != nil {
		return m.fromProtoListTaskPushConfigResponseFn(resp)
	}
	return pbconv.FromProtoListTaskPushConfigResponse(resp)
}

func (m *mockTransportConverter) FromProtoAgentCard(resp *a2agopb.AgentCard) (*a2a.AgentCard, error) {
	if m.fromProtoAgentCardFn != nil {
		return m.fromProtoAgentCardFn(resp)
	}
	return pbconv.FromProtoAgentCard(resp)
}

// TestTransport_ConverterErrors exercises every error branch in Transport by injecting
// a mockTransportConverter that returns an error for one specific method at a time.
func TestTransport_ConverterErrors(t *testing.T) {
	ctx := t.Context()
	convErr := errors.New("forced converter error")

	// minimalSrv returns minimal valid proto responses for "response decode error" sub-cases.
	minimalSrv := &mockProtoServer{
		sendMessageFn: func(_ context.Context, _ *a2agopb.SendMessageRequest) (*a2agopb.SendMessageResponse, error) {
			return &a2agopb.SendMessageResponse{
				Payload: &a2agopb.SendMessageResponse_Msg{
					Msg: &a2agopb.Message{MessageId: "resp", Role: a2agopb.Role_ROLE_AGENT},
				},
			}, nil
		},
		getTaskFn: func(_ context.Context, req *a2agopb.GetTaskRequest) (*a2agopb.Task, error) {
			id := "task-1"
			if req.GetName() != "" {
				id = req.GetName()[len("tasks/"):]
			}
			return &a2agopb.Task{Id: id, ContextId: "ctx", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED}}, nil
		},
		listTasksFn: func(_ context.Context, _ *a2agopb.ListTasksRequest) (*a2agopb.ListTasksResponse, error) {
			return &a2agopb.ListTasksResponse{}, nil
		},
		cancelTaskFn: func(_ context.Context, req *a2agopb.CancelTaskRequest) (*a2agopb.Task, error) {
			id := "task-1"
			if req.GetName() != "" {
				id = req.GetName()[len("tasks/"):]
			}
			return &a2agopb.Task{Id: id, Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_CANCELLED}}, nil
		},
		sendStreamingMessageFn: func(_ context.Context, _ *a2agopb.SendMessageRequest, stream slimrpc.RequestStream[*a2agopb.StreamResponse]) error {
			return stream.Send(&a2agopb.StreamResponse{
				Payload: &a2agopb.StreamResponse_Msg{
					Msg: &a2agopb.Message{MessageId: "resp", Role: a2agopb.Role_ROLE_AGENT},
				},
			})
		},
		taskSubscriptionFn: func(_ context.Context, _ *a2agopb.TaskSubscriptionRequest, stream slimrpc.RequestStream[*a2agopb.StreamResponse]) error {
			return stream.Send(&a2agopb.StreamResponse{
				Payload: &a2agopb.StreamResponse_Task{
					Task: &a2agopb.Task{Id: "task-1", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING}},
				},
			})
		},
		createTaskPushNotificationConfigFn: func(_ context.Context, req *a2agopb.CreateTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error) {
			return &a2agopb.TaskPushNotificationConfig{
				Name:                   req.GetParent() + "/pushNotificationConfigs/cfg-1",
				PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: "cfg-1"},
			}, nil
		},
		getTaskPushNotificationConfigFn: func(_ context.Context, req *a2agopb.GetTaskPushNotificationConfigRequest) (*a2agopb.TaskPushNotificationConfig, error) {
			return &a2agopb.TaskPushNotificationConfig{
				Name:                   req.GetName(),
				PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: "cfg-1"},
			}, nil
		},
		listTaskPushNotificationConfigFn: func(_ context.Context, _ *a2agopb.ListTaskPushNotificationConfigRequest) (*a2agopb.ListTaskPushNotificationConfigResponse, error) {
			return &a2agopb.ListTaskPushNotificationConfigResponse{}, nil
		},
		getAgentCardFn: func(_ context.Context, _ *a2agopb.GetAgentCardRequest) (*a2agopb.AgentCard, error) {
			return &a2agopb.AgentCard{Name: "test", ProtocolVersion: "0.3"}, nil
		},
	}

	mustStreamError := func(t *testing.T, seq iter.Seq2[a2a.Event, error]) {
		t.Helper()
		for _, err := range seq {
			if err != nil {
				return
			}
		}
	}

	t.Run("SendMessage/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoSendMessageRequestFn: func(*a2a.SendMessageRequest) (*a2agopb.SendMessageRequest, error) {
				return nil, convErr
			},
		}))
		_, err := transport.SendMessage(ctx, nil, &a2a.SendMessageRequest{
			Message: &a2a.Message{ID: "m1", Role: a2a.MessageRoleUser, Parts: a2a.ContentParts{a2a.NewTextPart("hi")}},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("SendMessage/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoSendMessageResponseFn: func(*a2agopb.SendMessageResponse) (a2a.SendMessageResult, error) {
				return nil, convErr
			},
		}))
		_, err := transport.SendMessage(ctx, nil, &a2a.SendMessageRequest{
			Message: &a2a.Message{ID: "m1", Role: a2a.MessageRoleUser, Parts: a2a.ContentParts{a2a.NewTextPart("hi")}},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("SendStreamingMessage/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoSendMessageRequestFn: func(*a2a.SendMessageRequest) (*a2agopb.SendMessageRequest, error) {
				return nil, convErr
			},
		}))
		seq := transport.SendStreamingMessage(ctx, nil, &a2a.SendMessageRequest{
			Message: &a2a.Message{ID: "m1", Role: a2a.MessageRoleUser, Parts: a2a.ContentParts{a2a.NewTextPart("hi")}},
		})
		mustStreamError(t, seq)
	})

	t.Run("SendStreamingMessage/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoStreamResponseFn: func(*a2agopb.StreamResponse) (a2a.Event, error) {
				return nil, convErr
			},
		}))
		seq := transport.SendStreamingMessage(ctx, nil, &a2a.SendMessageRequest{
			Message: &a2a.Message{ID: "m1", Role: a2a.MessageRoleUser, Parts: a2a.ContentParts{a2a.NewTextPart("hi")}},
		})
		mustStreamError(t, seq)
	})

	t.Run("GetTask/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoGetTaskRequestFn: func(*a2a.GetTaskRequest) (*a2agopb.GetTaskRequest, error) {
				return nil, convErr
			},
		}))
		_, err := transport.GetTask(ctx, nil, &a2a.GetTaskRequest{ID: "task-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetTask/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoTaskFn: func(*a2agopb.Task) (*a2a.Task, error) {
				return nil, convErr
			},
		}))
		_, err := transport.GetTask(ctx, nil, &a2a.GetTaskRequest{ID: "task-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ListTasks/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoListTasksRequestFn: func(*a2a.ListTasksRequest) (*a2agopb.ListTasksRequest, error) {
				return nil, convErr
			},
		}))
		_, err := transport.ListTasks(ctx, nil, &a2a.ListTasksRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ListTasks/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoListTasksResponseFn: func(*a2agopb.ListTasksResponse) (*a2a.ListTasksResponse, error) {
				return nil, convErr
			},
		}))
		_, err := transport.ListTasks(ctx, nil, &a2a.ListTasksRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CancelTask/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoCancelTaskRequestFn: func(*a2a.CancelTaskRequest) (*a2agopb.CancelTaskRequest, error) {
				return nil, convErr
			},
		}))
		_, err := transport.CancelTask(ctx, nil, &a2a.CancelTaskRequest{ID: "task-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CancelTask/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoTaskFn: func(*a2agopb.Task) (*a2a.Task, error) {
				return nil, convErr
			},
		}))
		_, err := transport.CancelTask(ctx, nil, &a2a.CancelTaskRequest{ID: "task-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("SubscribeToTask/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoTaskSubscriptionRequestFn: func(*a2a.SubscribeToTaskRequest) (*a2agopb.TaskSubscriptionRequest, error) {
				return nil, convErr
			},
		}))
		seq := transport.SubscribeToTask(ctx, nil, &a2a.SubscribeToTaskRequest{ID: "task-1"})
		mustStreamError(t, seq)
	})

	t.Run("SubscribeToTask/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoStreamResponseFn: func(*a2agopb.StreamResponse) (a2a.Event, error) {
				return nil, convErr
			},
		}))
		seq := transport.SubscribeToTask(ctx, nil, &a2a.SubscribeToTaskRequest{ID: "task-1"})
		mustStreamError(t, seq)
	})

	t.Run("GetTaskPushConfig/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoGetTaskPushConfigRequestFn: func(*a2a.GetTaskPushConfigRequest) (*a2agopb.GetTaskPushNotificationConfigRequest, error) {
				return nil, convErr
			},
		}))
		_, err := transport.GetTaskPushConfig(ctx, nil, &a2a.GetTaskPushConfigRequest{TaskID: "task-1", ID: "cfg-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetTaskPushConfig/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoTaskPushConfigFn: func(*a2agopb.TaskPushNotificationConfig) (*a2a.TaskPushConfig, error) {
				return nil, convErr
			},
		}))
		_, err := transport.GetTaskPushConfig(ctx, nil, &a2a.GetTaskPushConfigRequest{TaskID: "task-1", ID: "cfg-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateTaskPushConfig/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoCreateTaskPushConfigRequestFn: func(*a2a.CreateTaskPushConfigRequest) (*a2agopb.CreateTaskPushNotificationConfigRequest, error) {
				return nil, convErr
			},
		}))
		_, err := transport.CreateTaskPushConfig(ctx, nil, &a2a.CreateTaskPushConfigRequest{
			TaskID: "task-1", Config: a2a.PushConfig{ID: "cfg-1"},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateTaskPushConfig/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoTaskPushConfigFn: func(*a2agopb.TaskPushNotificationConfig) (*a2a.TaskPushConfig, error) {
				return nil, convErr
			},
		}))
		_, err := transport.CreateTaskPushConfig(ctx, nil, &a2a.CreateTaskPushConfigRequest{
			TaskID: "task-1", Config: a2a.PushConfig{ID: "cfg-1"},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ListTaskPushConfigs/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoListTaskPushConfigRequestFn: func(*a2a.ListTaskPushConfigRequest) (*a2agopb.ListTaskPushNotificationConfigRequest, error) {
				return nil, convErr
			},
		}))
		_, err := transport.ListTaskPushConfigs(ctx, nil, &a2a.ListTaskPushConfigRequest{TaskID: "task-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ListTaskPushConfigs/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoListTaskPushConfigResponseFn: func(*a2agopb.ListTaskPushNotificationConfigResponse) (*a2a.ListTaskPushConfigResponse, error) {
				return nil, convErr
			},
		}))
		_, err := transport.ListTaskPushConfigs(ctx, nil, &a2a.ListTaskPushConfigRequest{TaskID: "task-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("DeleteTaskPushConfig/request encode error", func(t *testing.T) {
		transport := startTestTransport(t, &mockProtoServer{}, withTransportConverter(&mockTransportConverter{
			toProtoDeleteTaskPushConfigRequestFn: func(*a2a.DeleteTaskPushConfigRequest) (*a2agopb.DeleteTaskPushNotificationConfigRequest, error) {
				return nil, convErr
			},
		}))
		err := transport.DeleteTaskPushConfig(ctx, nil, &a2a.DeleteTaskPushConfigRequest{TaskID: "task-1", ID: "cfg-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetExtendedAgentCard/response decode error", func(t *testing.T) {
		transport := startTestTransport(t, minimalSrv, withTransportConverter(&mockTransportConverter{
			fromProtoAgentCardFn: func(*a2agopb.AgentCard) (*a2a.AgentCard, error) {
				return nil, convErr
			},
		}))
		_, err := transport.GetExtendedAgentCard(ctx, nil, &a2a.GetExtendedAgentCardRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
