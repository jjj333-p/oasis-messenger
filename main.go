package main

import (
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
		//while chat, err := client.Recv(); chat != nil {
		//
		//}
		for {
			event, err := client.Recv()
			if err != nil {
				log.Println("Error receiving event - " + err.Error())
			}

			log.Println(event)
		}
	}()

	//options := xmpp.Options{}
}
