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
		subscriber := &TestNotificationService_SubscribeServer{
			resps: make(chan *api.SubscribeResult),
		}
		go func() {
			notification := <-subscriber.resps
			notificationService.Respond(ctx, &api.RespondRequest{
				RequestId: notification.RequestId,
				Response: &api.NotifyResponse{
					Action: notification.Request.Actions[0],
				},
			})
		}()
		notificationService.Subscribe(&api.SubscribeRequest{}, subscriber)
		notifyResponse, err := notificationService.Notify(ctx, &api.NotifyRequest{
			Title:   "Alert",
			Message: "Do you like this test?",
			Actions: []string{"yes", "no", "cancel"},
		})
		if err != nil {
			t.Errorf("error receiving user action %s", err)
		}
		if notifyResponse.Action != "yes" {
			t.Errorf("expected response 'yes' but was '%s'", notifyResponse.Action)
		}
	})

	t.Run("Notification without actions should return immediately", func(t *testing.T) {
		notificationService := NewNotificationService()
		notifyResponse, err := notificationService.Notify(ctx, &api.NotifyRequest{
			Title:   "FYI",
			Message: "Read this or not",
		})
		if err != nil {
			t.Errorf("error receiving response %s", err)
		}
		if len(notifyResponse.Action) > 0 {
			t.Errorf("expected no response but got %s", notifyResponse.Action)
		}
	})

	t.Run("Late subscriber and pending notifications", func(t *testing.T) {
		notificationService := NewNotificationService()

		// fire notification without any subscribers
		_, err := notificationService.Notify(ctx, &api.NotifyRequest{
			Title:   "First Message",
			Message: "Notification fired before subscription",
		})
		if err != nil {
			t.Errorf("error on notification %s", err)
		}

		// add a first subscriber
		receivedRequestChannel := make(chan *api.NotifyRequest)
		firstSubscriber := &TestNotificationService_SubscribeServer{
			resps: make(chan *api.SubscribeResult),
		}

		// verify that late subscriber consumes cached notification
		go func() {
			select {
			case subscriptionRequest, ok := <-firstSubscriber.resps:
				if !ok {
					t.Errorf("notification stream closed")
				}
				receivedRequestChannel <- subscriptionRequest.Request
			default:
				t.Errorf("late subscriber did not receive pending notification")
			}
		}()
		notificationService.Subscribe(&api.SubscribeRequest{}, firstSubscriber)
		<-receivedRequestChannel

		// add a second subscriber
		secondSubscriber := &TestNotificationService_SubscribeServer{
			resps: make(chan *api.SubscribeResult),
		}
		notificationService.Subscribe(&api.SubscribeRequest{}, secondSubscriber)

		// Second subscriber should only get second message
		go func() {
			<-firstSubscriber.resps
		}()
		go func() {
			subscriptionRequest, ok := <-secondSubscriber.resps
			if !ok {
				t.Errorf("notification stream closed")
			}
			receivedRequestChannel <- subscriptionRequest.Request
		}()
		notificationService.Notify(ctx, &api.NotifyRequest{
			Title:   "Second Message",
			Message: "",
		})
		request2 := <-receivedRequestChannel
		if request2.Title != "Second Message" {
			t.Errorf("late subscriber received processed notification")
		}
	})

	t.Run("Wrong action is rejected", func(t *testing.T) {
		notificationService := NewNotificationService()

		// fire notification without any subscribers
		go func() {
			_, err := notificationService.Notify(ctx, &api.NotifyRequest{
				Title:   "First Message",
				Message: "Notification with actions",
				Actions: []string{"ok"},
			})
			if err != nil {
				t.Errorf("error on notification %s", err)
			}
		}()

		// add a first subscriber
		subscriber := &TestNotificationService_SubscribeServer{
			resps: make(chan *api.SubscribeResult),
		}
		notificationService.Subscribe(&api.SubscribeRequest{}, subscriber)

		// receive notification
		subscriptionRequest, ok := <-subscriber.resps
		if !ok {
			t.Errorf("notification stream closed")
		}

		// invalid reponse
		_, err := notificationService.Respond(ctx, &api.RespondRequest{
			RequestId: subscriptionRequest.RequestId,
			Response: &api.NotifyResponse{
				Action: "invalid",
			},
		})
		if err == nil {
			t.Errorf("expected error on invalid response")
		}

		// valid reponse
		_, err = notificationService.Respond(ctx, &api.RespondRequest{
			RequestId: subscriptionRequest.RequestId,
			Response: &api.NotifyResponse{
				Action: "ok",
			},
		})
		if err != nil {
			t.Errorf("error on valid response: %s", err)
		}

		// stale reponse
		_, err = notificationService.Respond(ctx, &api.RespondRequest{
			RequestId: subscriptionRequest.RequestId,
			Response: &api.NotifyResponse{
				Action: "ok",
			},
		})
		if err == nil {
			t.Errorf("expected error on stale response")
		}
	})
}
