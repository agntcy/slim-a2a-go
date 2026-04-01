// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2aslimrpc

import (
	a2agopb "github.com/a2aproject/a2a-go/a2apb"
	a2a "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2apb/v0/pbconv"
)

// converter abstracts all pbconv calls made by both Handler and Transport.
// The default implementation delegates to the real pbconv package.
// Tests can inject a mock to exercise error-handling branches.
//
// v0 differences from v1:
//   - ExtractTaskID replaces FromProtoCancelTaskRequest and FromProtoSubscribeToTaskRequest
//   - No FromProtoGetExtendedAgentCardRequest (GetAgentCard ignores request body)
//   - ToProtoTaskSubscriptionRequest replaces ToProtoSubscribeToTaskRequest
//   - No ToProtoGetExtendedAgentCardRequest (GetAgentCard sends no request body)
type converter interface {
	// Inbound (proto → domain): used by Handler to decode requests, Transport to decode responses.
	FromProtoSendMessageRequest(*a2agopb.SendMessageRequest) (*a2a.SendMessageRequest, error)
	FromProtoGetTaskRequest(*a2agopb.GetTaskRequest) (*a2a.GetTaskRequest, error)
	ExtractTaskID(name string) (a2a.TaskID, error)
	FromProtoListTasksRequest(*a2agopb.ListTasksRequest) (*a2a.ListTasksRequest, error)
	FromProtoCreateTaskPushConfigRequest(*a2agopb.CreateTaskPushNotificationConfigRequest) (*a2a.CreateTaskPushConfigRequest, error)
	FromProtoGetTaskPushConfigRequest(*a2agopb.GetTaskPushNotificationConfigRequest) (*a2a.GetTaskPushConfigRequest, error)
	FromProtoListTaskPushConfigRequest(*a2agopb.ListTaskPushNotificationConfigRequest) (*a2a.ListTaskPushConfigRequest, error)
	FromProtoDeleteTaskPushConfigRequest(*a2agopb.DeleteTaskPushNotificationConfigRequest) (*a2a.DeleteTaskPushConfigRequest, error)
	FromProtoSendMessageResponse(*a2agopb.SendMessageResponse) (a2a.SendMessageResult, error)
	FromProtoStreamResponse(*a2agopb.StreamResponse) (a2a.Event, error)
	FromProtoTask(*a2agopb.Task) (*a2a.Task, error)
	FromProtoListTasksResponse(*a2agopb.ListTasksResponse) (*a2a.ListTasksResponse, error)
	FromProtoTaskPushConfig(*a2agopb.TaskPushNotificationConfig) (*a2a.TaskPushConfig, error)
	FromProtoListTaskPushConfigResponse(*a2agopb.ListTaskPushNotificationConfigResponse) (*a2a.ListTaskPushConfigResponse, error)
	FromProtoAgentCard(*a2agopb.AgentCard) (*a2a.AgentCard, error)
	// Outbound (domain → proto): used by Handler to encode responses, Transport to encode requests.
	ToProtoSendMessageRequest(*a2a.SendMessageRequest) (*a2agopb.SendMessageRequest, error)
	ToProtoGetTaskRequest(*a2a.GetTaskRequest) (*a2agopb.GetTaskRequest, error)
	ToProtoCancelTaskRequest(*a2a.CancelTaskRequest) (*a2agopb.CancelTaskRequest, error)
	ToProtoTaskSubscriptionRequest(*a2a.SubscribeToTaskRequest) (*a2agopb.TaskSubscriptionRequest, error)
	ToProtoListTasksRequest(*a2a.ListTasksRequest) (*a2agopb.ListTasksRequest, error)
	ToProtoCreateTaskPushConfigRequest(*a2a.CreateTaskPushConfigRequest) (*a2agopb.CreateTaskPushNotificationConfigRequest, error)
	ToProtoGetTaskPushConfigRequest(*a2a.GetTaskPushConfigRequest) (*a2agopb.GetTaskPushNotificationConfigRequest, error)
	ToProtoListTaskPushConfigRequest(*a2a.ListTaskPushConfigRequest) (*a2agopb.ListTaskPushNotificationConfigRequest, error)
	ToProtoDeleteTaskPushConfigRequest(*a2a.DeleteTaskPushConfigRequest) (*a2agopb.DeleteTaskPushNotificationConfigRequest, error)
	ToProtoSendMessageResponse(a2a.SendMessageResult) (*a2agopb.SendMessageResponse, error)
	ToProtoStreamResponse(a2a.Event) (*a2agopb.StreamResponse, error)
	ToProtoTask(*a2a.Task) (*a2agopb.Task, error)
	ToProtoListTasksResponse(*a2a.ListTasksResponse) (*a2agopb.ListTasksResponse, error)
	ToProtoTaskPushConfig(*a2a.TaskPushConfig) (*a2agopb.TaskPushNotificationConfig, error)
	ToProtoListTaskPushConfigResponse(*a2a.ListTaskPushConfigResponse) (*a2agopb.ListTaskPushNotificationConfigResponse, error)
	ToProtoAgentCard(*a2a.AgentCard) (*a2agopb.AgentCard, error)
}

