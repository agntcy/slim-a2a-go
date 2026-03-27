// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	a2agopb "github.com/a2aproject/a2a-go/a2apb"
	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2apb/v0/pbconv"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	slim_bindings "github.com/agntcy/slim-bindings-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb/v0"
)

func TestMain(m *testing.M) {
	tracingCfg := slim_bindings.NewTracingConfigWith("debug", false, false, []string{"slim=debug"})
	if err := slim_bindings.InitializeWithConfigs(
		slim_bindings.NewRuntimeConfig(),
		tracingCfg,
		[]slim_bindings.ServiceConfig{slim_bindings.NewServiceConfig()},
	); err != nil {
		fmt.Fprintf(os.Stderr, "slim init failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// startTestServer creates a new in-memory SLIM server registered with handler,
// and a paired client channel. Names are made unique per call using a UUID v4
// so concurrent or sequential tests never share a routing entry.
// The server is shut down and the channel destroyed via t.Cleanup.
func startTestServer(t *testing.T, handler a2asrv.RequestHandler) ourpb.A2AServiceClient {
	t.Helper()
	svc := slim_bindings.GetGlobalService()

	serverName := slim_bindings.NewName("agntcy", t.Name(), "srv")
	serverApp, err := svc.CreateAppWithSecret(serverName, "test-secret-for-slim-unit-tests-only")
	if err != nil {
		t.Fatalf("create server app: %v", err)
	}

	slimHandler := NewHandler(handler)
	server := slim_bindings.NewServer(serverApp, serverName)
	slimHandler.RegisterWith(server)
	go func() {
		if err := server.Serve(); err != nil {
			t.Logf("server.Serve() exited with error: %v", err)
		}
	}()

	// Give the server a moment to start and complete subscriptions before
	// the client tries to connect.
	time.Sleep(10 * time.Millisecond)

	clientName := slim_bindings.NewName("agntcy", t.Name(), "cli")
	clientApp, err := svc.CreateAppWithSecret(clientName, "test-secret-for-slim-unit-tests-only")
	if err != nil {
		t.Fatalf("create client app: %v", err)
	}

	channel := slim_bindings.NewChannel(clientApp, serverName)

	t.Cleanup(func() {
		channel.Destroy()
		server.Shutdown()
	})

	return ourpb.NewA2AServiceClient(channel)
}

// mockRequestHandler is a mock of a2asrv.RequestHandler.
type mockRequestHandler struct {
	tasks       map[a2a.TaskID]*a2a.Task
	pushConfigs map[a2a.TaskID]map[string]*a2a.TaskPushConfig

	// Fields to capture call parameters.
	capturedGetTaskRequest              *a2a.GetTaskRequest
	capturedListTasksRequest            *a2a.ListTasksRequest
	capturedCancelTaskRequest           *a2a.CancelTaskRequest
	capturedSendMessageRequest          *a2a.SendMessageRequest
	capturedSendMessageStreamRequest    *a2a.SendMessageRequest
	capturedSubscribeToTaskRequest      *a2a.SubscribeToTaskRequest
	capturedCreateTaskPushConfigRequest *a2a.CreateTaskPushConfigRequest
	capturedGetTaskPushConfigRequest    *a2a.GetTaskPushConfigRequest
	capturedListTaskPushConfigRequest   *a2a.ListTaskPushConfigRequest
	capturedDeleteTaskPushConfigRequest *a2a.DeleteTaskPushConfigRequest

	// Embed for forward-compatibility; override specific methods below.
	a2asrv.RequestHandler
	SendMessageFunc          func(ctx context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error)
	SendMessageStreamFunc    func(ctx context.Context, req *a2a.SendMessageRequest) iter.Seq2[a2a.Event, error]
	SubscribeToTaskFunc      func(ctx context.Context, req *a2a.SubscribeToTaskRequest) iter.Seq2[a2a.Event, error]
	getExtendedAgentCardFunc func(ctx context.Context, req *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error)
}

var _ a2asrv.RequestHandler = (*mockRequestHandler)(nil)

func (m *mockRequestHandler) GetTask(ctx context.Context, req *a2a.GetTaskRequest) (*a2a.Task, error) {
	m.capturedGetTaskRequest = req
	if task, ok := m.tasks[req.ID]; ok {
		if req.HistoryLength != nil && *req.HistoryLength > 0 {
			if len(task.History) > int(*req.HistoryLength) {
				task.History = task.History[len(task.History)-int(*req.HistoryLength):]
			}
		}
		return task, nil
	}
	return nil, fmt.Errorf("task not found, taskID: %s", req.ID)
}

func (m *mockRequestHandler) ListTasks(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	m.capturedListTasksRequest = req

	var tasks []*a2a.Task
	for _, task := range m.tasks {
		taskCopy := *task
		if req.ContextID != "" && req.ContextID != taskCopy.ContextID {
			continue
		}
		tasks = append(tasks, &taskCopy)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	return &a2a.ListTasksResponse{
		Tasks:         tasks,
		TotalSize:     len(tasks),
		NextPageToken: "",
	}, nil
}

func (m *mockRequestHandler) CancelTask(ctx context.Context, req *a2a.CancelTaskRequest) (*a2a.Task, error) {
	m.capturedCancelTaskRequest = req
	if task, ok := m.tasks[req.ID]; ok {
		task.Status = a2a.TaskStatus{State: a2a.TaskStateCanceled}
		m.tasks[req.ID] = task
		return task, nil
	}
	return nil, fmt.Errorf("task not found, taskID: %s", req.ID)
}

func (m *mockRequestHandler) SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error) {
	m.capturedSendMessageRequest = req
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, req)
	}
	return nil, errors.New("SendMessage not implemented")
}

func (m *mockRequestHandler) SendStreamingMessage(ctx context.Context, req *a2a.SendMessageRequest) iter.Seq2[a2a.Event, error] {
	m.capturedSendMessageStreamRequest = req
	if m.SendMessageStreamFunc != nil {
		return m.SendMessageStreamFunc(ctx, req)
	}
	return func(yield func(a2a.Event, error) bool) {
		yield(nil, errors.New("SendStreamingMessage not implemented"))
	}
}

func (m *mockRequestHandler) SubscribeToTask(ctx context.Context, req *a2a.SubscribeToTaskRequest) iter.Seq2[a2a.Event, error] {
	m.capturedSubscribeToTaskRequest = req
	if m.SubscribeToTaskFunc != nil {
		return m.SubscribeToTaskFunc(ctx, req)
	}
	return func(yield func(a2a.Event, error) bool) {
		yield(nil, errors.New("SubscribeToTask not implemented"))
	}
}

func (m *mockRequestHandler) CreateTaskPushConfig(ctx context.Context, req *a2a.CreateTaskPushConfigRequest) (*a2a.TaskPushConfig, error) {
	m.capturedCreateTaskPushConfigRequest = req
	if _, ok := m.tasks[req.TaskID]; ok {
		if _, ok := m.pushConfigs[req.TaskID]; !ok {
			m.pushConfigs[req.TaskID] = make(map[string]*a2a.TaskPushConfig)
		}
		taskPushConfig := &a2a.TaskPushConfig{TaskID: req.TaskID, Config: req.Config}
		m.pushConfigs[req.TaskID][req.Config.ID] = taskPushConfig
		return taskPushConfig, nil
	}
	return nil, fmt.Errorf("task for push config not found, taskID: %s", req.TaskID)
}

func (m *mockRequestHandler) GetTaskPushConfig(ctx context.Context, req *a2a.GetTaskPushConfigRequest) (*a2a.TaskPushConfig, error) {
	m.capturedGetTaskPushConfigRequest = req
	if _, ok := m.tasks[req.TaskID]; ok {
		if pushConfigs, ok := m.pushConfigs[req.TaskID]; ok {
			return pushConfigs[req.ID], nil
		}
		return nil, fmt.Errorf("push config not found, taskID: %s, configID: %s", req.TaskID, req.ID)
	}
	return nil, fmt.Errorf("task for push config not found, taskID: %s", req.TaskID)
}

func (m *mockRequestHandler) ListTaskPushConfigs(ctx context.Context, req *a2a.ListTaskPushConfigRequest) ([]*a2a.TaskPushConfig, error) {
	m.capturedListTaskPushConfigRequest = req
	if _, ok := m.tasks[req.TaskID]; ok {
		if pushConfigs, ok := m.pushConfigs[req.TaskID]; ok {
			var result []*a2a.TaskPushConfig
			for _, v := range pushConfigs {
				result = append(result, v)
			}
			return result, nil
		}
		return []*a2a.TaskPushConfig{}, nil
	}
	return []*a2a.TaskPushConfig{}, fmt.Errorf("task for push config not found, taskID: %s", req.TaskID)
}

func (m *mockRequestHandler) DeleteTaskPushConfig(ctx context.Context, req *a2a.DeleteTaskPushConfigRequest) error {
	m.capturedDeleteTaskPushConfigRequest = req
	if _, ok := m.tasks[req.TaskID]; ok {
		if pushConfigs, ok := m.pushConfigs[req.TaskID]; ok {
			if _, ok := pushConfigs[req.ID]; ok {
				delete(pushConfigs, req.ID)
				return nil
			}
		}
		return nil
	}
	return fmt.Errorf("task for push config not found, taskID: %s", req.TaskID)
}

func (m *mockRequestHandler) GetExtendedAgentCard(ctx context.Context, req *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error) {
	if m.getExtendedAgentCardFunc == nil {
		return nil, errors.New("no extended agent card producer configured")
	}
	return m.getExtendedAgentCardFunc(ctx, req)
}

// defaultSendMessageFn, defaultSendMessageStreamFn, defaultSubscribeToTaskFn are the
// default mock implementations used by SendMessage, SendStreamingMessage, and
// TaskSubscription tests respectively.

var defaultSendMessageFn = func(_ context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error) {
	if req == nil || req.Message == nil {
		return nil, errors.New("nil message")
	}
	if req.Message.ID == "handler-error" {
		return nil, errors.New("handler error")
	}
	taskID := req.Message.TaskID
	if taskID == "" {
		taskID = a2a.NewTaskID()
	}
	return &a2a.Message{
		ID:     fmt.Sprintf("%s-response", req.Message.ID),
		TaskID: taskID,
		Role:   a2a.MessageRoleAgent,
		Parts:  req.Message.Parts,
	}, nil
}

var defaultSendMessageStreamFn = func(_ context.Context, req *a2a.SendMessageRequest) iter.Seq2[a2a.Event, error] {
	if req == nil || req.Message == nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, errors.New("nil message"))
		}
	}
	if req.Message.ID == "handler-error" {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, errors.New("handler stream error"))
		}
	}
	taskID := req.Message.TaskID
	if taskID == "" {
		taskID = a2a.NewTaskID()
	}
	task := &a2a.Task{
		ID:        taskID,
		ContextID: req.Message.ContextID,
		Status:    a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}
	statusUpdate := a2a.NewStatusUpdateEvent(task, a2a.TaskStateWorking, nil)
	finalMessage := &a2a.Message{
		ID:     fmt.Sprintf("%s-response", req.Message.ID),
		TaskID: taskID,
		Role:   a2a.MessageRoleAgent,
		Parts:  req.Message.Parts,
	}
	events := []a2a.Event{task, statusUpdate, finalMessage}
	return func(yield func(a2a.Event, error) bool) {
		for _, e := range events {
			if !yield(e, nil) {
				return
			}
		}
	}
}

