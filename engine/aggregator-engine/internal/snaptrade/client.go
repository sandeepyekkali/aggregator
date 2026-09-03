package snaptrade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	clientID    string
	consumerKey string
	baseURL     string
	httpClient  *http.Client
}

func NewClient(clientID, consumerKey string) *Client {
	return &Client{
		clientID:    clientID,
		consumerKey: consumerKey,
		baseURL:     "https://api.snaptrade.com/api/v1",
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) RegisterUser(ctx context.Context, userID string) (string, error) {
	url := fmt.Sprintf("%s/snapTrade/registerUser?clientId=%s&consumerKey=%s", c.baseURL, c.clientID, c.consumerKey)
	payload := map[string]string{"userId": userID}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error registering user: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("snaptrade API returned status: %d", res.StatusCode)
	}

	var result struct {
		UserSecret string `json:"userSecret"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.UserSecret, nil
}

func (c *Client) GenerateLoginLink(ctx context.Context, userID, userSecret string) (string, error) {
	url := fmt.Sprintf("%s/snapTrade/login?clientId=%s&consumerKey=%s", c.baseURL, c.clientID, c.consumerKey)
	payload := map[string]string{
		"userId":     userID,
		"userSecret": userSecret,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error generating link: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("snaptrade API returned status: %d", res.StatusCode)
	}

	var result struct {
		RedirectURI string `json:"redirectURI"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.RedirectURI, nil
}
