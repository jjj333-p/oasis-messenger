package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/xmppo/go-xmpp"
	"log"
	"os"
	"strings"
)

func serverName(host string) string {
	return strings.Split(host, ":")[0]
}

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

	in := bufio.NewReader(os.Stdin)
	for line, err := in.ReadString('\n'); err == nil; line, err = in.ReadString('\n') {
		_, err := client.Send(xmpp.Chat{
			Remote: "jjj333@pain.agency",
			Type:   "chat",
			Text:   line,
		})
		if err != nil {
			log.Println("Error sending message - " + err.Error())
		}
	}

	//options := xmpp.Options{}
}
