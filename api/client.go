package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(host string, client_id, client_secret *string) (*APIClient, error) {
	config := clientcredentials.Config{
		ClientID:     *client_id,
		ClientSecret: *client_secret,
		TokenURL:     "https://" + host + "/oauth2/token",
	}
	c := APIClient{
		HTTPClient: config.Client(context.Background()),
		BaseURL:    "https://" + host + "/api/config/v1",
	}

	_, err := config.Token(context.Background())

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *APIClient) doRequest(req *http.Request) ([]byte, error) {
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}
