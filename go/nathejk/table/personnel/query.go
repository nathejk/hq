package personnel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type querier struct {
	db *sql.DB
}

func (q *querier) GetAll(ctx context.Context, f Filter) ([]*Person, error) {
	where := []string{}
	args := []any{}
	if f.YearSlug != "" {
		where = append(where, "year = ?")
		args = append(args, f.YearSlug)
	}
	if (f.UserIDs != nil) && (len(f.UserIDs) == 0) {
		return []*Person{}, nil
	}
	if len(f.UserIDs) == 1 {
		where = append(where, "userId = ?")
		args = append(args, f.UserIDs[0])
	}
	if len(f.UserIDs) > 1 {
		where = append(where, fmt.Sprintf("userId IN (?%s)", strings.Repeat(",?", len(f.UserIDs)-1)))
		for _, id := range f.UserIDs {
			args = append(args, id)
		}
	}
	if len(f.UserTypes) > 0 {
		where = append(where, fmt.Sprintf("userType IN (?%s)", strings.Repeat(",?", len(f.UserTypes)-1)))
		for _, id := range f.UserTypes {
			args = append(args, id)
		}
	}
	if f.Department != "" {
		where = append(where, "JSON_EXTRACT(additionals, '$.department') = ?")
		args = append(args, f.Department)
	}
	if len(where) == 0 {
		where = []string{"true"}
	}
	// The paid amount is joined as one pre-grouped aggregate rather than computed as a
	// correlated subquery per person (subtask 14 of task 135).
	//
	// The correlated form was correct — task 014 added the OR so gøglere who paid via an
	// order stopped being hidden — but its predicate is unindexable: with
	// `orderForeignKey = userId OR orderForeignKey IN (SELECT …)` the optimizer lists
	// idx_payment_order in possible_keys and declines it, giving type=ALL over every
	// payment row for every person. Indexing payment (also task 135) was necessary and
	// not sufficient; only removing the OR lets the key be used.
	//
	// So the two ways a payment reaches a person — straight to the userId (legacy) or via
	// an order owned by them — are unioned into one keyed set and summed once. UNION, not
	// UNION ALL: should an orderId ever equal a userId, both branches yield the same pair
	// and the payment must still count once, exactly as the OR counted it once. orders is
	// not filtered by ownerType, because the predicate replaced here was ownerId alone.
	//
	// The derived table's key column is `ownerId`, not `userId`: the filters above are
	// unqualified column names, and a joined `userId` would make `userId = ?` ambiguous.
	query := `SELECT userId, userType, armNumber, name, phone, email, groupName, korps, klan, signupStatus, tshirtSize, additionals,
		COALESCE(pay.paidAmount, 0) as paidAmount
		FROM personnel
		LEFT JOIN (
			SELECT k.ownerId, SUM(pa.amount) paidAmount
			FROM (
				SELECT userId ownerId, userId foreignKey FROM personnel
				UNION
				SELECT o.ownerId, o.orderId FROM orders o
			) k
			JOIN payment pa ON pa.orderForeignKey = k.foreignKey AND pa.status IN ('reserved', 'received')
			GROUP BY k.ownerId
		) pay ON pay.ownerId = personnel.userId
		WHERE ` + strings.Join(where, " AND ")
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return []*Person{}, nil
		default:
			return nil, err
		}
	}
	defer rows.Close()

	//totalRecords := 0
	personnel := []*Person{}
	for rows.Next() {
		var p Person
		var additionals []byte
		if err := rows.Scan(&p.ID, &p.UserType, &p.ArmNumber, &p.Name, &p.Phone, &p.Email, &p.Group, &p.Korps, &p.Klan, &p.Status, &p.TshirtSize, &additionals, &p.PaidAmount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(additionals, &p.Additionals); err != nil {
			p.Additionals = map[string]any{}
		}

		personnel = append(personnel, &p)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return personnel, nil
}

func (q *querier) GetByID(ctx context.Context, staffID types.UserID) (*Person, error) {
	if len(staffID) == 0 {
		return nil, tables.ErrRecordNotFound
	}

	query := `SELECT t.userId, t.armNumber, t.name, t.phone, t.email, t.groupName, t.korps, t.klan, t.signupStatus, t.tshirtSize, t.additionals
		FROM personnel t
		WHERE t.userId = ?`
	var t Person
	var additionals []byte
	err := q.db.QueryRow(query, staffID).Scan(
		&t.ID,
		&t.ArmNumber,
		&t.Name,
		&t.Phone,
		&t.Email,
		&t.Group,
		&t.Korps,
		&t.Klan,
		&t.Status,
		&t.TshirtSize,
		&additionals,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	t.Additionals = map[string]any{}
	if len(additionals) > 0 {
		if err := json.Unmarshal(additionals, &t.Additionals); err != nil {
			return nil, err
		}
	}

	return &t, nil
}
