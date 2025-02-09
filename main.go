package main

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/context"
	"io"
	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	"os"
)

type login_info struct {
	Host     string `json:"Host"`
	User     string `json:"User"`
	Password string `json:"Password"`
	TLSoff   bool   `json:"NoTLS"`
	StartTLS bool   `json:"StartTLS"`
}

// MessageBody is a message stanza that contains a body. It is normally used for
// chat messages.
type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}

//func serverName(host string) string {
//	return strings.Split(host, ":")[0]
//}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//temporary until db and login page exists
	loginJSONbytes, err := os.ReadFile("db/login.json")
	if err != nil {
		panic("Unable to read login.json - " + err.Error())
	}
	xmppConfig := login_info{}
	if err := json.Unmarshal(loginJSONbytes, &xmppConfig); err != nil {
		panic("Could not parse login.json - " + err.Error())
	}

	j, err := jid.Parse(xmppConfig.User)
	if err != nil {
		panic("Could not parse user - " + err.Error())
	}

	server := j.Domainpart()

	d := dial.Dialer{}

	conn, err := d.DialServer(ctx, "tcp", j, server)
	if err != nil {
		panic("Could not connect stage 1 - " + err.Error())
	}

	session, err := xmpp.NewSession(ctx, j.Domain(), j, conn, 0, xmpp.NewNegotiator(func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig {
		return xmpp.StreamConfig{
			Lang: "en",
			Features: []xmpp.StreamFeature{
				xmpp.BindResource(),
				xmpp.StartTLS(&tls.Config{
					ServerName: j.Domain().String(),
					MinVersion: tls.VersionTLS12,
				}),
				xmpp.SASL("", xmppConfig.Password, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
			},
			TeeIn:  nil,
			TeeOut: nil,
		}
	}))
	if err != nil {
		panic("Could not connect stage 2 - " + err.Error())
	}

	// Send initial presence to let the server know we want to receive messages.
	err = session.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		//return fmt.Errorf("Error sending initial presence: %w", err)
		panic("Error sending initial presence - " + err.Error())
	}

	_ = session.Serve(xmpp.HandlerFunc(func(tokenReadEncoder xmlstream.TokenReadEncoder, start *xml.StartElement) error {

		decoder := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), tokenReadEncoder))
		if _, err := decoder.Token(); err != nil {
			return err
		}

		// Ignore anything that's not a message. In a real system we'd want to at
		// least respond to IQs.
		if start.Name.Local != "message" {
			return nil
		}

		msg := MessageBody{}
		err = decoder.DecodeElement(&msg, start)
		if err != nil && err != io.EOF {
			fmt.Println("Error decoding element - " + err.Error())
			return nil
		}

		// Don'tokenReadEncoder reflect messages unless they are chat messages and actually have a
		// body.
		// In a real world situation we'd probably want to respond to IQs, at least.
		if msg.Body == "" || msg.Type != stanza.ChatMessage {
			return nil
		}

		reply := MessageBody{
			Message: stanza.Message{
				To: msg.From.Bare(),
			},
			Body: msg.Body,
		}

		fmt.Printf("Replying to message %q from %s with body %q", msg.ID, reply.To, reply.Body)
		err = tokenReadEncoder.Encode(reply)
		if err != nil {
			fmt.Printf("Error responding to message %q: %q", msg.ID, err)
		}

		return nil
	}))

}
