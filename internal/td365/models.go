package td365

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

/*
accountDetails:
{
  "t": "accountDetails",
  "d": {
    "ClientId": 100194764,
    "TradingAccountType": "Spread",
    "OpeningOrders": {
      "Status": 0,
      "TotalRecords": 0,
      "Records": [ ]
    },
    "Currencies": {
      "Status": 0,
      "TotalRecords": 1,
      "Records": [
        {
          "CurrencyCode": "DMD",
          "Currency": "DMD",
          "CurrencySymbol": "DMD",
          "AccountBalance": 213.94,
          "CreditAllocation": 10000.0,
          "OpenPL": -0.6,
          "AccountValuation": 10213.34,
          "InitialMargin": 79.9,
          "TradingResources": 10133.44,
          "Percentage": "250% +",
          "MarginPercentage": "12783.21%",
          "VariationMarginRequired": 10113.47,
          "WaivedInitialMarginLimit": 0.0,
          "pt": 0,
          "Status": 0,
          "IsTotal": false
        }
      ]
    },
    "Positions": {
      "Status": 0,
      "TotalRecords": 1,
      "Records": [
        {
          "PositionID": 23099554,
          "MarketID": 17068,
          "QuoteID": 6374,
          "CurrencySymbol": "DMD",
          "Type": "1",
          "MarketName": "Germany 40 - Rolling Cash",
          "Direction": "Ask",
          "ExpiryDateTime": "30/12/31",
          "CreationTime": "08/08/23 01:10:18",
          "CreationTimeUTC": "2023-08-08T00:10:18.6903102Z",
          "Stake": 1,
          "OpeningPrice": "15981.7",
          "OpeningPriceDecimal": 15981.7,
          "CurrentPrice": "15982.3",
          "CurrentPriceDecimal": 15982.3,
          "OpenPL": -0.6,
          "StopOrderPrice": "-",
          "LimitOrderPrice": "-",
          "IMR": 79.9,
          "PrcGenDecimalPlaces": 1,
          "BetPer": 1.0,
          "Tradable": true,
          "IsRollingMarket": true,
          "IsTriggered": false,
          "CurrencyCode": "DMD",
          "IsTotal": false
        }
      ]
    },
    "Alerts": {
      "TotalRecords": 0,
      "records": [ ]
    },
    "ClientLanguageId": 1,
    "CalculatedUTCTicks": 638270517454626006
  },
  "cid": "87c835bd-c821-425c-971b-0915460277ff"
}

accountDetails with a buy order

*/

type AccountSummaryPayload struct {
	AccountID               string  `json:"AccountID"`
	PlatformID              int     `json:"PlatformID"`
	AccountValuation        float64 `json:"AccountValuation"`
	FundedPercentageString  string  `json:"FundedPercentageString"`
	ClientId                int     `json:"ClientId"`
	TradingAccountType      string  `json:"TradingAccountType"`
	Margin                  float64 `json:"Margin"`
	OpenPnLQuote            float64 `json:"OpenPnLQuote"`
	AccountBalance          float64 `json:"AccountBalance"`
	Credit                  float64 `json:"Credit"`
	WaivedMargin            float64 `json:"WaivedMargin"`
	Resources               float64 `json:"Resources"`
	ChangeIMR               float64 `json:"ChangeIMR"`
	VariationMarginRequired float64 `json:"VariationMarginRequired"`
	MarginPercent           float64 `json:"MarginPercent"`
}

type OpeningOrderRecord struct {
	Currency           string    `json:"Currency"`
	CurrentPrice       float64   `json:"CurrentPrice"`
	Direction          string    `json:"Direction"`
	ExpiryDate         string    `json:"ExpiryDate"`
	GoodTill           string    `json:"GoodTill"`
	IDOLimitOrderPrice string    `json:"IDOLimitOrderPrice"`
	IDOStopOrderPrice  string    `json:"IDOStopOrderPrice"`
	IDOGuaranteed      bool      `json:"IDOGuaranteed"`
	IsTriggered        bool      `json:"IsTriggered"`
	LimitOrderPrice    string    `json:"LimitOrderPrice"`
	Margin             float64   `json:"Margin"`
	Market             string    `json:"Market"`
	MarketID           int       `json:"MarketID"`
	MarketTradable     bool      `json:"MarketTradable"`
	OrderID            int       `json:"OrderID"`
	Period             string    `json:"Period"`
	CreationTimeUTC    time.Time `json:"CreationTimeUTC"`
	QuoteId            int       `json:"QuoteId"`
	QuoteMode          string    `json:"QuoteMode"`
	Stake              int       `json:"Stake"`
	Status             int       `json:"Status"`
	StopOrderPrice     string    `json:"StopOrderPrice"`
	Type               string    `json:"Type"`
	TrailingPoint      int       `json:"TrailingPoint"`
	IsGuarantee        bool      `json:"IsGuarantee"`
	IsForceOpen        bool      `json:"IsForceOpen"`
	OrderPriceModeEnum string    `json:"OrderPriceModeEnum"`
	CurrencySymbol     string    `json:"CurrencySymbol"`
	CurrencyCode       string    `json:"CurrencyCode"`
}

