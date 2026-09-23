package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

type APIResource interface {
	Endpoint() string
}

// StatusError is a non-2xx response from the API.
//
// The status code is carried as a field rather than left to be recovered from
// the message, because the message embeds the response body: a body that happens
// to contain "status: 404" would otherwise read as a 404, and the branches that
// ask are the ones that drop a resource from Terraform state.
//
// Error() keeps the format the API layer has always produced. Diagnostics
// surface it to operators verbatim, so the wording is part of the interface.
type StatusError struct {
	Status int
	Body   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("status: %d, body: %s", e.Status, e.Body)
}

// HasStatus reports whether err is an API status error carrying the given code.
func HasStatus(err error, status int) bool {
	var statusErr *StatusError
	return errors.As(err, &statusErr) && statusErr.Status == status
}

// IsNotFound reports whether err represents a 404 from the API. Callers use it
// to treat a resource as deleted (e.g. remove it from Terraform state) rather
// than surfacing a hard error.
func IsNotFound(err error) bool {
	return HasStatus(err, http.StatusNotFound)
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

// ListItemsEndpoint performs a GET against a specific endpoint and returns the
// result as a slice. The group-policy membership read endpoints document a
// single JSON object in the spec, but the appliance has been observed
// returning a JSON array for at least one of them — so this decodes whichever
// shape the endpoint actually returns (a lone object becomes a one-element
// slice). Returns an empty slice on a 204/no-content response.
func ListItemsEndpoint[I APIResource](c *APIClient, endpoint string) ([]I, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}

	items := []I{}
	arrErr := json.Unmarshal(body, &items)
	if arrErr == nil {
		return items, nil
	}

	// Only fall back to single-object decode when the top-level value isn't an
	// array at all (encoding/json reports that as an UnmarshalTypeError whose
	// Value is "object" and, because the error is at the root, Field is
	// empty). If the body is an array but one of its elements fails to
	// decode, arrErr is the more useful diagnostic; the single-object decode
	// below would otherwise mask it behind a confusing "cannot unmarshal
	// object into Go value of type ..." error. Value alone is not enough to
	// tell the two apart: an array element that is itself object-shaped where
	// a scalar field was expected also reports Value == "object", but with
	// Field naming that element's field — checking Field == "" excludes it.
	var typeErr *json.UnmarshalTypeError
	if !errors.As(arrErr, &typeErr) || typeErr.Value != "object" || typeErr.Field != "" {
		return nil, arrErr
	}

	// Endpoint returned a single object rather than an array; decode it as one.
	var single I
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	return []I{single}, nil
}

// CreateItem POSTs item to its endpoint.
//
// It returns (nil, nil) when the API answers 204 No Content: the item WAS
// created, there is simply no body to decode. Callers that dereference or
// store the result must handle a nil item — writing it straight into Terraform
// state produces a null attribute and an "inconsistent result after apply".
// Echo the request back instead; the server accepted it.
//
// Note the asymmetry with UpdateItemEndpoint, which does not check for an
// empty body: a 204 on PATCH surfaces as "unexpected end of JSON input"
// rather than (nil, nil).
func CreateItem[I APIResource](c *APIClient, item I) (*I, error) {
	rb, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	// Endpoint and size only — see the logging policy in api/logging.go.
	c.LogString("✅ CreateItem [%s]: %d byte payload", item.Endpoint(), len(rb))

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