var defaultSubscribeToTaskFn = func(_ context.Context, req *a2a.SubscribeToTaskRequest) iter.Seq2[a2a.Event, error] {
	if req.ID == "handler-error" {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, errors.New("handler resubscribe error"))
		}
	}
	task := &a2a.Task{
		ID:        req.ID,
		ContextID: "resubscribe-context",
		Status:    a2a.TaskStatus{State: a2a.TaskStateWorking},
	}
	statusUpdate := a2a.NewStatusUpdateEvent(task, a2a.TaskStateCompleted, nil)
	events := []a2a.Event{task, statusUpdate}
	return func(yield func(a2a.Event, error) bool) {
		for _, e := range events {
			if !yield(e, nil) {
				return
			}
		}
	}
}

func TestHandler_GetTask(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("test-task")
	historyLen := int(10)
	mockHandler := &mockRequestHandler{
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context", Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted}},
		},
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name      string
		req       *a2agopb.GetTaskRequest
		want      *a2agopb.Task
		wantQuery *a2a.GetTaskRequest
		wantErr   bool
	}{
		{
			name: "success",
			req:  &a2agopb.GetTaskRequest{Name: fmt.Sprintf("tasks/%s", taskID)},
			want: &a2agopb.Task{
				Id:        string(taskID),
				ContextId: "test-context",
				Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED},
			},
			wantQuery: &a2a.GetTaskRequest{ID: taskID},
		},
		{
			name: "success with history",
			req:  &a2agopb.GetTaskRequest{Name: fmt.Sprintf("tasks/%s", taskID), HistoryLength: 10},
			want: &a2agopb.Task{
				Id:        string(taskID),
				ContextId: "test-context",
				Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED},
			},
			wantQuery: &a2a.GetTaskRequest{ID: taskID, HistoryLength: &historyLen},
		},
		{
			name:    "invalid name",
			req:     &a2agopb.GetTaskRequest{Name: "invalid/name"},
			wantErr: true,
		},
		{
			name:    "handler error",
			req:     &a2agopb.GetTaskRequest{Name: "tasks/handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedGetTaskRequest = nil
			resp, err := client.GetTask(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetTask() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetTask() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("GetTask() got = %v, want %v", resp, tt.want)
			}
			if tt.wantQuery != nil && !reflect.DeepEqual(mockHandler.capturedGetTaskRequest, tt.wantQuery) {
				t.Errorf("OnGetTask() query got = %v, want %v", mockHandler.capturedGetTaskRequest, tt.wantQuery)
			}
		})
	}
}