type PositionRecord struct {
	PositionID          int       `json:"PositionID"`
	MarketID            int       `json:"MarketID"`
	QuoteID             int       `json:"QuoteID"`
	CurrencySymbol      string    `json:"CurrencySymbol"`
	Type                string    `json:"Type"`
	MarketName          string    `json:"MarketName"`
	Direction           string    `json:"Direction"`
	ExpiryDateTime      string    `json:"ExpiryDateTime"`
	CreationTime        string    `json:"CreationTime"`
	CreationTimeUTC     time.Time `json:"CreationTimeUTC"`
	Stake               int       `json:"Stake"`
	OpeningPrice        string    `json:"OpeningPrice"`
	OpeningPriceDecimal float64   `json:"OpeningPriceDecimal"`
	CurrentPrice        string    `json:"CurrentPrice"`
	CurrentPriceDecimal float64   `json:"CurrentPriceDecimal"`
	OpenPL              float64   `json:"OpenPL"`
	StopOrderPrice      string    `json:"StopOrderPrice"`
	LimitOrderPrice     string    `json:"LimitOrderPrice"`
	IMR                 float64   `json:"IMR"`
	PrcGenDecimalPlaces int       `json:"PrcGenDecimalPlaces"`
	BetPer              float64   `json:"BetPer"`
	Tradable            bool      `json:"Tradable"`
	IsRollingMarket     bool      `json:"IsRollingMarket"`
	IsTriggered         bool      `json:"IsTriggered"`
	CurrencyCode        string    `json:"CurrencyCode"`
	IsTotal             bool      `json:"IsTotal"`
}

type CurrencyRecord struct {
	CurrencyCode             string  `json:"CurrencyCode"`
	Currency                 string  `json:"Currency"`
	CurrencySymbol           string  `json:"CurrencySymbol"`
	AccountBalance           float64 `json:"AccountBalance"`
	CreditAllocation         float64 `json:"CreditAllocation"`
	OpenPL                   float64 `json:"OpenPL"`
	AccountValuation         float64 `json:"AccountValuation"`
	InitialMargin            float64 `json:"InitialMargin"`
	TradingResources         float64 `json:"TradingResources"`
	Percentage               string  `json:"Percentage"`
	MarginPercentage         string  `json:"MarginPercentage"`
	VariationMarginRequired  float64 `json:"VariationMarginRequired"`
	WaivedInitialMarginLimit float64 `json:"WaivedInitialMarginLimit"`
	Pt                       int     `json:"pt"`
	Status                   int     `json:"Status"`
	IsTotal                  bool    `json:"IsTotal"`
}

type AlertRecord struct {
}

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

	// AccountSummary
	ClientId           int    `json:"ClientId"`
	TradingAccountType string `json:"TradingAccountType"`
	OpeningOrders      struct {
		Status       int                  `json:"Status"`
		TotalRecords int                  `json:"TotalRecords"`
		Records      []OpeningOrderRecord `json:"Records"`
	} `json:"OpeningOrders"`
	Currencies struct {
		Status       int              `json:"Status"`
		TotalRecords int              `json:"TotalRecords"`
		Records      []CurrencyRecord `json:"Records"`
	} `json:"Currencies"`
	Positions struct {
		Status       int              `json:"Status"`
		TotalRecords int              `json:"TotalRecords"`
		Records      []PositionRecord `json:"Records"`
	} `json:"Positions"`
	Alerts struct {
		TotalRecords int           `json:"TotalRecords"`
		Records      []AlertRecord `json:"records"`
	} `json:"Alerts"`
	ClientLanguageId   int   `json:"ClientLanguageId"`
	CalculatedUTCTicks int64 `json:"CalculatedUTCTicks"`
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

