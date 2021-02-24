// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package supervisor

import (
	"context"
	"fmt"
	"sync"

	"github.com/gitpod-io/gitpod/common-go/log"
	"github.com/gitpod-io/gitpod/supervisor/api"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxPendingNotifications = 100
)

// NotificationService implements the notification service API
type NotificationService struct {
	mutex                sync.Mutex
	subscribers          []*api.NotificationService_SubscribeServer
	nextNotificationId   uint64
	pendingNotifications map[uint64]*pendingNotification
}

type pendingNotification struct {
	message         *api.SubscribeResult
	responseChannel chan api.NotifyResponse
}

// RegisterGRPC registers a gRPC service
func (srv *NotificationService) RegisterGRPC(s *grpc.Server) {
	api.RegisterNotificationServiceServer(s, srv)
}

// RegisterREST registers a REST service
func (srv *NotificationService) RegisterREST(mux *runtime.ServeMux, grpcEndpoint string) error {
	return api.RegisterNotificationServiceHandlerFromEndpoint(context.Background(), mux, grpcEndpoint, []grpc.DialOption{grpc.WithInsecure()})
}

// Sends a notification to the user
func (srv *NotificationService) Notify(ctx context.Context, req *api.NotifyRequest) (*api.NotifyResponse, error) {
	if srv.nextNotificationId >= maxPendingNotifications {
		return nil, status.Error(codes.Internal, "Max number of pending notifications exceeded")
	}
	srv.mutex.Lock()
	var (
		requestId = srv.nextNotificationId
		message   = &api.SubscribeResult{
			RequestId: requestId,
			Request:   req,
		}
		staleSubscribers = []int{}
	)
	srv.nextNotificationId++
	for i, subscriber := range srv.subscribers {
		// TODO: why do I have to dereference here?
		var err = (*subscriber).Send(message)
		if err != nil {
			staleSubscribers = append(staleSubscribers, i)
		}
	}
	srv.removeSubscribers(staleSubscribers)
	if len(req.Actions) == 0 {
		srv.mutex.Unlock()
		return &api.NotifyResponse{}, nil
	}
	var channel = make(chan api.NotifyResponse)
	srv.pendingNotifications[requestId] = &pendingNotification{
		message:         message,
		responseChannel: channel,
	}
	srv.mutex.Unlock()

	firstResponse, ok := <-channel
	if !ok {
		return nil, status.Error(codes.Internal, "Error reading user reaction on notification from channel")
	}
	return &firstResponse, nil
}

// subscribes to notifications that are sent to the supervisor
func (srv *NotificationService) Subscribe(req *api.SubscribeRequest, resp api.NotificationService_SubscribeServer) error {
	srv.mutex.Lock()
	defer srv.mutex.Unlock()
	for _, pending := range srv.pendingNotifications {
		var err = resp.Send(pending.message)
		if err != nil {
			return fmt.Errorf("Cannot subscribe new subscriber as sending pending notification failed. %w", err)
		}
	}
	srv.subscribers = append(srv.subscribers, &resp)
	return nil
}

// reports user actions as response to a notification request
func (srv *NotificationService) Respond(ctx context.Context, req *api.RespondRequest) (*api.RespondResult, error) {
	srv.mutex.Lock()
	defer srv.mutex.Unlock()
	var pending = srv.pendingNotifications[req.RequestId]
	if pending == nil {
		log.Log.Infof("Invalid or late response to notification %d", req.RequestId)
	} else {
		if isActionAllowed(req.Response.Action, pending.message.Request) {
			delete(srv.pendingNotifications, req.RequestId)
			pending.responseChannel <- *req.Response
		} else {
			log.Log.WithFields(map[string]interface{}{
				"Notification": pending.message,
				"Action":       req.Response.Action,
			}).Error("Invalid user action on notification")
		}
	}
	return &api.RespondResult{}, nil
}

func isActionAllowed(action string, req *api.NotifyRequest) bool {
	for _, allowedAction := range req.Actions {
		if allowedAction == action {
			return true
		}
	}
	return false
}

func (srv *NotificationService) removeSubscribers(indices []int) {
	if len(indices) > 0 {
		log.Log.Errorf("Unsubscribing %d stale subscribers", len(indices))
		var (
			newSubscribers = make([]*api.NotificationService_SubscribeServer, len(srv.subscribers)-len(indices))
			j              = 0
		)
		for i := range srv.subscribers {
			if j < len(indices) && i == indices[j] {
				j++
			} else {
				newSubscribers[i-j] = srv.subscribers[i]
			}
		}
		srv.subscribers = newSubscribers
	}
}