// defaultConverter delegates every method to the real pbconv package.
type defaultConverter struct{}

func (defaultConverter) FromProtoSendMessageRequest(req *a2agopb.SendMessageRequest) (*a2a.SendMessageRequest, error) {
	return pbconv.FromProtoSendMessageRequest(req)
}

func (defaultConverter) FromProtoGetTaskRequest(req *a2agopb.GetTaskRequest) (*a2a.GetTaskRequest, error) {
	return pbconv.FromProtoGetTaskRequest(req)
}

func (defaultConverter) ExtractTaskID(name string) (a2a.TaskID, error) {
	return pbconv.ExtractTaskID(name)
}

func (defaultConverter) FromProtoListTasksRequest(req *a2agopb.ListTasksRequest) (*a2a.ListTasksRequest, error) {
	return pbconv.FromProtoListTasksRequest(req)
}

func (defaultConverter) FromProtoCreateTaskPushConfigRequest(req *a2agopb.CreateTaskPushNotificationConfigRequest) (*a2a.CreateTaskPushConfigRequest, error) {
	return pbconv.FromProtoCreateTaskPushConfigRequest(req)
}

func (defaultConverter) FromProtoGetTaskPushConfigRequest(req *a2agopb.GetTaskPushNotificationConfigRequest) (*a2a.GetTaskPushConfigRequest, error) {
	return pbconv.FromProtoGetTaskPushConfigRequest(req)
}

func (defaultConverter) FromProtoListTaskPushConfigRequest(req *a2agopb.ListTaskPushNotificationConfigRequest) (*a2a.ListTaskPushConfigRequest, error) {
	return pbconv.FromProtoListTaskPushConfigRequest(req)
}

func (defaultConverter) FromProtoDeleteTaskPushConfigRequest(req *a2agopb.DeleteTaskPushNotificationConfigRequest) (*a2a.DeleteTaskPushConfigRequest, error) {
	return pbconv.FromProtoDeleteTaskPushConfigRequest(req)
}

func (defaultConverter) FromProtoSendMessageResponse(resp *a2agopb.SendMessageResponse) (a2a.SendMessageResult, error) {
	return pbconv.FromProtoSendMessageResponse(resp)
}

func (defaultConverter) FromProtoStreamResponse(resp *a2agopb.StreamResponse) (a2a.Event, error) {
	return pbconv.FromProtoStreamResponse(resp)
}

func (defaultConverter) FromProtoTask(resp *a2agopb.Task) (*a2a.Task, error) {
	return pbconv.FromProtoTask(resp)
}

func (defaultConverter) FromProtoListTasksResponse(resp *a2agopb.ListTasksResponse) (*a2a.ListTasksResponse, error) {
	return pbconv.FromProtoListTasksResponse(resp)
}

func (defaultConverter) FromProtoTaskPushConfig(resp *a2agopb.TaskPushNotificationConfig) (*a2a.TaskPushConfig, error) {
	return pbconv.FromProtoTaskPushConfig(resp)
}

func (defaultConverter) FromProtoListTaskPushConfigResponse(resp *a2agopb.ListTaskPushNotificationConfigResponse) (*a2a.ListTaskPushConfigResponse, error) {
	return pbconv.FromProtoListTaskPushConfigResponse(resp)
}