type Account struct {
	Id             int    `json:"id"`
	Platform       string `json:"platform"`
	PlatformIcon   string `json:"platformIcon"`
	Account        string `json:"account"`
	Backend        string `json:"backend"`
	AccountType    string `json:"accountType"`
	Currency       string `json:"currency"`
	CurrencySymbol string `json:"currencySymbol"`
	Balance        string `json:"balance"`
	Equity         string `json:"equity"`
	Button         struct {
		Text   string `json:"text"`
		Type   string `json:"type"`
		LinkTo string `json:"linkTo"`
	} `json:"button"`
	PaymentsLink    string `json:"paymentsLink"`
	CtLoginId       string `json:"ct_login_id"`
	CtLoginPassword string `json:"ct_login_password"`
}

func (a Account) TradeUrl(s string) string {
	if a.AccountType == "DEMO" {
		return fmt.Sprintf("https://demo.tradedirect365.com%s", s)
	}
	return fmt.Sprintf("https://cloudtrade.tradedirect365.com%s", s)
}

func (a Account) WsUrl() string {
	if a.AccountType == "DEMO" {
		return "wss://demo-api.finsa.com.au/"
	}
	return "wss://prod-api.finsa.com.au/"
}

type AccountsResult struct {
	Count    int         `json:"count"`
	Next     interface{} `json:"next"`
	Previous interface{} `json:"previous"`
	Results  []Account   `json:"results"`
}

type Token struct {
	AccessToken string `json:"access_token"`
	IdToken     string `json:"id_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	ExpiryTime  time.Time
}

func NewToken(data io.Reader) Token {
	rv := Token{}
	err := json.NewDecoder(data).Decode(&rv)
	if err != nil {
		panic(err)
	}
	rv.ExpiryTime = time.Now().Add(time.Duration(rv.ExpiresIn) * time.Second)
	return rv
}

func (t *Token) GetIdToken() (rv IdToken) {
	// IdToken is a jwt
	parts := strings.Split(t.IdToken, ".")
	if len(parts) != 3 {
		panic("invalid id token")
	}

	dec, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		panic(err)
	}

	err = json.NewDecoder(strings.NewReader(string(dec))).Decode(&rv)
	if err != nil {
		panic(err)
	}
	return rv
}

type IdToken struct {
	HttpsFinsaId           int    `json:"https://finsa/id"`
	HttpsFinsaName         string `json:"https://finsa/name"`
	HttpsFinsaUserMetadata struct {
		About struct {
			DateOfBirth string `json:"date_of_birth"`
			AddrStreet  string `json:"addr_street"`
			AddrLine2   string `json:"addr_line_2"`
			AddrCity    string `json:"addr_city"`
			AddrZip     string `json:"addr_zip"`
		} `json:"about"`
		Personal struct {
			FirstName   string      `json:"first_name"`
			LastName    string      `json:"last_name"`
			Telephone   string      `json:"telephone"`
			AddrCountry string      `json:"addr_country"`
			Title       interface{} `json:"title"`
		} `json:"personal"`
	} `json:"https://finsa/user_metadata"`
	Nickname      string    `json:"nickname"`
	Name          string    `json:"name"`
	Picture       string    `json:"picture"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Iss           string    `json:"iss"`
	Aud           string    `json:"aud"`
	Iat           int       `json:"iat"`
	Exp           int       `json:"exp"`
	Sub           string    `json:"sub"`
}

/*
	{
	  "d": {
	    "__type": "TradingPlatform.Result",
	    "Status": 0,
	    "Message": null
	  }
*/
type Result struct {
	Type    string      `json:"__type"`
	Status  int         `json:"Status"`
	Message interface{} `json:"Message"`
}