func TestHandler_ListTasks(t *testing.T) {
	ctx := t.Context()
	taskID1, taskID2, taskID3 := a2a.NewTaskID(), a2a.NewTaskID(), a2a.NewTaskID()
	mockHandler := &mockRequestHandler{
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID1: {ID: taskID1, ContextID: "test-context1", Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted}},
			taskID2: {ID: taskID2, ContextID: "test-context2", Status: a2a.TaskStatus{State: a2a.TaskStateCompleted}},
			taskID3: {ID: taskID3, ContextID: "test-context1", Status: a2a.TaskStatus{State: a2a.TaskStateWorking}},
		},
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name       string
		req        *a2agopb.ListTasksRequest
		want       *a2agopb.ListTasksResponse
		wantParams *a2a.ListTasksRequest
		wantErr    bool
	}{
		{
			name: "success",
			req:  &a2agopb.ListTasksRequest{},
			want: &a2agopb.ListTasksResponse{
				Tasks: []*a2agopb.Task{
					{Id: string(taskID1), ContextId: "test-context1", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED}},
					{Id: string(taskID2), ContextId: "test-context2", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_COMPLETED}},
					{Id: string(taskID3), ContextId: "test-context1", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING}},
				},
				TotalSize:     3,
				NextPageToken: "",
			},
			wantParams: &a2a.ListTasksRequest{
				Status: a2a.TaskStateUnspecified,
			},
		},
		{
			name: "success with context filter",
			req: &a2agopb.ListTasksRequest{
				ContextId: "test-context1",
			},
			want: &a2agopb.ListTasksResponse{
				Tasks: []*a2agopb.Task{
					{Id: string(taskID1), ContextId: "test-context1", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED}},
					{Id: string(taskID3), ContextId: "test-context1", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING}},
				},
				TotalSize:     2,
				NextPageToken: "",
			},
			wantParams: &a2a.ListTasksRequest{
				ContextID: "test-context1",
				Status:    a2a.TaskStateUnspecified,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedListTasksRequest = nil
			resp, err := client.ListTasks(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ListTasks() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ListTasks() got unexpected error: %v", err)
			}
			// Sort both sides to make comparison order-independent.
			sort.Slice(resp.Tasks, func(i, j int) bool { return resp.Tasks[i].Id < resp.Tasks[j].Id })
			sort.Slice(tt.want.Tasks, func(i, j int) bool { return tt.want.Tasks[i].Id < tt.want.Tasks[j].Id })
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("ListTasks() got = %v, want %v", resp, tt.want)
			}
			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedListTasksRequest, tt.wantParams) {
				t.Errorf("OnListTasks() query got = %v, want %v", mockHandler.capturedListTasksRequest, tt.wantParams)
			}
		})
	}
}

