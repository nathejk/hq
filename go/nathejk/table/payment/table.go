package payment

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type Operation struct {
	Type   types.PaymentStatus `json:"type"`
	Amount int                 `json:"amount"`
	Time   string              `json:"time"`
}

type OperationList []Operation

func (o *OperationList) Scan(value interface{}) error {
	if value == nil {
		*o = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("OperationList.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, o)
}

func (o OperationList) Value() (driver.Value, error) {
	if o == nil {
		return "[]", nil
	}
	b, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

type Payment struct {
	Reference       string              `json:"reference" db:"reference"`
	Year            string              `json:"year" db:"year"`
	ReceiptEmail    types.EmailAddress  `json:"receiptEmail" db:"receiptEmail"`
	ReturnUrl       string              `json:"returnUrl" db:"returnUrl"`
	Currency        types.Currency      `json:"currency" db:"currency"`
	Amount          int                 `json:"amount" db:"amount"`
	Method          string              `json:"method" db:"method"`
	Status          types.PaymentStatus `json:"status" db:"status"`
	CreatedAt       string              `json:"createdAt" db:"createdAt"`
	ChangedAt       string              `json:"changedAt" db:"changedAt"`
	OrderForeignKey string              `json:"orderForeignKey" db:"orderForeignKey"`
	OrderType       string              `json:"orderType" db:"orderType"`
	Operations      OperationList       `json:"operations" db:"operations"`
}

type payment struct {
	querier
	consumer
}

func New(w tablerow.Consumer, r *sql.DB) *payment {
	table := &payment{querier: querier{db: r, r: goqu.New("mysql", r)}, consumer: consumer{w: w}}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed table.sql
var tableSchema string

func (t *payment) CreateTableSql() string {
	return tableSchema
}
