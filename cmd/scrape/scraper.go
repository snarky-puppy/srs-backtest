package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mwlazlo/srs/internal/pp"
)

const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/113.0"

type MarketGroup struct {
	D []struct {
		Type                      string `json:"__type"`
		ID                        int    `json:"ID"`
		Name                      string `json:"Name"`
		IsSuperGroup              bool   `json:"IsSuperGroup"`
		IsWhiteLabelPopularMarket bool   `json:"IsWhiteLabelPopularMarket"`
		HasSubscription           bool   `json:"HasSubscription"`
	} `json:"d"`
}

type FetchQuoteRequest struct {
	GroupID   string `json:"groupID"`
	Keyword   string `json:"keyword"`
	Portfolio bool   `json:"portfolio"`
	Search    bool   `json:"search"`
	Popular   bool   `json:"popular"`
}

type PopularQuotes []MarketQuote

func (q PopularQuotes) Find(s string) (MarketQuote, bool) {
	for _, quote := range q {
		if quote.MarketName == s {
			return quote, true
		}
	}
	return MarketQuote{}, false
}

type FetchQuoteResponse struct {
	D []MarketQuote `json:"d"`
}
type MarketQuote struct {
	Type                  string  `json:"__type"`
	MarketID              int     `json:"MarketID"`
	QuoteID               int     `json:"QuoteID"`
	AtQuoteAtMarket       int     `json:"AtQuoteAtMarket"`
	ExchangeID            int     `json:"ExchangeID"`
	PrcGenFractionalPrice int     `json:"PrcGenFractionalPrice"`
	PrcGenDecimalPlaces   int     `json:"PrcGenDecimalPlaces"`
	High                  int     `json:"High"`
	Low                   int     `json:"Low"`
	DailyChange           int     `json:"DailyChange"`
	Bid                   int     `json:"Bid"`
	Ask                   int     `json:"Ask"`
	BetPer                float64 `json:"BetPer"`
	IsGSLPercent          int     `json:"IsGSLPercent"`
	GSLDis                float64 `json:"GSLDis"`
	MinCloseOrderDisTicks float64 `json:"MinCloseOrderDisTicks"`
	MinOpenOrderDisTicks  float64 `json:"MinOpenOrderDisTicks"`
	DisplayBetPer         float64 `json:"DisplayBetPer"`
	IsInPortfolio         bool    `json:"IsInPortfolio"`
	Tradable              bool    `json:"Tradable"`
	TradeOnWeb            bool    `json:"TradeOnWeb"`
	CallOnly              bool    `json:"CallOnly"`
	MarketName            string  `json:"MarketName"`
	TradeStartTime        string  `json:"TradeStartTime"`
	Currency              string  `json:"Currency"`
	AllowGtdsStops        int     `json:"AllowGtdsStops"`
	ForceOpen             bool    `json:"ForceOpen"`
	Margin                float64 `json:"Margin"`
	MarginType            bool    `json:"MarginType"`
	GSLCharge             float32 `json:"GSLCharge"`
	IsGSLChargePercent    int     `json:"IsGSLChargePercent"`
	Spread                float64 `json:"Spread"`
	TradeRateType         int     `json:"TradeRateType"`
	OpenTradeRate         float32 `json:"OpenTradeRate"`
	CloseTradeRate        float32 `json:"CloseTradeRate"`
	MinOpenTradeRate      float32 `json:"MinOpenTradeRate"`
	MinCloseTradeRate     float32 `json:"MinCloseTradeRate"`
	PriceDecimal          float64 `json:"PriceDecimal"`
	Subscription          bool    `json:"Subscription"`
	SuperGroupID          int     `json:"SuperGroupID"`
}

