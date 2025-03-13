package xmppclient

import "pain.agency/oasis/internal/xmpp-client/room"

func (c *Client) GetRoomList() []string {
	c.userMapMtx.RLock()
	list := make([]string, len(c.userMap))
	for k := range c.userMap {
		list = append(list, k)
	}
	c.userMapMtx.RUnlock()

	return list
}

func (c *Client) GetRoom(id string) *room.Room {
	c.userMapMtx.RLock()
	defer c.userMapMtx.RUnlock()

	return c.userMap[id]
}

// same as room.AwaitNewMessage
func (c *Client) AwaitNewRoom() <-chan struct{} {
	notificationChannel := make(chan struct{}, 1)
	go func() {
		c.newRoomNotifier.Wait()
		close(notificationChannel)
	}()

	return notificationChannel
}
