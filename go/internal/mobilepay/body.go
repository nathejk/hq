package mobilepay

import (
	"time"
)

type TransactionType string

const (
	TransactionTypePayment  TransactionType = "Payment"
	TransactionTypeTransfer TransactionType = "Transfer"
)

type PaymentPoint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ProductType string `json:"productType"`
	Alias       string `json:"alias"`
}

type Transaction struct {
	PaymentID            *string         `json:"paymentId,omitempty"`
	Type                 TransactionType `json:"type"`
	Amount               float32         `json:"amount"`
	Currency             string          `json:"currencyCode,omitempty"`
	Timestamp            time.Time       `json:"timestamp"`
	Message              string          `json:"message,omitempty"`
	MerchantReference    *string         `json:"merchantReference,omitempty"`
	MerchantPaymentLabel *string         `json:"merchantPaymentLabel,omitempty"`
	TransferReference    string          `json:"transferReference"`
	TransferDate         string          `json:"transferDate"`
	UserPhoneNumber      string          `json:"userPhoneNumber,omitempty"`
	UserName             string          `json:"userName,omitempty"`
	LoyaltyID            *string         `json:"loyaltyId,omitempty"`
	MyShopNumber         string          `json:"myShopNumber,omitempty"`
	BrandName            *string         `json:"brandName,omitempty"`
	BrandID              *string         `json:"brandId,omitempty"`
	LocationID           *string         `json:"locationId,omitempty"`
	PosName              *string         `json:"posName,omitempty"`
	BeaconID             *string         `json:"beaconId,omitempty"`
	AgreementExternalID  *string         `json:"agreementExternalId,omitempty"`
	AgreementID          *string         `json:"agreementId,omitempty"`
	RefundID             *string         `json:"refundId,omitempty"`
}

type Transfer struct {
	ID                     string  `json:"id"`
	PaymentPointID         string  `json:"paymentPointId"`
	Reference              string  `json:"reference"`
	Date                   string  `json:"date"`
	TotalTransferredAmount float32 `json:"totalTransferredAmount"`
	Currency               string  `json:"currencyCode"`
}

type TransactionResponse struct {
	PageSize       int           `json:"pageSize"`
	NextPageNumber int           `json:"nextPageNumber"`
	Transactions   []Transaction `json:"transactions"`
}

type PaymentPointResponse struct {
	PageSize       int            `json:"pageSize"`
	NextPageNumber int            `json:"nextPageNumber"`
	PaymentPoints  []PaymentPoint `json:"paymentPoints"`
}

type TransferResponse struct {
	PageSize       int        `json:"pageSize"`
	NextPageNumber int        `json:"nextPageNumber"`
	Transfers      []Transfer `json:"transfers"`
}