/*
	{
	  "tradeType": 1,
	  "marketID": 17068,
	  "marketQuoteID": 6374,
	  "tradeModeID": false,
	  "orderStake": "1",
	  "orderModeID": 1,
	  "orderTypeID": 2,
	  "orderPriceModeID": 2,
	  "limitOrderPrice": 15600,
	  "stopOrderPrice": 0,
	  "hasIfDoneOrder": true,
	  "IDOIsGuarantee": false,
	  "IDOOrderModeID": 3,
	  "IDOLimitOrderPrice": "15700.0",
	  "IDOStopOrderPrice": "15570.0"
	}

// sell:

	{
	  "tradeType": 1,
	  "marketID": "17068",
	  "marketQuoteID": "6374",
	  "tradeModeID": true,
	  "orderStake": "66",
	  "orderModeID": 2,
	  "orderTypeID": 2,
	  "orderPriceModeID": 2,
	  "limitOrderPrice": 0,
	  "stopOrderPrice": 15705,
	  "hasIfDoneOrder": true,
	  "IDOIsGuarantee": false,
	  "IDOOrderModeID": 2,
	  "IDOLimitOrderPrice": 0,
	  "IDOStopOrderPrice": 15725
	}

// buy:

	{
	  "tradeType": 1,
	  "marketID": 17068,
	  "marketQuoteID": 6374,
	  "tradeModeID": false,
	  "orderStake": "66",
	  "orderModeID": 2,
	  "orderTypeID": 2,
	  "orderPriceModeID": 2,
	  "limitOrderPrice": 0,
	  "stopOrderPrice": 15710,
	  "hasIfDoneOrder": true,
	  "IDOIsGuarantee": false,
	  "IDOOrderModeID": 2,
	  "IDOLimitOrderPrice": 0,
	  "IDOStopOrderPrice": "15690.0"
	}

// sell response:

	{
	  "d": {
	    "__type": "TradingPlatform.OpenOrder",
	    "OrderID": "22838126",
	    "QuoteID": null,
	    "MarketID": null,
	    "Market": "Germany 40 - Rolling Cash",
	    "ExpiryDate": "31/12/30",
	    "TradeMode": "sell",
	    "Stake": "66.0",
	    "OrderMode": "Stop",
	    "OrderType": "GoodTillCancel",
	    "OrderPriceMode": "Quote",
	    "LimitOrderPrice": "0.0",
	    "StopOrderPrice": "15705.0",
	    "OrderStatus": null,
	    "IsForceOpen": false,
	    "IDOID": "22838126",
	    "IDOOrderMode": "Stop",
	    "IDOTradeMode": "buy",
	    "IDOIsGuaranteedStop": false,
	    "IDOLimitOrderPrice": "0",
	    "IDOStopOrderPrice": "15725",
	    "IDOTrailingPoint": null,
	    "Currency": "DMD",
	    "IsRollingMarket": false,
	    "Status": 0,
	    "Message": null
	  }
	}
*/
type InsertOpenOrderRequest struct {
	TradeType          int    `json:"tradeType"`
	MarketID           int    `json:"marketID"`
	MarketQuoteID      int    `json:"marketQuoteID"`
	TradeModeID        bool   `json:"tradeModeID"`
	OrderStake         string `json:"orderStake"`
	OrderModeID        int    `json:"orderModeID"`
	OrderTypeID        int    `json:"orderTypeID"`
	OrderPriceModeID   int    `json:"orderPriceModeID"`
	LimitOrderPrice    string `json:"limitOrderPrice"`
	StopOrderPrice     string `json:"stopOrderPrice"`
	HasIfDoneOrder     bool   `json:"hasIfDoneOrder"`
	IDOIsGuarantee     bool   `json:"IDOIsGuarantee"`
	IDOOrderModeID     int    `json:"IDOOrderModeID"`
	IDOLimitOrderPrice string `json:"IDOLimitOrderPrice"`
	IDOStopOrderPrice  string `json:"IDOStopOrderPrice"`
}

