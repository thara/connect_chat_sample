package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/bufbuild/connect-go"
	chatv1 "github.com/thara/connect_chat_sample/gen/proto/chat/v1"
	"github.com/thara/connect_chat_sample/gen/proto/chat/v1/chatv1connect"
)

type chatServiceHandler struct {
	chatv1connect.UnimplementedChatServiceHandler

	room *room
}

func (h *chatServiceHandler) Join(ctx context.Context, req *connect.Request[chatv1.JoinRequest]) (*connect.Response[chatv1.JoinResponse], error) {
	id := h.room.Join(req.Msg.GetName())
	return connect.NewResponse(&chatv1.JoinResponse{RecipientId: id}), nil
}

func (h *chatServiceHandler) Leave(ctx context.Context, req *connect.Request[chatv1.LeaveRequest]) (*connect.Response[chatv1.LeaveResponse], error) {
	h.room.Leave(req.Msg.GetRecipientId())
	return connect.NewResponse(&chatv1.LeaveResponse{}), nil
}

const clientIDHeader = "client-id"

var errClientIDRequired = errors.New("no token provided")

func (h *chatServiceHandler) Messages(ctx context.Context, stream *connect.BidiStream[chatv1.MessagesRequest, chatv1.MessagesResponse]) error {
	done := make(chan struct{})
	fail := make(chan error)

	id, ok := parseClientID(stream.RequestHeader().Get(clientIDHeader))
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, errClientIDRequired)
	}

	subscription := make(chan chatMessage)
	defer close(subscription)

	h.room.Subscribe(id, subscription)
	fmt.Println("subscribe by", id)

	go func() {
		for {
			in, err := stream.Receive()
			if err == io.EOF {
				done <- struct{}{}
				return
			} else if err != nil {
				fail <- err
				return
			}
			msg := message{recipientId: in.GetRecipientId(), text: in.GetMessage().GetText()}
			fmt.Println(msg)

			h.room.Post(msg)
		}
	}()

	go func() {
		for {
			latest := <-subscription

			msg := &chatv1.MessagesResponse{
				Recipient: &chatv1.Recipient{
					RecipientId: latest.message.recipientId,
					Name:        latest.name,
				},
				Message: &chatv1.Message{
					Text: latest.message.text,
				},
			}

			err := stream.Send(msg)
			if err != nil {
				fail <- err
			}
		}
	}()

	for {
		select {
		case <-done:
			h.room.Leave(id)
			return nil
		case err := <-fail:
			return err
		}
	}
}

func parseClientID(s string) (uint64, bool) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint64(n), true
}
