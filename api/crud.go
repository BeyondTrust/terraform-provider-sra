package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

type APIResource interface {
	Endpoint() string
}

func Get[I APIResource](c *APIClient) (*I, error) {
	var item I
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.RootURL, item.Endpoint()), nil)
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

func Post[I APIResource](c *APIClient, path string, item I, ignoreReturn bool) (*I, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", c.BaseURL, item.Endpoint(), path), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if ignoreReturn {
		return nil, nil
	}

	err = json.Unmarshal(body, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func ListItems[I APIResource](c *APIClient, query ...map[string]string) ([]I, error) {
	var tmp I
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.BaseURL, tmp.Endpoint()), nil)
	if err != nil {
		return nil, err
	}

	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query[0] {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	items := []I{}
	err = json.Unmarshal(resp, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func GetItem[I APIResource](c *APIClient, id *int) (*I, error) {
	var item I
	endpoint := item.Endpoint()
	if id != nil {
		endpoint = fmt.Sprintf("%s/%d", endpoint, *id)
	}

	return GetItemEndpoint[I](c, endpoint)
}

func GetItemEndpoint[I APIResource](c *APIClient, endpoint string) (*I, error) {
	var item I
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
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
	c.LogString("🎯 CreateItem pre-marshalling: %+v", item)
	rb, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	c.LogString("✅ CreateItem payload: %s", string(rb))

	var newItem I
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s", c.BaseURL, item.Endpoint()), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if body == nil {
		// success, but no content (204)
		return nil, nil
	}

	err = json.Unmarshal(body, &newItem)
	if err != nil {
		return nil, err
	}

	return &newItem, nil
}

func UpdateItem[I APIResource](c *APIClient, item I) (*I, error) {
	itemObj := reflect.ValueOf(item)
	id := itemObj.FieldByName("ID").Elem().Int()
	endpoint := fmt.Sprintf("%s/%d", item.Endpoint(), id)

	return UpdateItemEndpoint(c, item, endpoint)
}
func UpdateItemEndpoint[I APIResource](c *APIClient, item I, endpoint string) (*I, error) {
	rb, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), strings.NewReader(string(rb)))
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

func DeleteItem[I APIResource](c *APIClient, id *int) error {
	var tmp I
	endpoint := tmp.Endpoint()
	if id != nil {
		endpoint = fmt.Sprintf("%s/%d", endpoint, *id)
	}

	return DeleteItemEndpoint[I](c, endpoint)
}
func DeleteItemEndpoint[I APIResource](c *APIClient, endpoint string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
