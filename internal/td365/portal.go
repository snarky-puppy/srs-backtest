package td365

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

const (
	DemoAccountId = 2365530
	ProdSite      = "traders.td365.com"
	PortalSite    = "portal-api.tradenation.com"
)

func ProdUrl(path string) string {
	return "https://" + ProdSite + path
}

func PortalUrl(path string) string {
	return "https://" + PortalSite + path
}

type TradeManager interface {
	HandleTick(tick *models.Tick)
	PositionOpened(trade *models.Trade)
	PositionClosed(trade *models.Trade)
	Backfill(symbol models.Symbol, bars models.Series)
}

type Portal struct {
	client           *http.Client
	uat              *UserAgentTransport
	formData         FormData
	connection       *ConnectionProxy
	marketSuperGroup MarketGroup
	PopularQuotes    PopularQuotes
	token            Token
	idToken          IdToken
	Account          Account
	OriginUrl        string
	apiUrl           string
}

func LaunchPlatform(ctx context.Context, accountId int, tradeManager TradeManager) *Platform {
	var uat *UserAgentTransport
	portal := &Portal{
		client: &http.Client{
			Jar: func() http.CookieJar {
				cookieJar, err := cookiejar.New(nil)
				if err != nil {
					panic(err)
				}
				return cookieJar
			}(),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				log.Println("Redirected:", req.URL)
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
		Transport: portal.client.Transport,
		UserAgent: UserAgent,
	}
	portal.uat = uat
	portal.client.Transport = uat

	portal.Login()
	uat.Authorization = "Bearer " + portal.token.AccessToken
	portal.postLogin()
	portal.Account = portal.getAccount(accountId)

	portal.apiUrl, portal.OriginUrl = ApiUrl(portal.Account.AccountType)

	loginUrl := portal.fetchPortalLoginUrl()
	portal.launchPlatform(loginUrl)

	//portal.connection = NewConnectionProxy(
	//	portal.Account.AccountType,
	//	portal.Account.CtLoginId,
	//	portal.Account.Platform)

	//return portal
	return newPlatform(portal.client, uat, portal.Account, tradeManager)
}

func (p *Portal) Login() {

	login := os.Getenv("TDLOGIN")
	password := os.Getenv("TDPASSWD")
	if login == "" || password == "" {
		panic("TDLOGIN and TDPASSWD must be set")
	}

	loginRequestStr := `{"realm":"Username-Password-Authentication","client_id":"eeXrVwSMXPZ4pJpwStuNyiUa7XxGZRX9","scope":"openid","grant_type":"http://auth0.com/oauth/grant-type/password-realm","username":"%s","password":"%s"}`
	loginRequestStr = fmt.Sprintf(loginRequestStr, login, password)

	p.uat.Referer = ProdUrl("")
	p.uat.Origin = p.uat.Referer

	resp, err := p.client.Post("https://td365.eu.auth0.com/oauth/token",
		"application/json",
		strings.NewReader(loginRequestStr))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	p.token = NewToken(resp.Body)
	p.idToken = p.token.GetIdToken()
}

func (p *Portal) postLogin() {
	resp, err := p.client.Post(PortalUrl(fmt.Sprintf("/TD365/user/%d/login/", p.idToken.HttpsFinsaId)), "application/json", nil)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

}

func (p *Portal) getAccount(accountId int) Account {
	resp, err := p.client.Get(PortalUrl(fmt.Sprintf("/TD365/user/%d/accounts/", p.idToken.HttpsFinsaId)))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// debug resp.Body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println(string(body))

	var result AccountsResult
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&result)
	if err != nil {
		panic(err)
	}

	for _, a := range result.Results {
		log.Println("account", a.Id, a.AccountType, a.Account, a.Balance)
	}

	for _, a := range result.Results {
		if a.Id == accountId {
			return a
		}
	}

	panic("account not found")
}

func (p *Portal) fetchPortalLoginUrl() string {
	resp, err := p.client.Get(PortalUrl(p.Account.Button.LinkTo))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var response struct {
		Url string `json:"url"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		panic(err)
	}
	return response.Url
}

func (p *Portal) launchPlatform(url string) {
	log.Println("launchPlatform", url)
	resp, err := p.client.Get(url)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
}
