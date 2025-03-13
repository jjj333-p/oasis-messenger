package main

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/net/context"
	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	xmppclient "pain.agency/oasis/internal/xmpp-client"
)

// existed when i had the xml decoding in a goroutine, didnt work because pointer deref
// type msgListener func(tokenReadEncoder xmlstream.TokenReadEncoder, start *xml.StartElement) error
func main() {
	msgBodyChansChan := make(chan MsgBodyChan)

	// run the sdk in its own goroutine
	cl, err := xmppclient.New()

	// create fyne applet
	a := app.New()
	w := a.NewWindow("Hello World")
	input := widget.NewEntry()
	input.SetPlaceHolder("Enter jid...")
	// msgElements := make([]fyne.CanvasObject, 50)
	msgvbox := container.NewVBox()
	scroll := container.NewScroll(msgvbox)
	w.SetContent(container.NewVBox(input, scroll))

	go func() {
		// whichever chat is open, chat selector not yet created so hardcoded to jjj333@pain.agency
		room := cl.GetRoom("jjj333@pain.agency")
		for room == nil {
			time.Sleep(1 * time.Second)
			room = cl.GetRoom("jjj333@pain.agency")
		}
		for {
			select {
			case <-cl.AwaitNewRoom():
				fmt.Println("yaay! we got a new room!")
			case <-room.AwaitNewMessage():
				msg := room.Messages[len(room.Messages)]
				fmt.Println(msg)

				fEL := widget.NewLabel(msg.From.Bare().String())
				bEL := widget.NewLabel(msg.Body)
				EL := container.NewVBox(fEL, bEL)

				msgvbox.Add(EL)
				scroll.Refresh()
			}

			// msgElements = append(msgElements, xmppMessage.uiElement)
			// xmppMessage <- c
		}
		//	go func() {
		//		for msgBody := range chatChan.channel {
		//			fmt.Printf("%s %s\n", chatChan.from, msgBody.Body)
		//
		//		}
		//	}()
		//}
	}()

	w.ShowAndRun()
}

//func handleEvent(
//	tokenReadEncoder xmlstream.TokenReadEncoder,
//	sta  rt *xml.StartElement,
//	msgBodyChansChan chan MsgBodyChan,
//	openMsgBodyChans map[string]MsgBodyChan,
//) {
//
//
//
//	//messageChan <- body
//
//	//(*messages)[body.From.String()].add(body)
////	return // nil
//}

// basically run the sdk in its own goroutine
func initXMPP(msgBodyChansChan chan MsgBodyChan) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// temporary until db and login page exists
	loginJSONbytes, err := os.ReadFile("db/login.json")
	if err != nil {
		panic("Unable to read login.json - " + err.Error())
	}
	xmppConfig := loginInfo{}
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
		// return fmt.Errorf("Error sending initial presence: %w", err)
		panic("Error sending initial presence - " + err.Error())
	}

	// map of open channels to send messages into
	openMsgBodyChans := make(map[string]MsgBodyChan)
	chanLock := sync.Mutex{}

	_ = session.Serve(
		xmpp.HandlerFunc(
			func(tokenReadEncoder xmlstream.TokenReadEncoder, start *xml.StartElement) error {
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

				// pass back message, creating new channel if not open
				fmt.Printf("%s: %s", body.From.Bare().String(), body.Body)

				// check if theres an open channel for that chat, if not create one
				chanLock.Lock()
				c, ok := openMsgBodyChans[body.From.Bare().String()]
				if !ok || c.channel == nil {
					newChan := MsgBodyChan{
						from:    body.From.Bare().String(),
						channel: make(chan xmppMsg),
					}
					openMsgBodyChans[body.From.Bare().String()] = newChan
					c = newChan
					msgBodyChansChan <- newChan
				}
				chanLock.Unlock()

				// create ui element for message
				fEL := widget.NewLabel(body.From.Bare().String())
				bEL := widget.NewLabel(body.Body)
				EL := container.NewVBox(fEL, bEL)

				// pass into channel
				c.channel <- xmppMsg{
					body:      body,
					uiElement: EL,
					raw:       "Go fuck yourself",
				}

				return nil
			},
		),
	)

	return
}
