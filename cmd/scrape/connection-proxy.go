package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mwlazlo/srs/internal/pp"
)

type ConnectionProxy struct {
	token              string
	mainUrl            string
	websocketUrl       string
	loginId            string
	tradingAccountType string
	messages           []*Subscription
	conn               *websocket.Conn
	PriceCh            chan *Price
	connectionId       *string
	receiveAuthOK      chan bool
}

func (c *ConnectionProxy) ConnectLoop() {
	if c.token == "" {
		panic("token is empty")
	}

	c.Dial()
	done := c.ReadLoop()
	log.Println(1)
	<-c.receiveAuthOK
	log.Println(2)

	go func() {
		for {
			log.Println(3)
			select {
			case <-pp.GetGracefulCtx().Done():
				return
			case <-done:
				log.Println("Connection closed, reconnecting")
				c.Dial()
				done = c.ReadLoop()
				<-c.receiveAuthOK
				for _, msg := range c.messages {
					c.SendSubscription(msg)
				}
			}
			log.Println(4)
		}
	}()
}

func NewConnectionProxy(environment, loginId, tradingAccountType string) *ConnectionProxy {
	var (
		mainUrl string
	)

	switch environment {
	case "PROD":
		mainUrl = "prod-api.finsa.com.au"
		break
	case "DEMO":
		mainUrl = "demo-api.finsa.com.au"
		break
	default:
		panic("invalid environment")
	}

	rv := &ConnectionProxy{
		loginId:            loginId,
		tradingAccountType: tradingAccountType,
		mainUrl:            fmt.Sprintf("https://%s", mainUrl),
		websocketUrl:       fmt.Sprintf("wss://%s", mainUrl),
		PriceCh:            make(chan *Price, 10),
		receiveAuthOK:      make(chan bool, 0),
	}
	return rv
}

func (c *ConnectionProxy) Subscribe(quotes []Subscription) {
	log.Println("Subscribe: ", quotes)
	// {"quoteId":6374,"priceGrouping":"Sampled","action":"subscribe"}
	for _, quote := range quotes {
		found := c._findMessage(quote.QuoteID, quote.PriceGrouping)
		msg := &Subscription{
			QuoteID:       quote.QuoteID,
			PriceGrouping: quote.PriceGrouping,
			Action:        "subscribe",
		}
		if found == nil {
			c.messages = append(c.messages, msg)
		}
		c.SendSubscription(msg)
	}
}

//func (c *ConnectionProxy) IncludeAccountSummary(include bool) {
//	options := c._getOptions()
//	options["SubscribeToAccountSummary"] = include
//	c._sendOptions()
//}
//
//func (c *ConnectionProxy) IncludeAccountDetails(include bool) {
//	options := c._getOptions()
//	options["SubscribeToAccountDetails"] = include
//	c._sendOptions()
//}

//

func (c *ConnectionProxy) CurrentAccountSummary() interface{} {
	return nil // Return the lastAccountSummary
}

func (c *ConnectionProxy) _findMessage(quoteId int, priceGrouping string) *Subscription {
	for _, msg := range c.messages {
		if msg.PriceGrouping == priceGrouping && msg.QuoteID == quoteId {
			return msg
		}
	}
	return nil
}

func (c *ConnectionProxy) UpdateToken(token string) {
	c.token = token
}

func (c *ConnectionProxy) Dial() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	headers := http.Header{}
	headers.Set("User-Agent", UserAgent)
	headers.Set("Origin", "https://demo.tradedirect365.com.au")

	log.Printf("Connecting to WebSocket server: %s", c.websocketUrl)

	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(c.websocketUrl, headers)
	if err != nil {
		log.Fatal("Error connecting to WebSocket server:", err)
	}
}

func (c *ConnectionProxy) ReadLoop() chan struct{} {
	done := make(chan struct{})
	go func() {
		defer func() {
			err := c.conn.Close()
			if err != nil {
				log.Println("Error closing connection:", err)
			}
			log.Println("Connection closed")
			close(done)
		}()
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				return
			}
			log.Println("Received message:", string(message))

			// Parse the received message into a Response struct
			response := &Response{}
			if err := json.Unmarshal(message, &response); err != nil {
				log.Println("Failed to parse message:", err)
				continue
			}
			c.handleResponse(response)
		}
	}()
	return done
}