/*
	{
	  "d": {
	    "__type": "TradingPlatform.OpenOrder",
	    "OrderID": "22777380",
	    "QuoteID": null,
	    "MarketID": null,
	    "Market": "Germany 40 - Rolling Cash",
	    "ExpiryDate": "31/12/30",
	    "TradeMode": "buy",
	    "Stake": "1.0",
	    "OrderMode": "Limit",
	    "OrderType": "GoodTillCancel",
	    "OrderPriceMode": "Quote",
	    "LimitOrderPrice": "15600.0",
	    "StopOrderPrice": "0",
	    "OrderStatus": null,
	    "IsForceOpen": false,
	    "IDOID": "22777380",
	    "IDOOrderMode": "Both",
	    "IDOTradeMode": "sell",
	    "IDOIsGuaranteedStop": false,
	    "IDOLimitOrderPrice": "15700.0",
	    "IDOStopOrderPrice": "15570.0",
	    "IDOTrailingPoint": null,
	    "Currency": "DMD",
	    "IsRollingMarket": false,
	    "Status": 0,
	    "Message": null
	  }
	}
*/
type OpenOrder struct {
	Type                string      `json:"__type"`
	OrderID             string      `json:"OrderID"`
	QuoteID             interface{} `json:"QuoteID"`
	MarketID            interface{} `json:"MarketID"`
	Market              string      `json:"Market"`
	ExpiryDate          string      `json:"ExpiryDate"`
	TradeMode           string      `json:"TradeMode"`
	Stake               string      `json:"Stake"`
	OrderMode           string      `json:"OrderMode"`
	OrderType           string      `json:"OrderType"`
	OrderPriceMode      string      `json:"OrderPriceMode"`
	LimitOrderPrice     string      `json:"LimitOrderPrice"`
	StopOrderPrice      string      `json:"StopOrderPrice"`
	OrderStatus         string      `json:"OrderStatus"` // Active
	IsForceOpen         bool        `json:"IsForceOpen"`
	IDOID               string      `json:"IDOID"`
	IDOOrderMode        string      `json:"IDOOrderMode"`
	IDOTradeMode        string      `json:"IDOTradeMode"`
	IDOIsGuaranteedStop bool        `json:"IDOIsGuaranteedStop"`
	IDOLimitOrderPrice  string      `json:"IDOLimitOrderPrice"`
	IDOStopOrderPrice   string      `json:"IDOStopOrderPrice"`
	IDOTrailingPoint    string      `json:"IDOTrailingPoint"` // "0"
	Currency            string      `json:"Currency"`
	IsRollingMarket     bool        `json:"IsRollingMarket"`
	Status              int         `json:"Status"`
	Message             interface{} `json:"Message"`
}

type InsertOpenOrderResponse struct {
	D OpenOrder `json:"d"`
}

// POST	https://demo.tradedirect365.com/UTSAPI.asmx/GetOpenOrder
type GetOpenOrderRequest struct {
	OrderID int `json:"orderID"`
}
type GetOpenOrderResponse struct {
	D OpenOrder `json:"d"`
}

// POST /UTSAPI.asmx/DeleteOrder
type DeleteOrderRequest = GetOpenOrderRequest

type DeleteOrderResponse struct {
	D Result `json:"d"`
}

// POST	https://demo.tradedirect365.com/UTSAPI.asmx/AmendOpenOrder
/*
{
  "orderID": 22779355,
  "orderStake": "1",
  "orderModeID": 2,
  "orderTypeID": 2,
  "orderPriceModeID": 2,
  "limitOrderPrice": 0,
  "stopOrderPrice": 16000,
  "IDOAction": 1,
  "IDOIsGuarantee": false,
  "IDOOrderModeID": 2,
  "IDOLimitOrderPrice": 0,
  "IDOStopOrderPrice": "15990.0"
}
*/
type AmendOpenOrderRequest struct {
	OrderID            int    `json:"orderID"`
	OrderStake         string `json:"orderStake"`
	OrderModeID        int    `json:"orderModeID"`
	OrderTypeID        int    `json:"orderTypeID"`
	OrderPriceModeID   int    `json:"orderPriceModeID"`
	LimitOrderPrice    int    `json:"limitOrderPrice"`
	StopOrderPrice     int    `json:"stopOrderPrice"`
	IDOAction          int    `json:"IDOAction"`
	IDOIsGuarantee     bool   `json:"IDOIsGuarantee"`
	IDOOrderModeID     int    `json:"IDOOrderModeID"`
	IDOLimitOrderPrice int    `json:"IDOLimitOrderPrice"`
	IDOStopOrderPrice  string `json:"IDOStopOrderPrice"`
}

type AmmendOpenOrderResponse = InsertOpenOrderResponse