func TestHandler_CancelTask(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("test-task")
	mockHandler := &mockRequestHandler{
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context"},
		},
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name       string
		req        *a2agopb.CancelTaskRequest
		want       *a2agopb.Task
		wantParams *a2a.CancelTaskRequest
		wantErr    bool
	}{
		{
			name:       "success",
			req:        &a2agopb.CancelTaskRequest{Name: fmt.Sprintf("tasks/%s", taskID)},
			want:       &a2agopb.Task{Id: string(taskID), ContextId: "test-context", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_CANCELLED}},
			wantParams: &a2a.CancelTaskRequest{ID: taskID},
		},
		{
			name:    "invalid name",
			req:     &a2agopb.CancelTaskRequest{Name: "invalid/name"},
			wantErr: true,
		},
		{
			name:    "handler error",
			req:     &a2agopb.CancelTaskRequest{Name: "tasks/handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedCancelTaskRequest = nil
			resp, err := client.CancelTask(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CancelTask() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CancelTask() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("CancelTask() got = %v, want %v", resp, tt.want)
			}
			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedCancelTaskRequest, tt.wantParams) {
				t.Errorf("OnCancelTask() params got = %v, want %v", mockHandler.capturedCancelTaskRequest, tt.wantParams)
			}
		})
	}
}

func TestHandler_SendMessage(t *testing.T) {
	ctx := t.Context()
	mockHandler := &mockRequestHandler{
		SendMessageFunc: defaultSendMessageFn,
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name       string
		req        *a2agopb.SendMessageRequest
		want       *a2agopb.SendMessageResponse
		wantParams *a2a.SendMessageRequest
		wantErr    bool
	}{
		{
			name: "message sent successfully without config",
			req: &a2agopb.SendMessageRequest{
				Request: &a2agopb.Message{
					MessageId: "req-msg-123",
					TaskId:    "test-task-123",
					Role:      a2agopb.Role_ROLE_USER,
					Parts: []*a2agopb.Part{
						{Part: &a2agopb.Part_Text{Text: "Hello Agent"}},
					},
				},
			},
			want: &a2agopb.SendMessageResponse{
				Payload: &a2agopb.SendMessageResponse_Msg{
					Msg: &a2agopb.Message{
						MessageId: "req-msg-123-response",
						TaskId:    "test-task-123",
						Role:      a2agopb.Role_ROLE_AGENT,
						Parts: []*a2agopb.Part{
							{Part: &a2agopb.Part_Text{Text: "Hello Agent"}},
						},
					},
				},
			},
			wantParams: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:     "req-msg-123",
					TaskID: "test-task-123",
					Role:   a2a.MessageRoleUser,
					Parts:  a2a.ContentParts{a2a.NewTextPart("Hello Agent")},
				},
			},
		},
		{
			name:    "nil request message",
			req:     &a2agopb.SendMessageRequest{},
			wantErr: true,
		},
		{
			name: "invalid request",
			req: &a2agopb.SendMessageRequest{
				Request: &a2agopb.Message{
					Parts: []*a2agopb.Part{{Part: nil}},
				},
			},
			wantErr: true,
		},
		{
			name: "handler error",
			req: &a2agopb.SendMessageRequest{
				Request: &a2agopb.Message{MessageId: "handler-error"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedSendMessageRequest = nil
			resp, err := client.SendMessage(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SendMessage() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("SendMessage() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("SendMessage() got = %v, want %v", resp, tt.want)
			}
			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedSendMessageRequest, tt.wantParams) {
				t.Errorf("OnSendMessage() params got = %+v, want %+v", mockHandler.capturedSendMessageRequest, tt.wantParams)
			}
		})
	}
}

