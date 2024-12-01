package main

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/xmppo/go-xmpp"
	"log"
	"os"
)

//func serverName(host string) string {
//	return strings.Split(host, ":")[0]
//}

func main() {

	loginJSONbytes, err := os.ReadFile("db/login.json")
	if err != nil {
		panic("Unable to read login.json - " + err.Error())
	}

	xmppConfig := xmpp.Options{}

	if err := json.Unmarshal(loginJSONbytes, &xmppConfig); err != nil {
		panic("Could not parse login.json - " + err.Error())
	}

	fmt.Println(xmppConfig.User)

	client, err := xmppConfig.NewClient()
	if err != nil {
		panic("Could not login - " + err.Error())
	}

	go func() {
		for {
			event, err := client.Recv()
			if err != nil {
				log.Println("Error receiving event - " + err.Error())
			}

			switch v := event.(type) {
			case xmpp.Chat:
				fmt.Println(v.Remote, v.Text)
			case xmpp.Presence:
				fmt.Println(v.From, v.Show)
			}
		}
	}()

	a := app.New()
	w := a.NewWindow("Hello World")

	jidEntry := widget.NewEntry()
	jidEntry.SetPlaceHolder("JID")
	msgEntry := widget.NewEntry()
	msgEntry.SetPlaceHolder("Message")
	windowContent := container.NewVBox(jidEntry, msgEntry, widget.NewButton("Send", func() {
		_, err := client.Send(xmpp.Chat{
			Remote: jidEntry.Text,
			Type:   "chat",
			Text:   msgEntry.Text,
		})
		if err != nil {
			log.Println("Error sending message - " + err.Error())
		}
		msgEntry.SetText("")
	}))
	w.SetContent(windowContent)
	w.ShowAndRun()

	//options := xmpp.Options{}
}
