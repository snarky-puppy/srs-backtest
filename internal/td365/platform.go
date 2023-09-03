package td365

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/models"
)

type Platform struct {
	client       *http.Client
	account      Account
	uat          *UserAgentTransport
	connection   *ConnectionProxy
	tradeManager TradeManager
	positions    map[int]*models.Position
	orders       map[int]*models.Position
}

func formatPrice(price float64) string {
	if price == 0 {
		return ""
	}
	return fmt.Sprintf("%.4f", price)
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
	defer internal.Close(resp.Body)
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

func (p *Platform) CreateOrder(symbol models.Symbol, direction models.Direction, size, open, stop, target float64) *models.Order {

	var (
		orderModeID int
	)
	switch {
	case stop != 0 && target != 0:
		orderModeID = 3
	case stop != 0:
		orderModeID = 2
	case target != 0:
		orderModeID = 1
	default:
		orderModeID = 0
	}

	req := InsertOpenOrderRequest{
		TradeType:          1, // ? - hard coded
		MarketID:           symbol.MarketID,
		MarketQuoteID:      symbol.QuoteID,
		TradeModeID:        direction == models.Short,
		OrderStake:         fmt.Sprintf("%0.2f", size),
		OrderModeID:        orderModeID,
		OrderTypeID:        1, // good till cancelled
		OrderPriceModeID:   2, // 1 - at market, 2 - at quote
		LimitOrderPrice:    formatPrice(target),
		StopOrderPrice:     formatPrice(stop),
		HasIfDoneOrder:     stop != 0 || target != 0,
		IDOIsGuarantee:     false,
		IDOOrderModeID:     orderModeID,
		IDOLimitOrderPrice: formatPrice(target),
		IDOStopOrderPrice:  formatPrice(stop),
	}

	res, err := p.Post(p.account.TradeUrl("/UTSAPI.asmx/InsertOpenOrder"), req)
	if err != nil {
		log.Println("Failed to create order:", err)
		return nil
	}
	defer internal.Close(res.Body)

	var resp InsertOpenOrderResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		log.Println("Failed to parse InsertOpenOrderResponse:", err)
		return nil
	}

	order := &models.Order{
		Id:          internal.MustInt(resp.D.OrderID),
		Symbol:      symbol,
		Direction:   direction,
		Size:        size,
		OpenPrice:   open,
		StopPrice:   stop,
		TargetPrice: target,
	}

	return order
}

func (p *Platform) ExitPosition(id int, tick *models.Tick) *models.Position {

	position := p.positions[id]
	if position == nil {
		fmt.Println("ExitPosition: position not found", id)
		debug.PrintStack()
	}

	price := tick.Bid
	tradeMode := false
	if position.Direction == models.Short {
		price = tick.Ask
		tradeMode = true
	}

	req := InsertClosePositionRequest{
		IsKaazingFeed: true,
		Key:           tick.Key,
		MarketID:      position.Symbol.MarketID,
		PositionID:    position.Id,
		Price:         formatPrice(price),
		QuoteID:       position.Symbol.QuoteID,
		Stake:         formatPrice(position.Size),
		TradeMode:     tradeMode,
		UserAgent:     UserAgentShort,
	}

	res, err := p.Post(p.account.TradeUrl("/UTSAPI.asmx/InsertClosePosition"), req)
	if err != nil {
		log.Println("Failed to close position:", err)
		return nil
	}
	defer internal.Close(res.Body)

	var resp InsertClosePositionResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		log.Println("Failed to parse InsertClosePositionResponse:", err)
		return nil
	}

	position.ExitTime = time.Now()
	position.ExitPrice = resp.D.Price

	log.Println("Closed position", position.Id, "at", position.ExitPrice)

	delete(p.positions, id)

	return position
}

func (p *Platform) CancelOrder(id int) {
	req := DeleteOrderRequest{
		OrderID: id,
	}

	res, err := p.Post(p.account.TradeUrl("/UTSAPI.asmx/DeleteOrder"), req)
	if err != nil {
		log.Println("Failed to close order:", err)
		return
	}
	defer internal.Close(res.Body)

	delete(p.orders, id)
}

/**
OpenOrder - order to open position
CloseOrder - order to close position

AmendCloseOrder - position has order id
AmendOpenOrder - order not filled
    - stopOrderPrice - open price
	- IDOStopOrderPrice - stop price
*/

func (p *Platform) UpdatePosition(id int, stop, target float64) {
	position := p.positions[id]
	if position == nil {
		fmt.Println("UpdatePosition: position not found", id)
		debug.PrintStack()
		return
	}

	//req := AmendCloseOrderRequest{
	//	Market:  position.Symbol.MarketName,
	//	OrderID: position.Id,
	//}
}

func (p *Platform) GetBalance() float64 {
	//TODO implement me
	panic("implement me")
}

func (p *Platform) Subscribe(q models.Symbol) {
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

func (p *Platform) RequestBackFill(marketID int, loc *time.Location) {
	req := struct {
		MarketID         int  `json:"marketID"`
		GetAdvancedChart bool `json:"getAdvancedChart"`
	}{
		MarketID:         marketID,
		GetAdvancedChart: false,
	}

	res, err := p.Post(p.account.TradeUrl("/UTSAPI.asmx/GetChartURL"), req)
	if err != nil {
		panic(err)
	}
	defer internal.Close(res.Body)

	// calculate how many ticks to fetch based on current time
	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	// how many 1 minute intervals between now and start of day
	maxTicks := int(now.Sub(startOfDay).Minutes()) + 1

	res, err = p.client.Get(fmt.Sprintf("https://charts.finsa.com.au/data/minute/%d/mid?l=%d", marketID, maxTicks))
	if err != nil {
		panic(err)
	}
	defer internal.Close(res.Body)

	var data struct {
		Data []string `json:"data"`
	}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		panic(err)
	}

	var bars models.Series
	for _, s := range data.Data {
		bits := strings.Split(s, ",")
		if len(bits) != 5 {
			panic("bad data")
		}
		t, err := time.Parse(time.RFC3339, bits[0])
		if err != nil {
			panic(err)
		}

		pf := func(s string) float64 {
			rv, err := strconv.ParseFloat(s, 64)
			if err != nil {
				panic(err)
			}
			return internal.Round4(rv)
		}

		o := pf(bits[1])
		h := pf(bits[2])
		l := pf(bits[3])
		c := pf(bits[4])
		bars = append(bars, &models.Bar{
			Timestamp: t,
			Open:      o,
			High:      h,
			Low:       l,
			Close:     c,
			Duration:  time.Minute,
		})
	}

	sort.Slice(bars, func(i, j int) bool {
		return bars[i].Timestamp.Before(bars[j].Timestamp)
	})

	log.Println("backfill", len(bars), "bars")
	log.Println("backfill", bars[0].Timestamp, "to", bars[len(bars)-1].Timestamp)

	bars = bars.UpdateDuration(5 * time.Minute)

	p.tradeManager.Backfill(marketID, bars)
}

func (p *Platform) Post(url string, req interface{}) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return p.client.Post(url, "application/json; charset=utf-8", bytes.NewReader(body))
}

func newPlatform(client *http.Client, uat *UserAgentTransport, account Account, manager TradeManager) *Platform {
	rv := &Platform{
		client:       client,
		account:      account,
		uat:          uat,
		connection:   NewConnectionProxy(uat.CurrentUrl, account),
		tradeManager: manager,
	}

	rv.updateSessionToken()
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			rv.updateSessionToken()
		}
	}()

	rv.connection.StartConnectionLoop()

	return rv
}
