package main

import (
	"encoding/json"
	"time"
)

// {"quoteId":6374,"priceGrouping":"Sampled","action":"subscribe"}
type Subscription struct {
	QuoteID       int    `json:"QuoteID"`
	PriceGrouping string `json:"PriceGrouping"`
	Action        string `json:"Action"`
}

type HeartbeatData struct {
	SentByServer     time.Time `json:"SentByServer"`
	MessagesReceived int       `json:"MessagesReceived"`
	PricesReceived   int       `json:"PricesReceived"`
	MessagesSent     int       `json:"MessagesSent"`
	PricesSent       int       `json:"PricesSent"`
	ReceivedByClient time.Time `json:"ReceivedByClient"`
	SentByClient     time.Time `json:"SentByClient"`
	Action           string    `json:"action"`
}

type Data struct {
	HeartbeatData
	Result        bool     `json:"Result"`
	Error         string   `json:"Error"`
	HasError      bool     `json:"HasError"`
	Action        string   `json:"Action"`
	Current       []string `json:"Current"`
	PriceGrouping string   `json:"PriceGrouping"`
	Grouped       []string `json:"gp"`
	Sampled       []string `json:"sp"`
	Delayed       []string `json:"dp"`
}

type Response struct {
	D            Data   `json:"d"`
	T            string `json:"t"`
	ConnectionID string `json:"cid"`
}

func (r Response) String() string {
	buf, _ := json.Marshal(r)
	return string(buf)
}
