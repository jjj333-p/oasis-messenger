package main

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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

type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}
type xmppMsg struct {
	body      MessageBody
	raw       string
	uiElement fyne.CanvasObject
}
type msgCache struct {
	//cache map[string][]MessageBody
	cache  []xmppMsg
	window fyne.Window
}

// deal with this later
// only render messages when front
func (cash msgCache) isFront() bool {
	return true
}

func (cash msgCache) add(body MessageBody) {
	//cash.cache = append(cash.cache, msg)
	msg := xmppMsg{body: body}
	//if chat is open, render the element
	if cash.isFront() {
		fEL := widget.NewLabel(body.From.String())
		bEL := widget.NewLabel(body.Body)
		EL := container.NewVBox(fEL, bEL)
		msg.uiElement = EL
	}
}

// MessageBody is a message stanza that contains a body. It is normally used for
// chat messages.

//func serverName(host string) string {
//	return strings.Split(host, ":")[0]
//}

type msgListener func(tokenReadEncoder xmlstream.TokenReadEncoder, start *xml.StartElement) error

func main() {

	messages := make(map[string]msgCache)
	//cache := msgCache{make(map[string][]xmppMsg)}

	err := initXMPP(&messages)
	if err != nil {
		panic(err)
	}

	a := app.New()
	w := a.NewWindow("Hello World")

	h := messages["jjj333@pain.agency"]
	msgElements := make([]fyne.CanvasObject, 50)
	w.SetContent(container.NewScroll(container.NewVBox()))

	w.ShowAndRun()
	for _, m := range h.cache {

		if m.uiElement == nil {
			fEL := widget.NewLabel(m.body.From.String())
			bEL := widget.NewLabel(m.body.Body)
			EL := container.NewVBox(fEL, bEL)
			m.uiElement = EL
		}

		msgElements = append(msgElements, m.uiElement)
	}
}

func initXMPP(messages *map[string]msgCache) error {
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

	_ = session.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		return onMSG(t, start, messages)
	}))

	return nil
}

func onMSG(tokenReadEncoder xmlstream.TokenReadEncoder, start *xml.StartElement, messages *map[string]msgCache) error {

	decoder := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(*start), tokenReadEncoder))
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Ignore anything that's not a message. In a real system we'd want to at
	// least respond to IQs.
	if start.Name.Local != "message" {
		return nil
	}

	body := MessageBody{}
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

	(*messages)[body.From.String()].add(body)
	//messages.cache[body.From.String()] = append(messages.cache[body.From.String()], message)

	return nil
}