type FormData struct {
	Action                  string
	__EVENTTARGET           string
	__EVENTARGUMENT         string
	__VIEWSTATE             string
	__VIEWSTATEGENERATOR    string
	__EVENTVALIDATION       string
	hfLanguageID            string
	hfPlatform              string
	hfTradingLevel          string
	hfParentChildType       string
	hfWhiteLabelID          string
	hfAccountID             string
	hfTradingCurrencySymbol string
	hfTradingCurrencyCode   string
	hfAvailableCurrency     string
	hfClientTypeID          string
	hfPlatformID            string
	hfTradingMode           string
	hfEnablePolling         string
	hfSafeChargeUrl         string
	hfLoginID               string
	hfHomeEmail             string
	hfClientName            string
	hfSessionID             string
	hfWebStreaming          string
}

// UserAgentTransport is custom Transport that adds the User-Agent header to each request.
type UserAgentTransport struct {
	Transport  http.RoundTripper
	UserAgent  string
	Referer    string
	CurrentUrl string
}

// RoundTrip implements the RoundTripper interface.
func (t *UserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add the User-Agent header to the request
	req.Header.Set("User-Agent", t.UserAgent)
	req.Header.Set("Referer", t.Referer)

	// Perform the actual request using the underlying Transport
	return t.Transport.RoundTrip(req)
}

type Scraper struct {
	client           *http.Client
	uat              *UserAgentTransport
	formData         FormData
	connection       *ConnectionProxy
	marketSuperGroup MarketGroup
	PopularQuotes    PopularQuotes
}

func NewScraper() *Scraper {
	var uat *UserAgentTransport
	scraper := &Scraper{
		client: &http.Client{
			Jar: func() http.CookieJar {
				cookieJar, err := cookiejar.New(nil)
				if err != nil {
					panic(err)
				}
				return cookieJar
			}(),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				fmt.Println("Redirected from:", via[len(via)-1].URL)
				fmt.Println("Redirected to:", req.URL)
				uat.CurrentUrl = req.URL.String()
				uat.Referer = via[len(via)-1].URL.String()
				return nil
			},
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
	uat = &UserAgentTransport{
		Transport: scraper.client.Transport,
		UserAgent: UserAgent,
	}
	scraper.uat = uat

	scraper.client.Transport = uat

	scraper.OpenDemo()
	scraper.connection = NewConnectionProxy("DEMO", scraper.formData.hfLoginID, scraper.formData.hfPlatform)

	scraper.updateSessionToken()
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			scraper.updateSessionToken()
		}
	}()

	scraper.connection.ConnectLoop()
	scraper.FetchMarketSuperGroup()
	scraper.FetchPopularQuotes()

	return scraper
}

func (s *Scraper) OpenDemo() {

	link := s.getOneClickDemoLink()
	fmt.Printf("Link: %s\n", link)

	s.uat.Referer = link

	s.formData = s.clickDemoLink(link)
	fmt.Printf("Form: %+v\n", s.formData.Action)

	s.uat.Referer = s.uat.CurrentUrl
}

func (s *Scraper) updateSessionToken() {
	resp, err := s.client.Post("https://demo.tradedirect365.com/UTSAPI.asmx/UpdateClientSessionID", "application/json; charset=utf-8", nil)
	if err != nil {
		log.Println("Failed to update session token:", err)
	}
	if resp.StatusCode != 200 {
		log.Println("Failed to update session token:", resp.Status)
	}

	// link has query string "ots=xxxxxx"
	// xxxxxx is the key of the session cookie
	log.Println("Finding token", s.uat.CurrentUrl)
	re, err := regexp.Compile(`ots=(\w+)`)
	if err != nil {
		panic(err)
	}
	matches := re.FindStringSubmatch(s.uat.CurrentUrl)
	if len(matches) != 2 {
		panic("no match")
	}

	log.Println("ots", matches[1])

	token := ""
	cookies := s.client.Jar.Cookies(&url.URL{Scheme: "https", Host: "tradedirect365.com"})
	for _, cookie := range cookies {
		log.Println(cookie.Name, cookie.Value)
		if cookie.Name == matches[1] {
			token = cookie.Value
		}
	}

	if token == "" {
		panic("no token")
	}

	s.connection.UpdateToken(token)
}

