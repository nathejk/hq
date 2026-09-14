package memberverification

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/nathejk/shared-go/types"
)

type Queries interface {
	// ByMembers reads the verifications for the given members, keyed by member id.
	//
	// Keyed by member rather than returned as a slice because every caller is decorating a
	// roster it already has, and a member with nothing verified is simply absent from the
	// map — which is the same answer as an empty Verification, so callers need not check.
	ByMembers(ctx context.Context, year types.YearSlug, memberIDs []types.MemberID) (map[types.MemberID]Verification, error)
}

// Verification is what one member has proven, and when.
//
// The numbers are normalized (digits only), as the events carry them; an empty one means
// "not verified", not "no number".
type Verification struct {
	Phone           types.PhoneNumber `json:"phone,omitempty"`
	PhoneUts        int64             `json:"phoneUts,omitempty"`
	PhoneContact    types.PhoneNumber `json:"phoneContact,omitempty"`
	PhoneContactUts int64             `json:"phoneContactUts,omitempty"`
}

type querier struct {
	db *sql.DB
}

func (q *querier) ByMembers(ctx context.Context, year types.YearSlug, memberIDs []types.MemberID) (map[types.MemberID]Verification, error) {
	out := map[types.MemberID]Verification{}
	if len(memberIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := make([]any, 0, len(memberIDs)+1)
	args = append(args, string(year))
	placeholders := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		placeholders = append(placeholders, "?")
		args = append(args, string(id))
	}
	query := `SELECT memberId, phone, phoneUts, phoneContact, phoneContactUts
		FROM memberverification
		WHERE year = ? AND memberId IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id types.MemberID
		var v Verification
		if err := rows.Scan(&id, &v.Phone, &v.PhoneUts, &v.PhoneContact, &v.PhoneContactUts); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}
