package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type APIResource interface {
	id() *int
	endpoint() string
}

func ListItems[I APIResource](c *APIClient) ([]I, error) {
	var tmp I
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", c.BaseURL, tmp.endpoint()), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	items := []I{}
	err = json.Unmarshal(body, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func GetItem[I APIResource](c *APIClient, id int) (*I, error) {
	var item I
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%d", c.BaseURL, item.endpoint(), id), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func CreateItem[I APIResource](c *APIClient, item I) (*I, error) {
	rb, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	var newItem I
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, newItem.endpoint()), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &newItem)
	if err != nil {
		return nil, err
	}

	return &newItem, nil
}

func UpdateItem[I APIResource](c *APIClient, item I) (*I, error) {
	rb, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/%s/%d", c.BaseURL, item.endpoint(), *item.id()), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var newItem I
	err = json.Unmarshal(body, &newItem)
	if err != nil {
		return nil, err
	}

	return &newItem, nil
}

func DeleteItem[I APIResource](c *APIClient, id int) error {
	var tmp I
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s/%d", c.BaseURL, tmp.endpoint(), id), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