func TestHandler_SendStreamingMessage(t *testing.T) {
	ctx := t.Context()
	mockHandler := &mockRequestHandler{
		SendMessageStreamFunc: defaultSendMessageStreamFn,
	}
	client := startTestServer(t, mockHandler)

	taskID := a2a.TaskID("stream-task-123")
	msgID := "stream-req-1"
	contextID := "stream-context-abc"
	parts := []*a2agopb.Part{
		{Part: &a2agopb.Part_Text{Text: "streaming hello"}},
	}

	tests := []struct {
		name       string
		req        *a2agopb.SendMessageRequest
		want       []*a2agopb.StreamResponse
		wantParams *a2a.SendMessageRequest
		wantErr    bool
	}{
		{
			name: "success",
			req: &a2agopb.SendMessageRequest{
				Request: &a2agopb.Message{
					MessageId: msgID,
					TaskId:    string(taskID),
					ContextId: contextID,
					Parts:     parts,
					Role:      a2agopb.Role_ROLE_USER,
				},
			},
			want: []*a2agopb.StreamResponse{
				{
					Payload: &a2agopb.StreamResponse_Task{
						Task: &a2agopb.Task{
							Id:        string(taskID),
							ContextId: contextID,
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED},
						},
					},
				},
				{
					Payload: &a2agopb.StreamResponse_StatusUpdate{
						StatusUpdate: &a2agopb.TaskStatusUpdateEvent{
							TaskId:    string(taskID),
							ContextId: contextID,
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING},
						},
					},
				},
				{
					Payload: &a2agopb.StreamResponse_Msg{
						Msg: &a2agopb.Message{
							MessageId: fmt.Sprintf("%s-response", msgID),
							TaskId:    string(taskID),
							Role:      a2agopb.Role_ROLE_AGENT,
							Parts:     parts,
						},
					},
				},
			},
			wantParams: &a2a.SendMessageRequest{
				Message: &a2a.Message{
					ID:        msgID,
					TaskID:    taskID,
					ContextID: contextID,
					Role:      a2a.MessageRoleUser,
					Parts:     a2a.ContentParts{a2a.NewTextPart("streaming hello")},
				},
			},
		},
		{
			name:    "nil request message",
			req:     &a2agopb.SendMessageRequest{},
			wantErr: true,
		},
		{
			name: "invalid request",
			req: &a2agopb.SendMessageRequest{
				Request: &a2agopb.Message{
					Parts: []*a2agopb.Part{{Part: nil}},
				},
			},
			wantErr: true,
		},
		{
			name: "handler error",
			req: &a2agopb.SendMessageRequest{
				Request: &a2agopb.Message{MessageId: "handler-error"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedSendMessageStreamRequest = nil
			stream, err := client.SendStreamingMessage(ctx, tt.req)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("SendStreamingMessage() got unexpected setup error: %v", err)
			}

			var received []*a2agopb.StreamResponse
			for {
				resp, err := stream.Recv()
				if err != nil {
					if tt.wantErr {
						return // error propagated via Recv() — acceptable
					}
					t.Fatalf("stream.Recv() got unexpected error: %v", err)
				}
				if resp == nil {
					break
				}
				received = append(received, resp)
			}

			if tt.wantErr {
				// SLIM streaming: server errors end the stream silently (0 events).
				if len(received) == 0 {
					return
				}
				t.Fatalf("SendStreamingMessage() expected error or empty stream, got %d events", len(received))
			}

			if len(received) != len(tt.want) {
				t.Fatalf("SendStreamingMessage() received %d events, want %d", len(received), len(tt.want))
			}

			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedSendMessageStreamRequest, tt.wantParams) {
				t.Errorf("SendStreamingMessage() params got = %+v, want %+v", mockHandler.capturedSendMessageStreamRequest, tt.wantParams)
			}

			for i, wantResp := range tt.want {
				// Ignore timestamp in status-update events.
				if r, ok := received[i].GetPayload().(*a2agopb.StreamResponse_StatusUpdate); ok {
					if r.StatusUpdate.GetStatus() != nil {
						r.StatusUpdate.Status.Timestamp = nil
					}
				}
				if !proto.Equal(received[i], wantResp) {
					t.Errorf("SendStreamingMessage() event %d got = %v, want %v", i, received[i], wantResp)
				}
			}
		})
	}
}

