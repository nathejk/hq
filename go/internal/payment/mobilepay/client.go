package mobilepay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Auth struct {
	ClientID        string
	ClientSecret    string
	SubscriptionKey string
	SerialNumber    string
}

type Client interface {
	CreatePayment(key string, p Payment) (*CreatePaymentResponse, error)
	CapturePayment(ref PaymentReference, key string, amount Amount) (*ModificationResponse, error)
}

type client struct {
	baseurl     string
	auth        Auth
	accessToken string
	http        *http.Client
}

func New(dsn string) (*client, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	secret, _ := u.User.Password()
	c := &client{
		baseurl: fmt.Sprintf("https://%s", u.Host),
		auth: Auth{
			ClientID:        u.User.Username(),
			ClientSecret:    secret,
			SubscriptionKey: u.RawQuery,
			SerialNumber:    u.Fragment,
		},
		http: httpClient(),
	}
	return c, nil
}

type AccessTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    string `json:"expires_in"`
	ExtExpiresIn string `json:"ext_expires_in"`
	AccessToken  string `json:"access_token"`
	ExpiresOn    string `json:"expires_on"`
	NotBefore    string `json:"not_before"`
	Resource     string `json:"resource"`
}

func (c *client) GetAccessToken() (*AccessTokenResponse, error) {
	requestURL := fmt.Sprintf("%s/accesstoken/get", c.baseurl)
	req, err := http.NewRequest(http.MethodPost, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("client_id", c.auth.ClientID)
	req.Header.Add("client_secret", c.auth.ClientSecret)
	req.Header.Add("Ocp-Apim-Subscription-Key", c.auth.SubscriptionKey)
	req.Header.Add("Merchant-Serial-Number", c.auth.SerialNumber)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	var body AccessTokenResponse
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	return &body, nil
}

type ModificationRequest struct {
	Amount Amount `json:"modificationAmount"`
}
type ModificationResponse struct {
	Amount    Amount `json:"amount"`
	State     string `json:"state"`
	Aggregate struct {
		AuthorizedAmount Amount `json:"authorizedAmount"`
		CancelledAmount  Amount `json:"cancelledAmount"`
		CapturedAmount   Amount `json:"capturedAmount"`
		RefundedAmount   Amount `json:"refundedAmount"`
	} `json:"aggregate"`
	PspReference string           `json:"pspReference"`
	Reference    PaymentReference `json:"reference"`
}

type CreatePaymentResponse struct {
	RedirectUrl string           `json:"redirectUrl"`
	Reference   PaymentReference `json:"reference"`
}

func (c *client) CreatePayment(key string, p Payment) (*CreatePaymentResponse, error) {
	jsonBody, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	requestURL := fmt.Sprintf("%s/epayment/v1/payments", c.baseurl)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	token, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}
	c.accessToken = token.AccessToken
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Idempotency-Key", key)
	req.Header.Add("Ocp-Apim-Subscription-Key", c.auth.SubscriptionKey)
	req.Header.Add("Merchant-Serial-Number", c.auth.SerialNumber)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	var body CreatePaymentResponse
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	return &body, nil
}

func (c *client) GetPayment(ref PaymentReference) (*CreatePaymentResponse, error) {
	return nil, nil
}
func (c *client) CancelPayment(ref PaymentReference, cancelTransactionOnly bool) (*CreatePaymentResponse, error) {
	return nil, nil
}

func (c *client) CapturePayment(ref PaymentReference, key string, amount Amount) (*ModificationResponse, error) {
	jsonBody, err := json.Marshal(ModificationRequest{Amount: amount})
	if err != nil {
		return nil, err
	}

	requestURL := fmt.Sprintf("%s/epayment/v1/payments/%s/capture", c.baseurl, ref)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	token, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}
	c.accessToken = token.AccessToken
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Idempotency-Key", key)
	req.Header.Add("Ocp-Apim-Subscription-Key", c.auth.SubscriptionKey)
	req.Header.Add("Merchant-Serial-Number", c.auth.SerialNumber)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	var body ModificationResponse
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	return &body, nil
}
func (c *client) RefundPayment(ref PaymentReference) (*CreatePaymentResponse, error) {
	return nil, nil
}

/**
 * This endpoint is only available in the test environment. It allows developers to approve a payment
 * through the ePayment API without the use of the Vipps or MobilePay app. This is useful for automated
 * testing. Express checkout is not supported for this endpoint. Attempted use in production is not
 * allowed, and will fail. Important: All test users must manually approve at least one payment in the
 * Vipps or MobilePay app before this endpoint can be used for that user.
 */
func (c *client) ForceApprovePayment(ref PaymentReference) (*CreatePaymentResponse, error) {
	return nil, nil
}
