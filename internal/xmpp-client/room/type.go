package room

import (
	"sync"

	"pain.agency/oasis/internal/xmpp-client/message"
)

type Room struct {
	// This can just as easily be a channel or a database, just used for
	// simplicity's sake. Slices are very hard to do wrong.
	Messages []*message.MessageBody

	notifier *sync.Cond
}

func New() *Room {
	return &Room{
		notifier: sync.NewCond(&sync.Mutex{}),
	}
}

func (r *Room) Publish(msg *message.MessageBody) {
	r.Messages = append(r.Messages, msg)
	r.notifier.Broadcast()
}

// AwaitNewMessage returns a one-time-use channel that is closed when a new
// message is received by this room.
func (r *Room) AwaitNewMessage() <-chan struct{} {
	c := make(chan struct{}, 1)
	go func() {
		r.notifier.Wait()
		close(c)
	}()

	return c
}
