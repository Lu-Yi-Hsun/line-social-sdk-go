package apiservice

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (client *Client) GetAccessToken(redirectURL, code string) *GetAccessTokenCall {
	return &GetAccessTokenCall{
		c:           client,
		redirectURL: redirectURL,
		code:        code,
	}
}

// GetAccessTokenCall type
type GetAccessTokenCall struct {
	c   *Client
	ctx context.Context

	redirectURL string
	code        string
}

// WithContext method
func (call *GetAccessTokenCall) WithContext(ctx context.Context) *GetAccessTokenCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetAccessTokenCall) Do() (*TokenResponse, error) {
	buf := strings.NewReader(fmt.Sprintf("grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s&client_secret=%s", call.code, call.redirectURL, call.c.channelID, call.c.channelSecret))
	res, err := call.c.post(call.ctx, APIEndpointToken, buf)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return decodeToTokenResponse(res)
}

type Payload struct {
	Iss     string `json:"iss"`
	Sub     string `json:"sub"`
	Aud     string `json:"aud"`
	Exp     int    `json:"exp"`
	Iat     int    `json:"iat"`
	Nonce   string `json:"nonce"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// GetWebLoinURL - LINE LOGIN 2.1 get LINE Login URL
func (client *Client) GetWebLoinURL(redirectURL, state, scope, nounce, chatbotPrompt string) string {
	req, err := http.NewRequest("GET", path.Join(APIEndpointBase, APIEndpointAuthorize), nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	q := req.URL.Query()
	q.Add("response_type", "code")
	q.Add("client_id", client.channelID)
	q.Add("state", state)
	q.Add("scope", scope)
	q.Add("nounce", nounce)
	q.Add("redirect_uri", redirectURL)
	if len(chatbotPrompt) > 0 {
		q.Add("bot_prompt", chatbotPrompt)
	}
	q.Add("prompt", "consent")
	req.URL.RawQuery = q.Encode()
	log.Println(req.URL.String())
	return req.URL.String()
}

func GenerateNounce() string {
	return b64.StdEncoding.EncodeToString([]byte(randStringRunes(8)))
}

func DecodeIDToken(idToken string, channelID string) (*Payload, error) {
	splitToken := strings.Split(idToken, ".")
	if len(splitToken) < 3 {
		log.Println("Error: idToken size is wrong, size=", len(splitToken))
		return nil, fmt.Errorf("Error: idToken size is wrong. \n")
	}
	header, payload, signature := splitToken[0], splitToken[1], splitToken[2]
	log.Println("result:", header, payload, signature)

	log.Println("side of payload=", len(payload))
	payload = base64Decode(payload)
	log.Println("side of payload=", len(payload), payload)
	bPayload, err := b64.StdEncoding.DecodeString(payload)
	if err != nil {
		log.Println("base64 decode err:", err)
		return nil, fmt.Errorf("Error: base64 decode. \n")
	}
	log.Println("base64 decode succeess:", string(bPayload))

	retPayload := &Payload{}
	if err := json.Unmarshal(bPayload, retPayload); err != nil {
		return nil, fmt.Errorf("json unmarshal error, %v. \n", err)
	}

	// payload verification
	if strings.Compare(retPayload.Iss, "https://access.line.me") != 0 {
		return nil, fmt.Errorf("Payload verification wrong. Wrong issue organization. \n")
	}
	if strings.Compare(retPayload.Aud, channelID) != 0 {
		return nil, fmt.Errorf("Payload verification wrong. Wrong audience. \n")
	}

	return retPayload, nil
}

func base64Decode(payload string) string {
	rem := len(payload) % 4
	log.Println("rem of payload=", rem)
	if rem > 0 {
		i := 4 - rem
		for ; i > 0; i-- {
			payload = payload + "="
		}
	}
	return payload
}
