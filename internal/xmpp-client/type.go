package xmppclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"os"
	"sync"

	"pain.agency/oasis/internal/xmpp-client/room"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type loginInfo struct {
	Host     string `json:"Host"`
	User     string `json:"User"`
	Password string `json:"Password"`
	TLSoff   bool   `json:"NoTLS"`
	StartTLS bool   `json:"StartTLS"`
}

type Client struct {
	ctx    context.Context
	cancel context.CancelFunc

	session *xmpp.Session

	userMap    map[string]*room.Room // FIXME: this will become a huge memory hog, if left running for a long time, maybe a backing database would help.
	userMapMtx sync.RWMutex

	newRoomNotifier sync.Cond
}

func New() (*Client, error) {
	cl := Client{
		newRoomNotifier: *sync.NewCond(&sync.Mutex{}),
	}
	cl.ctx, cl.cancel = context.WithCancel(context.Background())

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

	conn, err := d.DialServer(cl.ctx, "tcp", j, server)
	if err != nil {
		panic("Could not connect stage 1 - " + err.Error())
	}

	cl.session, err = xmpp.NewSession(cl.ctx, j.Domain(), j, conn, 0, xmpp.NewNegotiator(func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig {
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
	err = cl.session.Send(cl.ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		// return fmt.Errorf("Error sending initial presence: %w", err)
		panic("Error sending initial presence - " + err.Error())
	}

	go cl.session.Serve(xmpp.HandlerFunc(cl.handleSession))

	return &cl, nil
}

func (c *Client) Shutdown() {
	c.cancel()
	c.session.Close()
}
