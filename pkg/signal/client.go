package signal

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	url       string
	conn      *websocket.Conn
	msgChan   chan SignalMessage
	closeChan chan struct{}
}

func NewClient(addr string) *Client {
	return &Client{
		url:       addr,
		msgChan:   make(chan SignalMessage, 100),
		closeChan: make(chan struct{}),
	}
}

func (c *Client) Messages() <-chan SignalMessage {
	return c.msgChan
}

func (c *Client) Connect() error {
	u, err := url.Parse(c.url)
	if err != nil {
		return err
	}

	log.Printf("Connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	c.conn = conn

	// Start reading loop
	go c.readLoop()
	return nil
}

func (c *Client) readLoop() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case <-c.closeChan:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Printf("read error: %v, attempting reconnect in 5s...", err)
				time.Sleep(5 * time.Second)
				c.reconnect()
				return
			}

			var signalMsg SignalMessage
			if err := json.Unmarshal(message, &signalMsg); err != nil {
				log.Printf("json unmarshal error: %v", err)
				continue
			}

			// Only forward messages that have content
			if signalMsg.Envelope.DataMessage != nil ||
				(signalMsg.Envelope.SyncMessage != nil && signalMsg.Envelope.SyncMessage.SentMessage != nil) {
				c.msgChan <- signalMsg
			}
		}
	}
}

func (c *Client) reconnect() {
	for {
		log.Println("Reconnecting...")
		if err := c.Connect(); err == nil {
			log.Println("Reconnected!")
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func (c *Client) Close() {
	close(c.closeChan)
	if c.conn != nil {
		c.conn.Close()
	}
}
