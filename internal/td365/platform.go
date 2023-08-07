package td365

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

type Platform struct {
	client     *http.Client
	account    Account
	uat        *UserAgentTransport
	connection *ConnectionProxy
}

func (p *Platform) GetPopularMarkets() MarketGroup {
	//TODO implement me
	panic("implement me")
}

func (p *Platform) GetPopularQuotes() []MarketQuote {
	body, err := json.Marshal(&FetchQuoteRequest{Popular: true})
	if err != nil {
		log.Fatal(err)
	}
	resp, err := p.client.Post(p.account.TradeUrl("/UTSAPI.asmx/GetMarketQuote"),
		"application/json; charset=utf-8",
		bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		log.Printf("status code error: %d %s", resp.StatusCode, resp.Status)
		panic("status code error")
	}

	var quotes FetchQuoteResponse
	err = json.NewDecoder(resp.Body).Decode(&quotes)
	if err != nil {
		log.Fatal(err)
	}

	return quotes.D
}

func (p *Platform) CreateOrder(symbol models.Symbol, direction models.Direction, size, open, stop, target float64) *models.Trade {
	//TODO implement me
	panic("implement me")
}

func (p *Platform) ExitPosition(id int) *models.Trade {
	//TODO implement me
	panic("implement me")
}

func (p *Platform) CancelOrder(id int) {
	//TODO implement me
	panic("implement me")
}

func (p *Platform) UpdatePosition(id int, stop, target float64) {
	//TODO implement me
	panic("implement me")
}

func (p *Platform) GetBalance() float64 {
	//TODO implement me
	panic("implement me")
}

func (p *Platform) Subscribe(q MarketQuote) {
	subs := []Subscription{
		{
			QuoteID:       q.QuoteID,
			PriceGrouping: "sampled",
			Action:        "subscribe",
		},
	}
	p.connection.Subscribe(subs)
}

func (p *Platform) updateSessionToken() {
	resp, err := p.client.Post(p.account.TradeUrl("/UTSAPI.asmx/UpdateClientSessionID"), "application/json; charset=utf-8", nil)
	if err != nil {
		log.Println("Failed to update session token:", err)
	}
	if resp.StatusCode != 200 {
		log.Println("Failed to update session token:", resp.Status)
	}

	// Platform URL has query string "ots=xxxxxx"
	// xxxxxx is the key of the session cookie
	log.Println("Finding token", p.uat.CurrentUrl)
	re, err := regexp.Compile(`ots=(\w+)`)
	if err != nil {
		panic(err)
	}
	matches := re.FindStringSubmatch(p.uat.CurrentUrl)
	if len(matches) != 2 {
		panic("no match")
	}

	log.Println("ots", matches[1])

	token := ""
	cookies := p.client.Jar.Cookies(&url.URL{Scheme: "https", Host: "tradedirect365.com"})
	for _, cookie := range cookies {
		log.Println(cookie.Name, cookie.Value)
		if cookie.Name == matches[1] {
			token = cookie.Value
		}
	}

	if token == "" {
		panic("no token")
	}

	p.connection.UpdateToken(token)
}

func newPlatform(client *http.Client, uat *UserAgentTransport, account Account) *Platform {
	rv := &Platform{
		client:     client,
		account:    account,
		uat:        uat,
		connection: NewConnectionProxy(uat.CurrentUrl, account),
	}

	rv.updateSessionToken()
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			rv.updateSessionToken()
		}
	}()

	go rv.connection.ConnectLoop()

	return rv
}
