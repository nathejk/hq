package payment

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type Queries interface {
	GetAll(context.Context, Filter) ([]Payment, error)
	GetByReference(context.Context, string) (*Payment, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

func (q *querier) GetAll(ctx context.Context, f Filter) ([]Payment, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	where := goqu.Ex{}
	if f.Year != "" {
		where["year"] = f.Year
	}
	if len(f.TeamIDs) > 0 {
		where["orderForeignKey"] = f.TeamIDs
	}

	var payments []Payment
	err := q.r.From("payment").Where(where).Order(goqu.I("createdAt").Asc()).ScanStructs(&payments)
	if err != nil {
		return nil, err
	}
	return payments, nil
}

func (q *querier) GetAll_old(ctx context.Context, teamID types.TeamID) ([]*Payment, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT reference, receiptEmail, returnUrl, year, currency, FLOOR(amount/100), method, status, createdAt, changedAt, orderForeignKey, orderType
		FROM payment
		WHERE orderForeignKey = ?`
	args := []any{teamID} //filters.Year, filters.Year}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//	totalRecords := 0
	var payments []*Payment
	for rows.Next() {
		var p Payment
		err := rows.Scan(&p.Reference, &p.ReceiptEmail, &p.ReturnUrl, &p.Year, &p.Currency, &p.Amount, &p.Method, &p.Status, &p.CreatedAt, &p.ChangedAt, &p.OrderForeignKey, &p.OrderType)
		if err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}
	//metadata := Metadata{} //calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return payments, nil
}

func (q *querier) GetByReference(ctx context.Context, ref string) (*Payment, error) {
	if len(ref) == 0 {
		return nil, tables.ErrRecordNotFound
	}

	query := `SELECT receiptEmail, returnUrl, year, currency, FLOOR(amount/100), method, status, createdAt, changedAt, orderForeignKey, orderType
		FROM payment
		WHERE reference = ?`
	var p Payment
	err := q.db.QueryRow(query, ref).Scan(
		&p.ReceiptEmail,
		&p.ReturnUrl,
		&p.Year,
		&p.Currency,
		&p.Amount,
		&p.Method,
		&p.Status,
		&p.CreatedAt,
		&p.ChangedAt,
		&p.OrderForeignKey,
		&p.OrderType,
	)
	p.Reference = ref
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &p, nil
}

func (q *querier) AmountDueByTeamID(teamID types.TeamID) {
}

func (q *querier) AmountPaidByTeamID(teamID types.TeamID) int {
	query := `SELECT FLOOR(SUM(amount)/100) FROM payment WHERE orderForeignKey = ? AND status IN (?, ?)`
	var paidAmount int
	if err := q.db.QueryRow(query, teamID, types.PaymentStatusReserved, types.PaymentStatusReceived).Scan(&paidAmount); err != nil {
		return 0
	}
	return paidAmount
}
