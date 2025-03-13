package message

import "mellium.im/xmpp/stanza"

type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}