func (s *Scraper) clickDemoLink(link string) (formData FormData) {
	doc := s.docFromLink(link)

	doc.Find("form#form1").Each(func(_ int, s *goquery.Selection) {
		action, _ := s.Attr("action")
		formData.Action = action
	})

	doc.Find("form#form1 input[type=hidden]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		value, _ := s.Attr("value")

		switch name {
		case "__EVENTTARGET":
			formData.__EVENTTARGET = value
		case "__EVENTARGUMENT":
			formData.__EVENTARGUMENT = value
		case "__VIEWSTATE":
			formData.__VIEWSTATE = value
		case "__VIEWSTATEGENERATOR":
			formData.__VIEWSTATEGENERATOR = value
		case "__EVENTVALIDATION":
			formData.__EVENTVALIDATION = value
		case "hfLanguageID":
			formData.hfLanguageID = value
		case "hfPlatform":
			formData.hfPlatform = value
		case "hfTradingLevel":
			formData.hfTradingLevel = value
		case "hfParentChildType":
			formData.hfParentChildType = value
		case "hfWhiteLabelID":
			formData.hfWhiteLabelID = value
		case "hfAccountID":
			formData.hfAccountID = value
		case "hfTradingCurrencySymbol":
			formData.hfTradingCurrencySymbol = value
		case "hfTradingCurrencyCode":
			formData.hfTradingCurrencyCode = value
		case "hfAvailableCurrency":
			formData.hfAvailableCurrency = value
		case "hfClientTypeID":
			formData.hfClientTypeID = value
		case "hfPlatformID":
			formData.hfPlatformID = value
		case "hfTradingMode":
			formData.hfTradingMode = value
		case "hfEnablePolling":
			formData.hfEnablePolling = value
		case "hfSafeChargeUrl":
			formData.hfSafeChargeUrl = value
		case "hfLoginID":
			formData.hfLoginID = value
		case "hfHomeEmail":
			formData.hfHomeEmail = value
		case "hfClientName":
			formData.hfClientName = value
		case "hfSessionID":
			formData.hfSessionID = value
		case "hfWebStreaming":
			formData.hfWebStreaming = value
		}
	})
	return
}

func (s *Scraper) getOneClickDemoLink() (rv string) {
	doc := s.docFromLink("https://td365.com/")

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if s.Text() == "Try Demo Account" {
			var ok bool
			rv, ok = s.Attr("href")
			if !ok {
				panic("no href")
			}
		}
	})

	return
}

func (s *Scraper) docFromLink(link string) *goquery.Document {
	// Request the HTML page.
	res, err := s.client.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	return doc
}

func (s *Scraper) FetchMarketSuperGroup() {
	// https: //cloudtrade.tradedirect365.com/UTSAPI.asmx/GetMarketSuperGroup
	resp, err := s.client.Post("https://demo.tradedirect365.com/UTSAPI.asmx/GetMarketSuperGroup", "application/json; charset=utf-8", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&s.marketSuperGroup)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Scraper) FetchPopularQuotes() {

	body, err := json.Marshal(&FetchQuoteRequest{Popular: true})
	if err != nil {
		log.Fatal(err)
	}

	// https: //cloudtrade.tradedirect365.com/UTSAPI.asmx/GetMarketSuperGroup
	resp, err := s.client.Post("https://demo.tradedirect365.com/UTSAPI.asmx/GetMarketQuote",
		"application/json; charset=utf-8",
		bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	var quotes FetchQuoteResponse
	err = json.NewDecoder(resp.Body).Decode(&quotes)
	if err != nil {
		log.Fatal(err)
	}

	s.PopularQuotes = quotes.D
}

func (s *Scraper) Subscribe(q MarketQuote) {
	subs := []Subscription{
		{
			QuoteID:       q.QuoteID,
			PriceGrouping: "sampled",
			Action:        "subscribe",
		},
	}
	s.connection.Subscribe(subs)
}

func (s *Scraper) GetPriceChannel() chan *pp.Price {
	return s.connection.PriceCh
}
