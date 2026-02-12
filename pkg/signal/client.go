package signal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	url       string
	conn      *websocket.Conn
	httpURL   string // set when using HTTP polling mode
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

	// HTTP(S) scheme → use long-polling
	if u.Scheme == "http" || u.Scheme == "https" {
		log.Printf("Connecting to %s (HTTP polling)", u.String())
		// Verify the endpoint is reachable
		resp, err := http.Get(u.String())
		if err != nil {
			return fmt.Errorf("http poll: %w", err)
		}
		resp.Body.Close()
		c.httpURL = u.String()
		go c.pollLoop()
		return nil
	}

	// WS(S) scheme → use WebSocket
	log.Printf("Connecting to %s (WebSocket)", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	c.conn = conn
	go c.readLoop()
	return nil
}

func (c *Client) pollLoop() {
	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		resp, err := http.Get(c.httpURL)
		if err != nil {
			log.Printf("poll error: %v, retrying in 5s...", err)
			select {
			case <-c.closeChan:
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || len(body) == 0 {
			continue
		}

		// Response can be a single message or an array
		body = bytes_TrimSpace(body)
		if len(body) == 0 {
			continue
		}

		if body[0] == '[' {
			var msgs []SignalMessage
			if err := json.Unmarshal(body, &msgs); err != nil {
				// Try as array of raw envelopes
				log.Printf("poll json array unmarshal error: %v", err)
				continue
			}
			for _, msg := range msgs {
				if msg.Envelope.DataMessage != nil ||
					(msg.Envelope.SyncMessage != nil && msg.Envelope.SyncMessage.SentMessage != nil) {
					c.msgChan <- msg
				}
			}
		} else if body[0] == '{' {
			var msg SignalMessage
			if err := json.Unmarshal(body, &msg); err != nil {
				log.Printf("poll json unmarshal error: %v", err)
				continue
			}
			if msg.Envelope.DataMessage != nil ||
				(msg.Envelope.SyncMessage != nil && msg.Envelope.SyncMessage.SentMessage != nil) {
				c.msgChan <- msg
			}
		}

		// Small delay between polls to avoid hammering
		select {
		case <-c.closeChan:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func bytes_TrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
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