func (defaultConverter) FromProtoAgentCard(resp *a2agopb.AgentCard) (*a2a.AgentCard, error) {
	return pbconv.FromProtoAgentCard(resp)
}

func (defaultConverter) ToProtoSendMessageRequest(req *a2a.SendMessageRequest) (*a2agopb.SendMessageRequest, error) {
	return pbconv.ToProtoSendMessageRequest(req)
}

func (defaultConverter) ToProtoGetTaskRequest(req *a2a.GetTaskRequest) (*a2agopb.GetTaskRequest, error) {
	return pbconv.ToProtoGetTaskRequest(req)
}

func (defaultConverter) ToProtoCancelTaskRequest(req *a2a.CancelTaskRequest) (*a2agopb.CancelTaskRequest, error) {
	return pbconv.ToProtoCancelTaskRequest(req)
}

func (defaultConverter) ToProtoTaskSubscriptionRequest(req *a2a.SubscribeToTaskRequest) (*a2agopb.TaskSubscriptionRequest, error) {
	return pbconv.ToProtoTaskSubscriptionRequest(req)
}

func (defaultConverter) ToProtoListTasksRequest(req *a2a.ListTasksRequest) (*a2agopb.ListTasksRequest, error) {
	return pbconv.ToProtoListTasksRequest(req)
}

func (defaultConverter) ToProtoCreateTaskPushConfigRequest(req *a2a.CreateTaskPushConfigRequest) (*a2agopb.CreateTaskPushNotificationConfigRequest, error) {
	return pbconv.ToProtoCreateTaskPushConfigRequest(req)
}

func (defaultConverter) ToProtoGetTaskPushConfigRequest(req *a2a.GetTaskPushConfigRequest) (*a2agopb.GetTaskPushNotificationConfigRequest, error) {
	return pbconv.ToProtoGetTaskPushConfigRequest(req)
}

func (defaultConverter) ToProtoListTaskPushConfigRequest(req *a2a.ListTaskPushConfigRequest) (*a2agopb.ListTaskPushNotificationConfigRequest, error) {
	return pbconv.ToProtoListTaskPushConfigRequest(req)
}

func (defaultConverter) ToProtoDeleteTaskPushConfigRequest(req *a2a.DeleteTaskPushConfigRequest) (*a2agopb.DeleteTaskPushNotificationConfigRequest, error) {
	return pbconv.ToProtoDeleteTaskPushConfigRequest(req)
}

func (defaultConverter) ToProtoSendMessageResponse(result a2a.SendMessageResult) (*a2agopb.SendMessageResponse, error) {
	return pbconv.ToProtoSendMessageResponse(result)
}

func (defaultConverter) ToProtoStreamResponse(event a2a.Event) (*a2agopb.StreamResponse, error) {
	return pbconv.ToProtoStreamResponse(event)
}

func (defaultConverter) ToProtoTask(task *a2a.Task) (*a2agopb.Task, error) {
	return pbconv.ToProtoTask(task)
}

func (defaultConverter) ToProtoListTasksResponse(resp *a2a.ListTasksResponse) (*a2agopb.ListTasksResponse, error) {
	return pbconv.ToProtoListTasksResponse(resp)
}

func (defaultConverter) ToProtoTaskPushConfig(cfg *a2a.TaskPushConfig) (*a2agopb.TaskPushNotificationConfig, error) {
	return pbconv.ToProtoTaskPushConfig(cfg)
}

func (defaultConverter) ToProtoListTaskPushConfigResponse(resp *a2a.ListTaskPushConfigResponse) (*a2agopb.ListTaskPushNotificationConfigResponse, error) {
	return pbconv.ToProtoListTaskPushConfigResponse(resp)
}

func (defaultConverter) ToProtoAgentCard(card *a2a.AgentCard) (*a2agopb.AgentCard, error) {
	return pbconv.ToProtoAgentCard(card)
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// withHandlerConverter injects a custom converter into Handler (unexported — test seam only).
func withHandlerConverter(c converter) HandlerOption {
	return func(h *Handler) { h.conv = c }
}

// TransportOption configures a Transport.
type TransportOption func(*Transport)

// withTransportConverter injects a custom converter into Transport (unexported — test seam only).
func withTransportConverter(c converter) TransportOption {
	return func(t *Transport) { t.conv = c }
}
