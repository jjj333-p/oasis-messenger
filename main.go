package main

import (
	"encoding/json"
	"fmt"
	"github.com/xmppo/go-xmpp"
	"strings"
)

func serverName(host string) string {
	return strings.Split(host, ":")[0]
}

func main() {

	xmppConfig := xmpp.Options{}

	if err := json.Unmarshal([]byte("{\"user\":\"meow\"}"), &xmppConfig); err != nil {
		panic(err)
	}

	fmt.Println(xmppConfig.User)

	//options := xmpp.Options{}
}
