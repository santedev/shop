package paypal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"shop/config"
	"strings"
	"time"
)

func getAcessToken() (*accessToken, error) {
	var accessToken *accessToken
	paypal := getPaypal()
	if paypal.accessToken != nil && !time.Now().After(paypal.accessToken.ExpiresIn) {
		accessToken = paypal.accessToken
		return accessToken, nil
	}
	accessToken, err := fetchAccessToken()
	if err != nil {
		return nil, fmt.Errorf("could not get access token: %w", err)
	}
	paypal.accessToken = accessToken
	return accessToken, nil
}

func fetchAccessToken() (*accessToken, error) {
	clientID := config.Envs.PaypalKey
	clientSecret := config.Envs.PaypalSecret

	data := "grant_type=client_credentials"
	reqBody := strings.NewReader(data)

	auth := clientID + ":" + clientSecret
	authHeaderValue := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	req, err := http.NewRequest("POST", accessTokenURL(), reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", authHeaderValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	at := &accessToken{}
	err = json.Unmarshal(body, at)
	if err != nil {
		return nil, err
	}
	at.ExpiresIn = time.Now().Add(at.ExpiresInInt * time.Second)
	return at, nil
}