// https://demo.tradedirect365.com/UTSAPI.asmx/RequestTrade
/*
{
  "marketID": 17068,
  "quoteID": 6374,
  "price": 15921,
  "stake": "1",
  "tradeType": 1,
  "tradeMode": true,
  "hasClosingOrder": false,
  "isGuaranteed": false,
  "orderModeID": 0,
  "orderTypeID": 2,
  "orderPriceModeID": 0,
  "limitOrderPrice": 0,
  "stopOrderPrice": 0,
  "trailingPoint": 0,
  "closePositionID": 0,
  "isKaazingFeed": true,
  "userAgent": "Firefox (115.0)",
  "key": "+Cvmv+vetT2EdhPo15Wnp6MvVQXqqYWOWyV+h72pgj8="
}
*/
type RequestTradeRequest struct {
	MarketID         int    `json:"marketID"`
	QuoteID          int    `json:"quoteID"`
	Price            int    `json:"price"`
	Stake            string `json:"stake"`
	TradeType        int    `json:"tradeType"`
	TradeMode        bool   `json:"tradeMode"`
	HasClosingOrder  bool   `json:"hasClosingOrder"`
	IsGuaranteed     bool   `json:"isGuaranteed"`
	OrderModeID      int    `json:"orderModeID"`
	OrderTypeID      int    `json:"orderTypeID"`
	OrderPriceModeID int    `json:"orderPriceModeID"`
	LimitOrderPrice  int    `json:"limitOrderPrice"`
	StopOrderPrice   int    `json:"stopOrderPrice"`
	TrailingPoint    int    `json:"trailingPoint"`
	ClosePositionID  int    `json:"closePositionID"`
	IsKaazingFeed    bool   `json:"isKaazingFeed"`
	UserAgent        string `json:"userAgent"`
	Key              string `json:"key"`
}

/*
	{
	  "d": {
	    "__type": "TradingPlatform.TradeRequest",
	    "MarketID": 17068,
	    "Direction": "sell",
	    "Market": "Germany 40 - Rolling Cash",
	    "ExpiryDate": "31/12/30",
	    "Price": 15921.0,
	    "Stake": 1.0,
	    "TradeStatus": null,
	    "PositionID": 23093406,
	    "ReferralID": "0",
	    "CloseBets": null,
	    "OrderMode": "None",
	    "OrderType": "GoodTillCancel",
	    "StopOrderPrice": "0",
	    "LimitOrderPrice": "0",
	    "OrderID": "0",
	    "Status": 0,
	    "Message": null
	  }
*/
type RequestTradeResponse struct {
	D struct {
		Type            string      `json:"__type"`
		MarketID        int         `json:"MarketID"`
		Direction       string      `json:"Direction"`
		Market          string      `json:"Market"`
		ExpiryDate      string      `json:"ExpiryDate"`
		Price           float64     `json:"Price"`
		Stake           float64     `json:"Stake"`
		TradeStatus     interface{} `json:"TradeStatus"`
		PositionID      int         `json:"PositionID"`
		ReferralID      string      `json:"ReferralID"`
		CloseBets       interface{} `json:"CloseBets"`
		OrderMode       string      `json:"OrderMode"`
		OrderType       string      `json:"OrderType"`
		StopOrderPrice  string      `json:"StopOrderPrice"`
		LimitOrderPrice string      `json:"LimitOrderPrice"`
		OrderID         string      `json:"OrderID"`
		Status          int         `json:"Status"`
		Message         interface{} `json:"Message"`
	} `json:"d"`
}

// POST https://demo.tradedirect365.com/UTSAPI.asmx/GetOpenPositionDetails?AccountID=2365530
type GetOpenPositionDetailsRequest struct {
	PositionID int `json:"positionID"`
}

