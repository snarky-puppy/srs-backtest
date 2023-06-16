package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

// {"quoteId":6374,"priceGrouping":"Sampled","action":"subscribe"}
type Subscription struct {
	QuoteID       int    `json:"QuoteID"`
	PriceGrouping string `json:"PriceGrouping"`
	Action        string `json:"Action"`
}

// 6647,  quoteid
// 32810.6, bid
// 32812.1, ask
// -219.7, daily
// d,      arrow
// 1,      tradeable
// 33010.2, high
// 32790.5, low
// SJceTo2Ccn+zJEjuordzCQR53Wii/TPEfT/YgsbMTeo=, encrypted
// 0, Callonly
// 32811.3, last_traded_price
// 638211389974460000, unknown1
// 828331 unknown2

type Price struct {
	QuoteID          int       `json:"QuoteID"`
	Bid              float64   `json:"Bid"`
	Ask              float64   `json:"Ask"`
	DailyChange      float64   `json:"DailyChange"`
	Arrow            string    `json:"Arrow"`
	Tradable         bool      `json:"Tradable"`
	High             float64   `json:"High"`
	Low              float64   `json:"Low"`
	EncryptedMessage string    `json:"EncryptedMessage"`
	CallOnly         bool      `json:"CallOnly"`
	LastTradedPrice  float64   `json:"LastTradedPrice"`
	Grouping         string    `json:"grouping"`
	Timestamp        time.Time `json:"timestamp"`
	Unknown2         int64     `json:"Unknown2"`
}

func (p Price) String() string {
	rv, _ := json.Marshal(p)
	return string(rv)
}

func newFloat(s string) float64 {
	rv, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Println(err)
	}
	// round rv to 6 decimal places
	rv = float64(int(rv*1000000)) / 1000000
	return rv
}

func NewPrice(line, grouping string) *Price {
	fields := strings.Split(line, ",")
	quoteID, _ := strconv.Atoi(fields[0])
	bid := newFloat(fields[1])
	ask := newFloat(fields[2])
	dailyChange := newFloat(fields[3])
	arrow := fields[4]
	tradable, _ := strconv.ParseBool(fields[5])
	high := newFloat(fields[6])
	low := newFloat(fields[7])
	encryptedMessage := fields[8]
	callOnly, _ := strconv.ParseBool(fields[9])
	lastTradedPrice := newFloat(fields[10])
	winTicks, _ := strconv.ParseInt(fields[11], 10, 64)
	unk2, _ := strconv.ParseInt(fields[12], 10, 64)

	price := Price{
		QuoteID:          quoteID,
		Bid:              bid,
		Ask:              ask,
		DailyChange:      dailyChange,
		Arrow:            arrow,
		Tradable:         tradable,
		High:             high,
		Low:              low,
		EncryptedMessage: encryptedMessage,
		CallOnly:         callOnly,
		LastTradedPrice:  lastTradedPrice,
		Grouping:         grouping,
		Timestamp:        time.Unix(0, ((winTicks)-60*60*24*365*1970*10000000)*100),
		Unknown2:         unk2,
	}
	return &price
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