func TestHandler_TaskSubscription(t *testing.T) {
	ctx := t.Context()
	mockHandler := &mockRequestHandler{
		SubscribeToTaskFunc: defaultSubscribeToTaskFn,
	}
	client := startTestServer(t, mockHandler)
	taskID := a2a.TaskID("resub-task-456")

	tests := []struct {
		name        string
		req         *a2agopb.TaskSubscriptionRequest
		want        []*a2agopb.StreamResponse
		wantRequest *a2a.SubscribeToTaskRequest
		wantErr     bool
	}{
		{
			name: "success",
			req:  &a2agopb.TaskSubscriptionRequest{Name: fmt.Sprintf("tasks/%s", taskID)},
			want: []*a2agopb.StreamResponse{
				{
					Payload: &a2agopb.StreamResponse_Task{
						Task: &a2agopb.Task{
							Id:        string(taskID),
							ContextId: "resubscribe-context",
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING},
						},
					},
				},
				{
					Payload: &a2agopb.StreamResponse_StatusUpdate{
						StatusUpdate: &a2agopb.TaskStatusUpdateEvent{
							TaskId:    string(taskID),
							ContextId: "resubscribe-context",
							Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_COMPLETED},
						},
					},
				},
			},
			wantRequest: &a2a.SubscribeToTaskRequest{ID: taskID},
		},
		{
			name:    "invalid name",
			req:     &a2agopb.TaskSubscriptionRequest{Name: "invalid/name"},
			wantErr: true,
		},
		{
			name:    "handler error",
			req:     &a2agopb.TaskSubscriptionRequest{Name: "tasks/handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedSubscribeToTaskRequest = nil
			stream, err := client.TaskSubscription(ctx, tt.req)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("TaskSubscription() got unexpected setup error: %v", err)
			}

			var received []*a2agopb.StreamResponse
			for {
				resp, err := stream.Recv()
				if err != nil {
					if tt.wantErr {
						return // error propagated via Recv() — acceptable
					}
					t.Fatalf("stream.Recv() got unexpected error: %v", err)
				}
				if resp == nil {
					break
				}
				received = append(received, resp)
			}

			if tt.wantErr {
				// SLIM streaming: server errors end the stream silently (0 events).
				if len(received) == 0 {
					return
				}
				t.Fatalf("TaskSubscription() expected error or empty stream, got %d events", len(received))
			}

			if len(received) != len(tt.want) {
				t.Fatalf("TaskSubscription() received %d events, want %d", len(received), len(tt.want))
			}

			if tt.wantRequest != nil && !reflect.DeepEqual(mockHandler.capturedSubscribeToTaskRequest, tt.wantRequest) {
				t.Errorf("OnResubscribeToTask() params got = %v, want %v", mockHandler.capturedSubscribeToTaskRequest, tt.wantRequest)
			}

			for i, wantResp := range tt.want {
				// Ignore timestamp in status-update events.
				if r, ok := received[i].GetPayload().(*a2agopb.StreamResponse_StatusUpdate); ok {
					if r.StatusUpdate.GetStatus() != nil {
						r.StatusUpdate.Status.Timestamp = nil
					}
				}
				if !proto.Equal(received[i], wantResp) {
					t.Errorf("TaskSubscription() event %d got = %v, want %v", i, received[i], wantResp)
				}
			}
		})
	}
}

