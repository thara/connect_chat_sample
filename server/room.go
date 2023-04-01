package main

import (
	"context"
	"math/rand"
	"time"
)

type message struct {
	recipientId uint64
	text        string
}

type room struct {
	messages chan message
	join     chan recipient
	leave    chan uint64

	subscription chan newSubscriber
}

type subscriber chan<- chatMessage

type recipient struct {
	id         uint64
	name       string
	subscriber subscriber
}

type newSubscriber struct {
	id         uint64
	subscriber subscriber
}

type chatMessage struct {
	message
	name string
}

func newRoom(ctx context.Context) *room {
	messages := make(chan message, 16)
	subscription := make(chan newSubscriber)

	join := make(chan recipient)
	leave := make(chan uint64)

	go func() {
		recipients := map[uint64]*recipient{}

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-messages:
				src, ok := recipients[msg.recipientId]
				if !ok {
					continue
				}
				cmsg := chatMessage{
					message: msg,
					name:    src.name,
				}
				for _, dst := range recipients {
					if dst.id == src.id {
						continue
					}
					dst.subscriber <- cmsg
				}
			case s := <-subscription:
				recipients[s.id].subscriber = s.subscriber
			case r := <-join:
				recipients[r.id] = &r
			case id := <-leave:
				delete(recipients, id)
			}
		}
	}()

	return &room{
		messages:     messages,
		join:         join,
		leave:        leave,
		subscription: subscription,
	}
}

func (r *room) Join(name string) uint64 {
	r2 := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := r2.Uint64()

	ch := make(chan<- chatMessage, 16)

	r.join <- recipient{id: id, name: name, subscriber: ch}

	return id
}

func (r *room) Leave(id uint64) {
	r.leave <- id
}

func (r *room) Post(m message) {
	r.messages <- m
}

func (r *room) Subscribe(id uint64, s subscriber) {
	r.subscription <- newSubscriber{id: id, subscriber: s}
}
