package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2/clientcredentials"
)

type APIClient struct {
	RootURL    string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(host string, client_id *string, client_secret *string) (*APIClient, error) {
	hostURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	if hostURL.Scheme == "" {
		hostURL.Scheme = "https"
	}

	config := clientcredentials.Config{
		ClientID:     *client_id,
		ClientSecret: *client_secret,
		TokenURL:     hostURL.String() + "/oauth2/token",
	}
	ctx := context.Background()
	c := APIClient{
		HTTPClient: config.Client(ctx),
		RootURL:    hostURL.String(),
		BaseURL:    hostURL.String() + "/api/config/v1",
	}

	_, err = config.Token(ctx)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *APIClient) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", "SRA-Terraform-Plugin")
	req.Header.Set("Accept", "application/json")
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

	if res.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	return body, err
}