func TestHandler_CreateTaskPushNotificationConfig(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("test-task")
	mockHandler := &mockRequestHandler{
		pushConfigs: make(map[a2a.TaskID]map[string]*a2a.TaskPushConfig),
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context"},
		},
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name        string
		req         *a2agopb.CreateTaskPushNotificationConfigRequest
		want        *a2agopb.TaskPushNotificationConfig
		wantRequest *a2a.CreateTaskPushConfigRequest
		wantErr     bool
	}{
		{
			name: "success",
			req: &a2agopb.CreateTaskPushNotificationConfigRequest{
				Parent: fmt.Sprintf("tasks/%s", taskID),
				Config: &a2agopb.TaskPushNotificationConfig{
					PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: "test-config"},
				},
			},
			want: &a2agopb.TaskPushNotificationConfig{
				Name:                   fmt.Sprintf("tasks/%s/pushNotificationConfigs/test-config", taskID),
				PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: "test-config"},
			},
			wantRequest: &a2a.CreateTaskPushConfigRequest{
				TaskID: taskID,
				Config: a2a.PushConfig{ID: "test-config"},
			},
		},
		{
			name:    "invalid request",
			req:     &a2agopb.CreateTaskPushNotificationConfigRequest{Parent: "invalid/parent"},
			wantErr: true,
		},
		{
			name: "handler error",
			req: &a2agopb.CreateTaskPushNotificationConfigRequest{
				Parent: "tasks/handler-error",
				Config: &a2agopb.TaskPushNotificationConfig{
					PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: "test-config"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedCreateTaskPushConfigRequest = nil
			resp, err := client.CreateTaskPushNotificationConfig(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CreateTaskPushNotificationConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateTaskPushNotificationConfig() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("CreateTaskPushNotificationConfig() got = %v, want %v", resp, tt.want)
			}
			if tt.wantRequest != nil && !reflect.DeepEqual(mockHandler.capturedCreateTaskPushConfigRequest, tt.wantRequest) {
				t.Errorf("OnCreateTaskPushNotificationConfig() request got = %v, want %v", mockHandler.capturedCreateTaskPushConfigRequest, tt.wantRequest)
			}
		})
	}
}

func TestHandler_GetTaskPushNotificationConfig(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("test-task")
	configID := "test-config"
	mockHandler := &mockRequestHandler{
		pushConfigs: map[a2a.TaskID]map[string]*a2a.TaskPushConfig{
			taskID: {
				configID: {TaskID: taskID, Config: a2a.PushConfig{ID: configID}},
			},
		},
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context"},
		},
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name       string
		req        *a2agopb.GetTaskPushNotificationConfigRequest
		want       *a2agopb.TaskPushNotificationConfig
		wantParams *a2a.GetTaskPushConfigRequest
		wantErr    bool
	}{
		{
			name: "success",
			req: &a2agopb.GetTaskPushNotificationConfigRequest{
				Name: fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s", taskID, configID),
			},
			want: &a2agopb.TaskPushNotificationConfig{
				Name:                   fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s", taskID, configID),
				PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: configID},
			},
			wantParams: &a2a.GetTaskPushConfigRequest{
				TaskID: taskID,
				ID:     configID,
			},
		},
		{
			name:    "invalid request",
			req:     &a2agopb.GetTaskPushNotificationConfigRequest{Name: "tasks/test-task/invalid/test-config"},
			wantErr: true,
		},
		{
			name:    "handler error",
			req:     &a2agopb.GetTaskPushNotificationConfigRequest{Name: "tasks/handler-error/pushNotificationConfigs/test-config"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedGetTaskPushConfigRequest = nil
			resp, err := client.GetTaskPushNotificationConfig(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetTaskPushNotificationConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetTaskPushNotificationConfig() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("GetTaskPushNotificationConfig() got = %v, want %v", resp, tt.want)
			}
			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedGetTaskPushConfigRequest, tt.wantParams) {
				t.Errorf("OnGetTaskPushConfig() request got = %v, want %v", mockHandler.capturedGetTaskPushConfigRequest, tt.wantParams)
			}
		})
	}
}

func TestHandler_ListTaskPushNotificationConfig(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("test-task")
	configID := "test-config"
	mockHandler := &mockRequestHandler{
		pushConfigs: map[a2a.TaskID]map[string]*a2a.TaskPushConfig{
			taskID: {
				fmt.Sprintf("%s-1", configID): {TaskID: taskID, Config: a2a.PushConfig{ID: fmt.Sprintf("%s-1", configID)}},
				fmt.Sprintf("%s-2", configID): {TaskID: taskID, Config: a2a.PushConfig{ID: fmt.Sprintf("%s-2", configID)}},
			},
		},
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context"},
		},
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name       string
		req        *a2agopb.ListTaskPushNotificationConfigRequest
		want       *a2agopb.ListTaskPushNotificationConfigResponse
		wantParams *a2a.ListTaskPushConfigRequest
		wantErr    bool
	}{
		{
			name: "success",
			req:  &a2agopb.ListTaskPushNotificationConfigRequest{Parent: fmt.Sprintf("tasks/%s", taskID)},
			want: &a2agopb.ListTaskPushNotificationConfigResponse{
				Configs: []*a2agopb.TaskPushNotificationConfig{
					{
						Name:                   fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s-1", taskID, configID),
						PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: fmt.Sprintf("%s-1", configID)},
					},
					{
						Name:                   fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s-2", taskID, configID),
						PushNotificationConfig: &a2agopb.PushNotificationConfig{Id: fmt.Sprintf("%s-2", configID)},
					},
				},
			},
			wantParams: &a2a.ListTaskPushConfigRequest{TaskID: taskID},
		},
		{
			name:    "invalid parent",
			req:     &a2agopb.ListTaskPushNotificationConfigRequest{Parent: "invalid/parent"},
			wantErr: true,
		},
		{
			name:    "handler error",
			req:     &a2agopb.ListTaskPushNotificationConfigRequest{Parent: "tasks/handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedListTaskPushConfigRequest = nil
			resp, err := client.ListTaskPushNotificationConfig(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ListTaskPushNotificationConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ListTaskPushNotificationConfig() got unexpected error: %v", err)
			}
			sort.Slice(resp.Configs, func(i, j int) bool { return resp.Configs[i].Name < resp.Configs[j].Name })
			sort.Slice(tt.want.Configs, func(i, j int) bool { return tt.want.Configs[i].Name < tt.want.Configs[j].Name })
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("ListTaskPushNotificationConfig() got = %v, want %v", resp, tt.want)
			}
			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedListTaskPushConfigRequest, tt.wantParams) {
				t.Errorf("OnListTaskPushConfigs() request got = %v, want %v", mockHandler.capturedListTaskPushConfigRequest, tt.wantParams)
			}
		})
	}
}