func (c *ConnectionProxy) handleResponse(response *Response) {
	switch response.T {
	case "authenticationResponse":
		/*
			{
				"action":"authentication",
				"loginId":"demorubARb6491",
				"tradingAccountType":"SPREAD",
				"token":"VjmBgwtMdVUEPRgo+DCldf0VtLhgqohsyG4HX7wwSrwZBHemWIhVQZpY5r+zrJ/6vSWnFGQkFl4x6/r+"
			}

			{
				"t":"authenticationResponse",
				"d": {
					"Result":true,
					"Action":"authentication",
					"HasError":false
				},
				"cid":"afc718bb-80fa-412b-a865-00d4c7320a3c"
			}
		*/
		if response.D.Result {
			fmt.Println("Connection ID:", response.ConnectionID)
			if c.connectionId != nil {
				reconnect := make(map[string]interface{})
				reconnect["action"] = "reconnect"
				reconnect["originalConnectionId"] = *c.connectionId
				log.Println("Reconnecting to WebSocket server", *c.connectionId)
				if err := c.WriteJSON(reconnect); err != nil {
					log.Println("Failed to send reconnect message:", err)
				}
			} else {
				log.Println("Connected to WebSocket server")
			}
			c.receiveAuthOK <- true
			c.connectionId = &response.ConnectionID
		} else {
			log.Fatal("Failed to authenticate to WebSocket server", response.String())
		}
	case "subscribeResponse":
		if response.D.Error != "" {
			log.Println(response.D.Error)
		} else if response.D.Current != nil {
			for _, x := range response.D.Current {
				c.PriceCh <- NewPrice(x, response.D.PriceGrouping)
			}
		}
	case "connectResponse":
		if response.D.Error != "" {
			log.Println(response.D.Error)
		} else {
			authMessage := make(map[string]interface{})
			authMessage["action"] = "authentication"
			authMessage["loginId"] = c.loginId
			authMessage["tradingAccountType"] = c.tradingAccountType
			authMessage["token"] = c.token
			if err := c.WriteJSON(authMessage); err != nil {
				log.Println("Failed to send authentication message:", err)
			}
		}
	case "reconnectResponse":
		if response.D.Error != "" {
			log.Println(response.D.Error)
		}
		log.Println("Reconnected to WebSocket server")
	case "p":
		if response.D.Grouped != nil {
			for _, x := range response.D.Grouped {
				c.PriceCh <- NewPrice(x, "Grouped")
			}
		}
		if response.D.Sampled != nil {
			for _, x := range response.D.Sampled {
				c.PriceCh <- NewPrice(x, "Sampled")
			}
		}
		if response.D.Delayed != nil {
			for _, x := range response.D.Delayed {
				c.PriceCh <- NewPrice(x, "Delayed")
			}
		}
	case "accountSummary":
		log.Println("Received account summary:", response.String())
	case "accountDetails":
		log.Println("Received account details:", response.String())
	case "heartbeat":
		hb := HeartbeatData{
			SentByServer:     response.D.SentByServer,
			MessagesReceived: response.D.MessagesReceived,
			PricesReceived:   response.D.PricesReceived,
			MessagesSent:     response.D.MessagesSent,
			PricesSent:       response.D.PricesSent,
			ReceivedByClient: time.Now().UTC(),
			SentByClient:     time.Now().UTC(),
			Action:           "heartbeat",
		}
		if err := c.WriteJSON(&hb); err != nil {
			log.Println("Failed to send heartbeat message:", err)
		}
	}
}

func (c *ConnectionProxy) WriteJSON(payload interface{}) error {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	err := enc.Encode(payload)
	if err != nil {
		log.Println("Failed to encode message:", err)
		return err
	}
	log.Println("Sending message:", buf.String())
	return c.conn.WriteJSON(payload)
}

func (c *ConnectionProxy) Close() {
	err := c.conn.Close()
	if err != nil {
		log.Println("Failed to close WebSocket connection:", err)
	}
}

func (c *ConnectionProxy) SendSubscription(msg *Subscription) {
	log.Println("Sending subscription message:", msg)
	if err := c.WriteJSON(msg); err != nil {
		log.Println("Failed to send subscription message:", err)
	}
}
