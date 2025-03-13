package xmppclient

import (
	"encoding/xml"
	"fmt"
	"io"

	"pain.agency/oasis/internal/xmpp-client/message"
	"pain.agency/oasis/internal/xmpp-client/room"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

func (c *Client) handleSession(tokenReadEncoder xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	decoder := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), tokenReadEncoder))
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Ignore anything that's not a message. In a real system we'd want to at
	// least respond to IQs.
	if start.Name.Local != "message" {
		// go handle
		return nil
	}

	var body message.MessageBody
	err := decoder.DecodeElement(&body, start)
	if err != nil && err != io.EOF {
		fmt.Println("Error decoding element - " + err.Error())
		return nil
	}

	// Don'tokenReadEncoder reflect messages unless they are chat messages and actually have a
	// body.
	// In a real world situation we'd probably want to respond to IQs, at least.
	if body.Body == "" || body.Type != stanza.ChatMessage {
		return nil
	}

	// pass back message, creating new channel if not open
	fmt.Printf("%s: %s", body.From.Bare().String(), body.Body)

	// check if theres an open channel for that chat, if not create one
	c.userMapMtx.RLock()
	destinationRoom, roomExists := c.userMap[body.From.Bare().String()]
	c.userMapMtx.RUnlock()
	if !roomExists {
		destinationRoom = room.New()
		c.userMapMtx.Lock()
		c.userMap[body.From.Bare().String()] = destinationRoom
		c.userMapMtx.Unlock()
		c.newRoomNotifier.Broadcast()
	}

	destinationRoom.Publish(&body)

	return nil
}