func TestHandler_DeleteTaskPushNotificationConfig(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("test-task")
	configID := "test-config"
	mockHandler := &mockRequestHandler{
		pushConfigs: map[a2a.TaskID]map[string]*a2a.TaskPushConfig{
			taskID: {
				configID: {TaskID: taskID, Config: a2a.PushConfig{ID: configID}},
			},
		},
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context"},
		},
	}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name       string
		req        *a2agopb.DeleteTaskPushNotificationConfigRequest
		want       *emptypb.Empty
		wantParams *a2a.DeleteTaskPushConfigRequest
		wantErr    bool
	}{
		{
			name: "success",
			req: &a2agopb.DeleteTaskPushNotificationConfigRequest{
				Name: fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s", taskID, configID),
			},
			want: &emptypb.Empty{},
			wantParams: &a2a.DeleteTaskPushConfigRequest{
				TaskID: taskID,
				ID:     configID,
			},
		},
		{
			name: "invalid request",
			req: &a2agopb.DeleteTaskPushNotificationConfigRequest{
				Name: fmt.Sprintf("tasks/%s/invalid/%s", taskID, configID),
			},
			wantErr: true,
		},
		{
			name: "handler error",
			req: &a2agopb.DeleteTaskPushNotificationConfigRequest{
				Name: fmt.Sprintf("tasks/handler-error/pushNotificationConfigs/%s", configID),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedDeleteTaskPushConfigRequest = nil
			resp, err := client.DeleteTaskPushNotificationConfig(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DeleteTaskPushNotificationConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteTaskPushNotificationConfig() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("DeleteTaskPushNotificationConfig() got = %v, want %v", resp, tt.want)
			}
			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedDeleteTaskPushConfigRequest, tt.wantParams) {
				t.Errorf("OnDeleteTaskPushConfig() request got = %v, want %v", mockHandler.capturedDeleteTaskPushConfigRequest, tt.wantParams)
			}
		})
	}
}

func TestHandler_GetAgentCard(t *testing.T) {
	ctx := t.Context()

	a2aCard := &a2a.AgentCard{
		Name:                "Test Agent",
		SupportedInterfaces: []*a2a.AgentInterface{{ProtocolVersion: "0.3"}},
	}
	pCard, err := pbconv.ToProtoAgentCard(a2aCard)
	if err != nil {
		t.Fatalf("failed to convert agent card for test setup: %v", err)
	}

	badCard := &a2a.AgentCard{
		Capabilities: a2a.AgentCapabilities{
			Extensions: []a2a.AgentExtension{{Params: map[string]any{"bad": func() {}}}},
		},
	}

	mockHandler := &mockRequestHandler{}
	client := startTestServer(t, mockHandler)

	tests := []struct {
		name    string
		cardFn  func(context.Context, *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error)
		want    *a2agopb.AgentCard
		wantErr bool
	}{
		{
			name: "success",
			cardFn: func(_ context.Context, _ *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error) {
				return a2aCard, nil
			},
			want: pCard,
		},
		{
			name:    "nil producer",
			cardFn:  nil, // mock returns error when func is nil
			wantErr: true,
		},
		{
			name: "producer returns nil card",
			cardFn: func(_ context.Context, _ *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error) {
				return nil, nil
			},
			want: &a2agopb.AgentCard{},
		},
		{
			name: "producer fails",
			cardFn: func(_ context.Context, _ *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error) {
				return badCard, nil // badCard has func() in metadata — proto conversion fails
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.getExtendedAgentCardFunc = tt.cardFn
			resp, err := client.GetAgentCard(ctx, &a2agopb.GetAgentCardRequest{})
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetAgentCard() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetAgentCard() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("GetAgentCard() got = %v, want %v", resp, tt.want)
			}
		})
	}
}
