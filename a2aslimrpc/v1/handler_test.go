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

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2agopb "github.com/a2aproject/a2a-go/v2/a2apb/v1"
	"github.com/a2aproject/a2a-go/v2/a2apb/v1/pbconv"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	slim_bindings "github.com/agntcy/slim-bindings-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	ourpb "github.com/agntcy/slim-a2a-go/a2apb/v1"
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
// SubscribeToTask tests respectively.

var defaultSendMessageFn = func(_ context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error) {
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
	badMetaTaskID := a2a.TaskID("bad-meta-task")
	historyLen := int(10)
	mockHandler := &mockRequestHandler{
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context", Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted}},
			badMetaTaskID: {
				ID: badMetaTaskID, ContextID: "test-context",
				Status:   a2a.TaskStatus{State: a2a.TaskStateSubmitted},
				Metadata: map[string]any{"bad": struct{}{}},
			},
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
			req:  &a2agopb.GetTaskRequest{Id: string(taskID)},
			want: &a2agopb.Task{
				Id:        string(taskID),
				ContextId: "test-context",
				Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED},
			},
			wantQuery: &a2a.GetTaskRequest{ID: taskID},
		},
		{
			name: "success with history",
			req:  &a2agopb.GetTaskRequest{Id: string(taskID), HistoryLength: proto.Int32(10)},
			want: &a2agopb.Task{
				Id:        string(taskID),
				ContextId: "test-context",
				Status:    &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED},
			},
			wantQuery: &a2a.GetTaskRequest{ID: taskID, HistoryLength: &historyLen},
		},
		{
			name:    "handler error",
			req:     &a2agopb.GetTaskRequest{Id: "handler-error"},
			wantErr: true,
		},
		{
			name:    "response conversion error",
			req:     &a2agopb.GetTaskRequest{Id: string(badMetaTaskID)},
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
				TotalSize: 3,
			},
			wantParams: &a2a.ListTasksRequest{},
		},
		{
			name: "success with context filter",
			req:  &a2agopb.ListTasksRequest{ContextId: "test-context1"},
			want: &a2agopb.ListTasksResponse{
				Tasks: []*a2agopb.Task{
					{Id: string(taskID1), ContextId: "test-context1", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_SUBMITTED}},
					{Id: string(taskID3), ContextId: "test-context1", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_WORKING}},
				},
				TotalSize: 2,
			},
			wantParams: &a2a.ListTasksRequest{ContextID: "test-context1"},
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
			sort.Slice(resp.Tasks, func(i, j int) bool { return resp.Tasks[i].Id < resp.Tasks[j].Id })
			sort.Slice(tt.want.Tasks, func(i, j int) bool { return tt.want.Tasks[i].Id < tt.want.Tasks[j].Id })
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("ListTasks() got = %v, want %v", resp, tt.want)
			}
			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedListTasksRequest, tt.wantParams) {
				t.Errorf("OnListTasks() params got = %v, want %v", mockHandler.capturedListTasksRequest, tt.wantParams)
			}
		})
	}

	t.Run("response conversion error", func(t *testing.T) {
		badMetaID := a2a.TaskID("bad-meta-list-task")
		badMock := &mockRequestHandler{
			tasks: map[a2a.TaskID]*a2a.Task{
				badMetaID: {
					ID: badMetaID, ContextID: "bad-ctx",
					Status:   a2a.TaskStatus{State: a2a.TaskStateSubmitted},
					Metadata: map[string]any{"bad": struct{}{}},
				},
			},
		}
		badClient := startTestServer(t, badMock)
		_, err := badClient.ListTasks(ctx, &a2agopb.ListTasksRequest{})
		if err == nil {
			t.Fatal("ListTasks() expected error from response conversion, got nil")
		}
	})
}

