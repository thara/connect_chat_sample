package main

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/thara/connect_chat_sample/gen/proto/chat/v1/chatv1connect"
)

const address = "localhost:8080"

func main() {
	mux := http.NewServeMux()
	path, handler := chatv1connect.NewChatServiceHandler(&chatServiceHandler{
		room: newRoom(context.Background()),
	})
	mux.Handle(path, handler)
	fmt.Println("... Listening on", address)

	http.ListenAndServe(address, h2c.NewHandler(mux, &http2.Server{}))
}
