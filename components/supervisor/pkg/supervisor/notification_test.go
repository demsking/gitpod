package supervisor

import (
	"context"
	"testing"

	"github.com/gitpod-io/gitpod/supervisor/api"
	"google.golang.org/grpc"
)

var ctx = context.Background()

type TestNotificationService_SubscribeServer struct {
	resps chan *api.SubscribeResult
	grpc.ServerStream
}

func (subscribeServer *TestNotificationService_SubscribeServer) Send(resp *api.SubscribeResult) error {
	subscribeServer.resps <- resp
	return nil
}

func (subscribeServer *TestNotificationService_SubscribeServer) Context() context.Context {
	return ctx
}

func Test(t *testing.T) {
	t.Run("Test happy path", func(t *testing.T) {
		notificationService := NewNotificationService()
		subscribeServer := &TestNotificationService_SubscribeServer{
			resps: make(chan *api.SubscribeResult),
		}
		go func() {
			notification := <-subscribeServer.resps
			notificationService.Respond(ctx, &api.RespondRequest{
				RequestId: notification.RequestId,
				Response: &api.NotifyResponse{
					Action: notification.Request.Actions[0],
				},
			})
		}()
		notificationService.Subscribe(&api.SubscribeRequest{}, subscribeServer)
		notifyResponse, _ := notificationService.Notify(ctx, &api.NotifyRequest{
			Title:   "Alert",
			Message: "Do you like this test?",
			Actions: []string{"yes", "no", "cancel"},
		})
		if notifyResponse.Action != "yes" {
			t.Errorf("Expected response 'yes' but was '%s'", notifyResponse.Action)
		}
	})
}