/*
	{
	  "d": {
	    "openPosition": {
	      "__type": "TradingPlatform.Position",
	      "MarketID": "17068",
	      "QuoteID": "6374",
	      "PositionID": "23093406",
	      "Direction": "sell",
	      "InitStake": "1",
	      "OpenPrice": "15921",
	      "MarketName": "Germany 40 - Rolling Cash",
	      "PrcGenFractionalPrice": "",
	      "PrcGenDecimalPlaces": "1",
	      "BetPer": "1.000000",
	      "MinCloseOrderDisTicks": "2.000000",
	      "GuranteedStopDisTicks": "10.000000",
	      "TransactionDate": "07/08/23 09:29:02",
	      "Bid": "0",
	      "Ask": "0",
	      "isGSLPercent": "1",
	      "GSLDis": "2.000000",
	      "Currency": "EUR",
	      "DisplayBetPer": "1.000000",
	      "AtQuoteAtMarket": 1,
	      "Status": 0,
	      "Message": null
	    },
	    "marketDetails": {
	      "__type": "TradingPlatform.Market",
	      "MarketID": 17068,
	      "QuoteID": 6374,
	      "AtQuoteAtMarket": 1,
	      "ExchangeID": 155,
	      "PrcGenFractionalPrice": 0,
	      "PrcGenDecimalPlaces": 1,
	      "High": 0,
	      "Low": 0,
	      "DailyChange": 0,
	      "Bid": 0,
	      "Ask": 0,
	      "BetPer": 1.0,
	      "IsGSLPercent": 1,
	      "GSLDis": 2.0,
	      "MinCloseOrderDisTicks": 2.0,
	      "MinOpenOrderDisTicks": 2.0,
	      "DisplayBetPer": 1.0,
	      "IsInPortfolio": false,
	      "Tradable": true,
	      "TradeOnWeb": true,
	      "CallOnly": false,
	      "MarketName": "Germany 40 - Rolling Cash",
	      "TradeStartTime": "",
	      "Currency": "EUR",
	      "AllowGtdsStops": 1,
	      "ForceOpen": true,
	      "Margin": 0.5,
	      "MarginType": false,
	      "GSLCharge": 3.0,
	      "IsGSLChargePercent": 0,
	      "Spread": 6.0,
	      "TradeRateType": 0,
	      "OpenTradeRate": 0.0,
	      "CloseTradeRate": 0.0,
	      "MinOpenTradeRate": 0.0,
	      "MinCloseTradeRate": 0.0,
	      "PriceDecimal": 1.0,
	      "Subscription": false,
	      "SuperGroupID": 1
	    },
	    "webInfo": {
	      "__type": "TradingPlatform.ClientWebOptionInfo",
	      "CFDDefaultStake": 1.0,
	      "IsDealAlwayHedge": false,
	      "IsDealAlwayGuarantee": false,
	      "IsOneClickTrade": false,
	      "IsOrderAlwayHedge": false,
	      "IsOrderAlwayGuarantee": false,
	      "StopTypeID": 1,
	      "TradeOrderTypeID": 2,
	      "DealDefaultStake": 1.0,
	      "OrderDefaultStake": 1.0,
	      "WebMinStake": 0.5,
	      "WebMaxStake": 5000.0
	    }
	  }
	}
*/
type GetOpenPositionDetailsResponse struct {
	D struct {
		OpenPosition struct {
			Type                  string      `json:"__type"`
			MarketID              string      `json:"MarketID"`
			QuoteID               string      `json:"QuoteID"`
			PositionID            string      `json:"PositionID"`
			Direction             string      `json:"Direction"`
			InitStake             string      `json:"InitStake"`
			OpenPrice             string      `json:"OpenPrice"`
			MarketName            string      `json:"MarketName"`
			PrcGenFractionalPrice string      `json:"PrcGenFractionalPrice"`
			PrcGenDecimalPlaces   string      `json:"PrcGenDecimalPlaces"`
			BetPer                string      `json:"BetPer"`
			MinCloseOrderDisTicks string      `json:"MinCloseOrderDisTicks"`
			GuranteedStopDisTicks string      `json:"GuranteedStopDisTicks"`
			TransactionDate       string      `json:"TransactionDate"`
			Bid                   string      `json:"Bid"`
			Ask                   string      `json:"Ask"`
			IsGSLPercent          string      `json:"isGSLPercent"`
			GSLDis                string      `json:"GSLDis"`
			Currency              string      `json:"Currency"`
			DisplayBetPer         string      `json:"DisplayBetPer"`
			AtQuoteAtMarket       int         `json:"AtQuoteAtMarket"`
			Status                int         `json:"Status"`
			Message               interface{} `json:"Message"`
		} `json:"openPosition"`
		MarketDetails struct {
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
			GSLCharge             float64 `json:"GSLCharge"`
			IsGSLChargePercent    int     `json:"IsGSLChargePercent"`
			Spread                float64 `json:"Spread"`
			TradeRateType         int     `json:"TradeRateType"`
			OpenTradeRate         float64 `json:"OpenTradeRate"`
			CloseTradeRate        float64 `json:"CloseTradeRate"`
			MinOpenTradeRate      float64 `json:"MinOpenTradeRate"`
			MinCloseTradeRate     float64 `json:"MinCloseTradeRate"`
			PriceDecimal          float64 `json:"PriceDecimal"`
			Subscription          bool    `json:"Subscription"`
			SuperGroupID          int     `json:"SuperGroupID"`
		} `json:"marketDetails"`
		WebInfo struct {
			Type                  string  `json:"__type"`
			CFDDefaultStake       float64 `json:"CFDDefaultStake"`
			IsDealAlwayHedge      bool    `json:"IsDealAlwayHedge"`
			IsDealAlwayGuarantee  bool    `json:"IsDealAlwayGuarantee"`
			IsOneClickTrade       bool    `json:"IsOneClickTrade"`
			IsOrderAlwayHedge     bool    `json:"IsOrderAlwayHedge"`
			IsOrderAlwayGuarantee bool    `json:"IsOrderAlwayGuarantee"`
			StopTypeID            int     `json:"StopTypeID"`
			TradeOrderTypeID      int     `json:"TradeOrderTypeID"`
			DealDefaultStake      float64 `json:"DealDefaultStake"`
			OrderDefaultStake     float64 `json:"OrderDefaultStake"`
			WebMinStake           float64 `json:"WebMinStake"`
			WebMaxStake           float64 `json:"WebMaxStake"`
		} `json:"webInfo"`
	} `json:"d"`
}

