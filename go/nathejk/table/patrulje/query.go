package patrulje

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
	GetAll(context.Context, Filter) ([]Patrulje, error)
	GetByID(context.Context, types.TeamID) (*Patrulje, error)
	GetStartedTeamIDs(context.Context, Filter) ([]types.TeamID, error)
	GetStartedTeams(context.Context, Filter) ([]StartedTeam, error)
	GetDiscontinuedTeamIDs(context.Context, Filter) ([]types.TeamID, error)
	AssignedNumbers(context.Context, types.YearSlug) (map[types.TeamID]string, error)
}

// StartedTeam is a patrol on the route and its strength there.
//
// Deliberately two columns. The post list's four numbers per line need exactly this
// and are recomputed on every scan during the race — peak 17 a minute — while GetAll
// carries three correlated subqueries per row (member count, t-shirt count, a payment
// sum with a nested IN) and measures 226ms against this season's data, against 0.3ms
// for the query below. Reaching for the fat row to read two fields is how a page that
// answered in milliseconds comes to take a quarter of a second per scan.
type StartedTeam struct {
	TeamID types.TeamID
	// ActiveMemberCount is zero for a patrol nobody is left racing on: the canonical
	// test for udgået. Maintained by the spejderstatus projection.
	ActiveMemberCount int
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

func (q *querier) GetAll(ctx context.Context, filters Filter) ([]Patrulje, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT p.teamId, teamNumber, name, groupName, korps, liga, contactName, contactPhone, contactEmail, contactRole, signupStatus, activeMemberCount,
			(SELECT COUNT(*) FROM spejder s where p.teamId = s.teamId) memberCount,
			(SELECT COUNT(*) FROM spejder s where p.teamId = s.teamId AND s.tshirtSize != '') tshirtCount,
			(SELECT COALESCE(SUM(pay.amount), 0) FROM payment pay
				WHERE pay.status IN ('reserved', 'received')
				  AND (pay.orderForeignKey = p.teamId
				       OR pay.orderForeignKey IN (SELECT o.orderId FROM orders o WHERE o.ownerType = 'patrulje' AND o.ownerId = p.teamId))) as paidAmount
		FROM patrulje p
		WHERE (LOWER(p.year) = LOWER(?) OR ? = '')`
	args := []any{filters.YearSlug, filters.YearSlug}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//totalRecords := 0
	patruljer := []Patrulje{}
	for rows.Next() {
		var p Patrulje
		if err := rows.Scan(&p.TeamID, &p.TeamNumber, &p.Name, &p.Group, &p.Korps, &p.Liga, &p.ContactName, &p.ContactPhone, &p.ContactEmail, &p.ContactRole, &p.SignupStatus, &p.ActiveMemberCount, &p.MemberCount, &p.TshirtCount, &p.PaidAmount); err != nil {
			return nil, err
		}
		payableAmount := p.TshirtCount*175 + p.MemberCount*250
		if p.SignupStatus != "" {
		} else if p.PaidAmount == 0 {
			p.SignupStatus = types.SignupStatusPay
		} else if p.PaidAmount >= payableAmount {
			p.SignupStatus = types.SignupStatusPaid
		} else {
			p.SignupStatus = types.SignupStatusSemipaid
		}
		patruljer = append(patruljer, p)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}
	//metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return patruljer, nil
}

func (q *querier) GetByID(ctx context.Context, teamID types.TeamID) (*Patrulje, error) {
	if len(teamID) == 0 {
		return nil, tables.ErrRecordNotFound
	}

	query := `SELECT p.teamId, p.teamNumber, p.name, p.groupName, p.korps, p.liga, p.memberCount, p.activeMemberCount, p.signupStatus
		FROM patrulje p
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		WHERE p.teamId = ?`
	var p Patrulje
	err := q.db.QueryRow(query, teamID).Scan(
		&p.TeamID,
		&p.TeamNumber,
		&p.Name,
		&p.Group,
		&p.Korps,
		&p.Liga,
		&p.MemberCount,
		&p.ActiveMemberCount,
		&p.SignupStatus,
	)
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

func (q *querier) GetStartedTeamIDs(ctx context.Context, f Filter) ([]types.TeamID, error) {
	where := goqu.Ex{
		"signupStatus": string(types.SignupStatusStarted),
	}
	if f.YearSlug != "" {
		where["year"] = f.YearSlug
	}

	var teamIDs []types.TeamID
	err := q.r.From("patrulje").Select("teamId").Where(where).ScanVals(&teamIDs)
	if err != nil {
		return nil, err
	}
	return teamIDs, nil
}

func (q *querier) GetDiscontinuedTeamIDs(ctx context.Context, f Filter) ([]types.TeamID, error) {
	return []types.TeamID{}, nil
}

// GetStartedTeams lists the patrols on the route with their remaining strength.
//
// See StartedTeam for why this exists next to GetAll.
func (q *querier) GetStartedTeams(ctx context.Context, f Filter) ([]StartedTeam, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT teamId, activeMemberCount FROM patrulje WHERE signupStatus = ?`
	args := []any{string(types.SignupStatusStarted)}
	if f.YearSlug != "" {
		query += ` AND year = ?`
		args = append(args, string(f.YearSlug))
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := []StartedTeam{}
	for rows.Next() {
		var t StartedTeam
		if err := rows.Scan(&t.TeamID, &t.ActiveMemberCount); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

/*
func (q *querier) GetContact(teamID types.TeamID) (*Contact, error) {
	if len(teamID) == 0 {
		return nil, tables.ErrRecordNotFound
	}

	query := `SELECT p.contactName, p.contactPhone, p.contactEmail, p.contactRole
		FROM patrulje p
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		WHERE p.teamId = ?`
	c := Contact{TeamID: teamID}
	err := q.db.QueryRow(query, teamID).Scan(
		&c.Name,
		&c.Phone,
		&c.Email,
		&c.Role,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &c, nil
}*/

// AssignedNumbers returns the team numbers already held by patruljer in the given
// year, keyed by team.
//
// Deliberately narrow, and deliberately not GetAll. The number-assignment saga
// needs exactly these two columns to know who is already numbered and how high the
// numbering has reached, and it needs them during replay, while the projections are
// still being rebuilt and the database is at its busiest.
//
// GetAll cannot serve that: it carries a hardcoded 3-second timeout that overrides
// whatever budget the caller set, and it computes three correlated subqueries per
// row — two counts over spejder and a payment sum containing a nested IN over
// orders. Under replay load in production that exceeded three seconds, the saga
// treated it as "cannot read existing numbers", stayed dormant to avoid re-issuing
// a number, and nothing was ever numbered. This query touches one table and reads
// two varchar columns.
//
// Rows with an empty teamNumber are skipped rather than returned as blanks: every
// caller only cares about numbers that exist.
func (q *querier) AssignedNumbers(ctx context.Context, year types.YearSlug) (map[types.TeamID]string, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT teamId, teamNumber FROM patrulje WHERE year = ? AND teamNumber <> ''`,
		string(year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	numbers := map[types.TeamID]string{}
	for rows.Next() {
		var (
			teamID types.TeamID
			number string
		)
		if err := rows.Scan(&teamID, &number); err != nil {
			return nil, err
		}
		numbers[teamID] = number
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return numbers, nil
}
