package mobilepay

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ApiClient interface {
	Transactions(size, page int) (*TransactionResponse, error)
	Transfers(size, page int) (*TransferResponse, error)
	PaymentPoints(size, page int) (*PaymentPointResponse, error)
}

type client struct {
	url string
	key string
}

func NewApiClient(url, key string) *client {
	return &client{
		url: url,
		key: key,
	}
}
func (c *client) request(route string, body interface{}) error {
	req, _ := http.NewRequest("GET", c.url+route, nil)
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(&body)
}

func (c *client) Transactions(size, page int) (*TransactionResponse, error) {
	route := fmt.Sprintf("/v3/reporting/transactions?pagesize=%d&pagenumber=%d", size, page)
	body := TransactionResponse{}

	if err := c.request(route, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *client) Transfers(size, page int) (*TransferResponse, error) {
	route := fmt.Sprintf("/v3/reporting/transfers?pagesize=%d&pagenumber=%d", size, page)
	body := TransferResponse{}

	if err := c.request(route, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *client) PaymentPoints(size, page int) (*PaymentPointResponse, error) {
	route := fmt.Sprintf("/v3/reporting/paymentpoints?pagesize=%d&pagenumber=%d", size, page)
	body := PaymentPointResponse{}

	if err := c.request(route, &body); err != nil {
		return nil, err
	}
	return &body, nil
}