// POST https://demo.tradedirect365.com/UTSAPI.asmx/InsertClosePosition
/*
{
  "marketID": 17068,
  "positionID": 23093406,
  "quoteID": 6374,
  "price": "15921.9",
  "stake": 1,
  "tradeMode": false,
  "isKaazingFeed": true,
  "userAgent": "Firefox (115.0)",
  "key": "k//pF64C6cwNcMfo0oHEZ5FDyiQmySiNc5w4ng2Mptc="
}

*/
type InsertClosePositionRequest struct {
	MarketID      int    `json:"marketID"`
	PositionID    int    `json:"positionID"`
	QuoteID       int    `json:"quoteID"`
	Price         string `json:"price"`
	Stake         string `json:"stake"`
	TradeMode     bool   `json:"tradeMode"`
	IsKaazingFeed bool   `json:"isKaazingFeed"`
	UserAgent     string `json:"userAgent"`
	Key           string `json:"key"`
}

/*
	{
	  "d": {
	    "__type": "TradingPlatform.TradeRequest",
	    "MarketID": 17068,
	    "Direction": "buy",
	    "Market": "Germany 40 - Rolling Cash",
	    "ExpiryDate": "31/12/30",
	    "Price": 15921.9,
	    "Stake": 1.0,
	    "TradeStatus": null,
	    "PositionID": 23093406,
	    "ReferralID": "0",
	    "CloseBets": {
	      "ProfitLoss": null,
	      "ClosedBet": [
	        {
	          "ReferenceNo": null,
	          "OpenPrice": "15921.0",
	          "ProfitLoss": "-0.9"
	        }
	      ]
	    },
	    "OrderMode": "",
	    "OrderType": "",
	    "StopOrderPrice": "",
	    "LimitOrderPrice": "",
	    "OrderID": "",
	    "Status": 0,
	    "Message": null
	  }
	}
*/
type InsertClosePositionResponse struct {
	D struct {
		Type        string      `json:"__type"`
		MarketID    int         `json:"MarketID"`
		Direction   string      `json:"Direction"`
		Market      string      `json:"Market"`
		ExpiryDate  string      `json:"ExpiryDate"`
		Price       float64     `json:"Price"`
		Stake       float64     `json:"Stake"`
		TradeStatus interface{} `json:"TradeStatus"`
		PositionID  int         `json:"PositionID"`
		ReferralID  string      `json:"ReferralID"`
		CloseBets   struct {
			ProfitLoss interface{} `json:"ProfitLoss"`
			ClosedBet  []struct {
				ReferenceNo interface{} `json:"ReferenceNo"`
				OpenPrice   string      `json:"OpenPrice"`
				ProfitLoss  string      `json:"ProfitLoss"`
			} `json:"ClosedBet"`
		} `json:"CloseBets"`
		OrderMode       string      `json:"OrderMode"`
		OrderType       string      `json:"OrderType"`
		StopOrderPrice  string      `json:"StopOrderPrice"`
		LimitOrderPrice string      `json:"LimitOrderPrice"`
		OrderID         string      `json:"OrderID"`
		Status          int         `json:"Status"`
		Message         interface{} `json:"Message"`
	} `json:"d"`
}

type AmendCloseOrderRequest struct {
	Market           string `json:"market"`
	OrderID          int    `json:"orderID"`
	OrderStake       string `json:"orderStake"`
	OrderModeID      int    `json:"orderModeID"`
	OrderTypeID      int    `json:"orderTypeID"`
	OrderPriceModeID int    `json:"orderPriceModeID"`
	LimitOrderPrice  int    `json:"limitOrderPrice"`
	StopOrderPrice   string `json:"stopOrderPrice"`
	TrailingPoint    int    `json:"trailingPoint"`
	IsGuaranteed     bool   `json:"isGuaranteed"`
}