func TestHandler_CancelTask(t *testing.T) {
	ctx := t.Context()
	taskID := a2a.TaskID("test-task")
	badMetaTaskID := a2a.TaskID("bad-meta-task")
	mockHandler := &mockRequestHandler{
		tasks: map[a2a.TaskID]*a2a.Task{
			taskID: {ID: taskID, ContextID: "test-context"},
			badMetaTaskID: {
				ID: badMetaTaskID, ContextID: "test-context",
				Metadata: map[string]any{"bad": struct{}{}},
			},
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
			req:        &a2agopb.CancelTaskRequest{Id: string(taskID)},
			want:       &a2agopb.Task{Id: string(taskID), ContextId: "test-context", Status: &a2agopb.TaskStatus{State: a2agopb.TaskState_TASK_STATE_CANCELED}},
			wantParams: &a2a.CancelTaskRequest{ID: taskID},
		},
		{
			name:    "handler error",
			req:     &a2agopb.CancelTaskRequest{Id: "handler-error"},
			wantErr: true,
		},
		{
			name:    "response conversion error",
			req:     &a2agopb.CancelTaskRequest{Id: string(badMetaTaskID)},
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
				Message: &a2agopb.Message{
					MessageId: "req-msg-123",
					TaskId:    "test-task-123",
					Role:      a2agopb.Role_ROLE_USER,
					Parts: []*a2agopb.Part{
						{Content: &a2agopb.Part_Text{Text: "Hello Agent"}},
					},
				},
			},
			want: &a2agopb.SendMessageResponse{
				Payload: &a2agopb.SendMessageResponse_Message{
					Message: &a2agopb.Message{
						MessageId: "req-msg-123-response",
						TaskId:    "test-task-123",
						Role:      a2agopb.Role_ROLE_AGENT,
						Parts: []*a2agopb.Part{
							{Content: &a2agopb.Part_Text{Text: "Hello Agent"}},
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
				Message: &a2agopb.Message{
					Parts: []*a2agopb.Part{{Content: nil}},
				},
			},
			wantErr: true,
		},
		{
			name: "handler error",
			req: &a2agopb.SendMessageRequest{
				Message: &a2agopb.Message{
					MessageId: "handler-error",
					Role:      a2agopb.Role_ROLE_USER,
					Parts: []*a2agopb.Part{
						{Content: &a2agopb.Part_Text{Text: "Hello Agent"}},
					},
				},
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
				t.Errorf("OnSendMessage() params got = %v, want %v", mockHandler.capturedSendMessageRequest, tt.wantParams)
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
		{Content: &a2agopb.Part_Text{Text: "streaming hello"}},
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
				Message: &a2agopb.Message{
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
					Payload: &a2agopb.StreamResponse_Message{
						Message: &a2agopb.Message{
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
				Message: &a2agopb.Message{
					Parts: []*a2agopb.Part{{Content: nil}},
				},
			},
			wantErr: true,
		},
		{
			name: "handler error",
			req: &a2agopb.SendMessageRequest{
				Message: &a2agopb.Message{
					MessageId: "handler-error",
					Role:      a2agopb.Role_ROLE_USER,
					Parts:     parts,
				},
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

	t.Run("response conversion error", func(t *testing.T) {
		badStreamMock := &mockRequestHandler{
			SendMessageStreamFunc: func(_ context.Context, _ *a2a.SendMessageRequest) iter.Seq2[a2a.Event, error] {
				return func(yield func(a2a.Event, error) bool) {
					// Yield a Task with NaN metadata — ToProtoStreamResponse will fail.
					task := &a2a.Task{
						ID:       "conv-err",
						Status:   a2a.TaskStatus{State: a2a.TaskStateWorking},
						Metadata: map[string]any{"bad": struct{}{}},
					}
					yield(task, nil)
				}
			},
		}
		badClient := startTestServer(t, badStreamMock)
		stream, err := badClient.SendStreamingMessage(ctx, &a2agopb.SendMessageRequest{
			Message: &a2agopb.Message{
				MessageId: "conv-err-msg",
				Role:      a2agopb.Role_ROLE_USER,
				Parts:     []*a2agopb.Part{{Content: &a2agopb.Part_Text{Text: "trigger"}}},
			},
		})
		if err != nil {
			return // error at setup is acceptable
		}
		for {
			resp, err := stream.Recv()
			if err != nil || resp == nil {
				return // error or stream end — expected
			}
			t.Fatalf("expected stream to end with error, got response: %v", resp)
		}
	})
}

func TestHandler_SubscribeToTask(t *testing.T) {
	ctx := t.Context()
	mockHandler := &mockRequestHandler{
		SubscribeToTaskFunc: defaultSubscribeToTaskFn,
	}
	client := startTestServer(t, mockHandler)
	taskID := a2a.TaskID("resub-task-456")

	tests := []struct {
		name       string
		req        *a2agopb.SubscribeToTaskRequest
		want       []*a2agopb.StreamResponse
		wantParams *a2a.SubscribeToTaskRequest
		wantErr    bool
	}{
		{
			name: "success",
			req:  &a2agopb.SubscribeToTaskRequest{Id: string(taskID)},
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
			wantParams: &a2a.SubscribeToTaskRequest{ID: taskID},
		},
		{
			name:    "handler error",
			req:     &a2agopb.SubscribeToTaskRequest{Id: "handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedSubscribeToTaskRequest = nil
			stream, err := client.SubscribeToTask(ctx, tt.req)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("SubscribeToTask() got unexpected setup error: %v", err)
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
				t.Fatalf("SubscribeToTask() expected error or empty stream, got %d events", len(received))
			}

			if len(received) != len(tt.want) {
				t.Fatalf("SubscribeToTask() received %d events, want %d", len(received), len(tt.want))
			}

			if tt.wantParams != nil && !reflect.DeepEqual(mockHandler.capturedSubscribeToTaskRequest, tt.wantParams) {
				t.Errorf("OnResubscribeToTask() params got = %v, want %v", mockHandler.capturedSubscribeToTaskRequest, tt.wantParams)
			}

			for i, wantResp := range tt.want {
				// Ignore timestamp in status-update events.
				if r, ok := received[i].GetPayload().(*a2agopb.StreamResponse_StatusUpdate); ok {
					if r.StatusUpdate.GetStatus() != nil {
						r.StatusUpdate.Status.Timestamp = nil
					}
				}
				if !proto.Equal(received[i], wantResp) {
					t.Errorf("SubscribeToTask() event %d got = %v, want %v", i, received[i], wantResp)
				}
			}
		})
	}

	t.Run("response conversion error", func(t *testing.T) {
		badSubMock := &mockRequestHandler{
			SubscribeToTaskFunc: func(_ context.Context, _ *a2a.SubscribeToTaskRequest) iter.Seq2[a2a.Event, error] {
				return func(yield func(a2a.Event, error) bool) {
					// Yield a Task with NaN metadata — ToProtoStreamResponse will fail.
					task := &a2a.Task{
						ID:       "conv-err-sub",
						Status:   a2a.TaskStatus{State: a2a.TaskStateWorking},
						Metadata: map[string]any{"bad": struct{}{}},
					}
					yield(task, nil)
				}
			},
		}
		badClient := startTestServer(t, badSubMock)
		stream, err := badClient.SubscribeToTask(ctx, &a2agopb.SubscribeToTaskRequest{Id: "any-task"})
		if err != nil {
			return // error at setup is acceptable
		}
		for {
			resp, err := stream.Recv()
			if err != nil || resp == nil {
				return // error or stream end — expected
			}
			t.Fatalf("expected stream to end with error, got response: %v", resp)
		}
	})
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
		req         *a2agopb.TaskPushNotificationConfig
		want        *a2agopb.TaskPushNotificationConfig
		wantRequest *a2a.CreateTaskPushConfigRequest
		wantErr     bool
	}{
		{
			name: "success",
			req: &a2agopb.TaskPushNotificationConfig{
				TaskId: string(taskID),
				Id:     "test-config",
				Url:    "https://example.com",
			},
			want: &a2agopb.TaskPushNotificationConfig{
				TaskId: string(taskID),
				Id:     "test-config",
				Url:    "https://example.com",
			},
			wantRequest: &a2a.CreateTaskPushConfigRequest{
				TaskID: taskID,
				Config: a2a.PushConfig{ID: "test-config", URL: "https://example.com"},
			},
		},
		{
			name: "handler error",
			req: &a2agopb.TaskPushNotificationConfig{
				TaskId: "handler-error",
				Id:     "test-config",
				Url:    "https://example.com",
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
				TaskId: string(taskID),
				Id:     configID,
			},
			want: &a2agopb.TaskPushNotificationConfig{
				TaskId: string(taskID),
				Id:     configID,
			},
			wantParams: &a2a.GetTaskPushConfigRequest{
				TaskID: taskID,
				ID:     configID,
			},
		},
		{
			name:    "handler error",
			req:     &a2agopb.GetTaskPushNotificationConfigRequest{TaskId: "handler-error", Id: configID},
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

func TestHandler_ListTaskPushNotificationConfigs(t *testing.T) {
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
		req        *a2agopb.ListTaskPushNotificationConfigsRequest
		want       *a2agopb.ListTaskPushNotificationConfigsResponse
		wantParams *a2a.ListTaskPushConfigRequest
		wantErr    bool
	}{
		{
			name: "success",
			req:  &a2agopb.ListTaskPushNotificationConfigsRequest{TaskId: string(taskID)},
			want: &a2agopb.ListTaskPushNotificationConfigsResponse{
				Configs: []*a2agopb.TaskPushNotificationConfig{
					{TaskId: string(taskID), Id: fmt.Sprintf("%s-1", configID)},
					{TaskId: string(taskID), Id: fmt.Sprintf("%s-2", configID)},
				},
			},
			wantParams: &a2a.ListTaskPushConfigRequest{TaskID: taskID},
		},
		{
			name:    "handler error",
			req:     &a2agopb.ListTaskPushNotificationConfigsRequest{TaskId: "handler-error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.capturedListTaskPushConfigRequest = nil
			resp, err := client.ListTaskPushNotificationConfigs(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ListTaskPushNotificationConfigs() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ListTaskPushNotificationConfigs() got unexpected error: %v", err)
			}
			sort.Slice(resp.Configs, func(i, j int) bool { return resp.Configs[i].Id < resp.Configs[j].Id })
			sort.Slice(tt.want.Configs, func(i, j int) bool { return tt.want.Configs[i].Id < tt.want.Configs[j].Id })
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("ListTaskPushNotificationConfigs() got = %v, want %v", resp, tt.want)
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
				TaskId: string(taskID),
				Id:     configID,
			},
			want: &emptypb.Empty{},
			wantParams: &a2a.DeleteTaskPushConfigRequest{
				TaskID: taskID,
				ID:     configID,
			},
		},
		{
			name: "handler error",
			req: &a2agopb.DeleteTaskPushNotificationConfigRequest{
				TaskId: "handler-error",
				Id:     configID,
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

func TestHandler_GetExtendedAgentCard(t *testing.T) {
	ctx := t.Context()

	a2aCard := &a2a.AgentCard{Name: "Test Agent", SupportedInterfaces: []*a2a.AgentInterface{{ProtocolVersion: a2a.Version}}}
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
			resp, err := client.GetExtendedAgentCard(ctx, &a2agopb.GetExtendedAgentCardRequest{})
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetExtendedAgentCard() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetExtendedAgentCard() got unexpected error: %v", err)
			}
			if !proto.Equal(resp, tt.want) {
				t.Fatalf("GetExtendedAgentCard() got = %v, want %v", resp, tt.want)
			}
		})
	}
}
