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
	Identities(context.Context, types.YearSlug) ([]Identity, error)
}

// Identity is the three fields needed to label a patrol on a map: who it is, not how it is doing.
//
// Its own type, and its own query, for the same reason StartedTeam has one. The position endpoint
// is polled by a map layer while the race runs, and GetAll aggregates over spejder, orders and
// payment to build the fat row — hundreds of milliseconds against a season's data. Reaching for
// that row to read a name and a number is how a polled endpoint becomes the slowest thing in HQ.
type Identity struct {
	TeamID     types.TeamID
	TeamNumber string
	Name       string
}

// StartedTeam is a patrol on the route: who it is, and its strength there.
//
// Deliberately one table and no aggregates. Every caller is a screen about the race, and they
// are recomputed on every scan — peak 17 a minute — while GetAll joins three derived tables to
// compute member counts, t-shirt counts and a payment sum unioned across orders, and measures
// hundreds of milliseconds against this season's data against 0.3ms for the query below.
// Reaching for the fat row to read a name is how a page that answered in milliseconds comes to
// take a quarter of a second per scan.
//
// The labels (number, name, group) cost nothing to carry because they are columns of the row
// already being read; it is the joins that are expensive, not the width.
type StartedTeam struct {
	TeamID     types.TeamID
	TeamNumber string
	Name       string
	Group      string
	// ActiveMemberCount is zero for a patrol nobody is left racing on: the canonical
	// test for udgået. Maintained by the spejderstatus projection.
	ActiveMemberCount int
	// StartedUts is when the patrol went on the route, or 0 for a row projected before
	// the column existed. Read here because anything plotting the race against time
	// needs to know when each team entered it — see bingoSeries in cmd/api/bingo.go.
	StartedUts int64
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

func (q *querier) GetAll(ctx context.Context, filters Filter) ([]Patrulje, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Aggregates are joined as pre-grouped derived tables rather than computed as
	// correlated subqueries per row. The correlated form was three subqueries per
	// patrol against tables that carry no usable index for them — `spejder` is keyed
	// (year, memberId), so a lookup by teamId is a full scan, and `payment` has no
	// index on orderForeignKey at all — which made this O(patruljer × rows) and, in
	// production, exceeded the 3-second budget below: the patrol list answered 500.
	// Both schemas live in shared-go, so the fix that this repo owns is to scan each
	// table once and join.
	//
	// The payment side unions the two ways a payment reaches a team — straight to the
	// teamId (older years) or via an order owned by it — into one keyed set before
	// summing, which is the same set the OR + nested IN described.
	query := `SELECT p.teamId, p.teamNumber, p.name, p.groupName, p.korps, p.liga, p.contactName, p.contactPhone, p.contactEmail, p.contactRole, p.signupStatus, p.activeMemberCount,
			COALESCE(m.memberCount, 0) memberCount,
			COALESCE(m.tshirtCount, 0) tshirtCount,
			COALESCE(pay.paidAmount, 0) paidAmount
		FROM patrulje p
		LEFT JOIN (
			SELECT s.teamId, COUNT(*) memberCount, SUM(s.tshirtSize != '') tshirtCount
			FROM spejder s GROUP BY s.teamId
		) m ON m.teamId = p.teamId
		LEFT JOIN (
			SELECT k.teamId, SUM(pa.amount) paidAmount
			FROM (
				SELECT o.ownerId teamId, o.orderId foreignKey FROM orders o WHERE o.ownerType = 'patrulje'
				UNION ALL
				SELECT p2.teamId, p2.teamId FROM patrulje p2
			) k
			JOIN payment pa ON pa.orderForeignKey = k.foreignKey AND pa.status IN ('reserved', 'received')
			GROUP BY k.teamId
		) pay ON pay.teamId = p.teamId
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

// Identities lists a year's patruljer by id, number and name.
//
// Every patrol, not only the started ones: a position is evidence in its own right, and a team that
// reported from the bus before anybody pressed start is exactly the team an operator is looking for.
// Filtering to `started` here would hide it.
func (q *querier) Identities(ctx context.Context, year types.YearSlug) ([]Identity, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT teamId, teamNumber, name FROM patrulje WHERE LOWER(year) = LOWER(?)`,
		string(year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Identity{}
	for rows.Next() {
		var i Identity
		if err := rows.Scan(&i.TeamID, &i.TeamNumber, &i.Name); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *querier) GetByID(ctx context.Context, teamID types.TeamID) (*Patrulje, error) {
	if len(teamID) == 0 {
		return nil, tables.ErrRecordNotFound
	}

	// year, remark and remarkSeverity are selected because the remark command reads this
	// row: it needs the note to dirty-check against, and the year to publish onto the
	// patrol's own year rather than whatever the caller happens to think is current.
	query := `SELECT p.teamId, p.year, p.teamNumber, p.name, p.groupName, p.korps, p.liga, p.memberCount, p.activeMemberCount, p.signupStatus, p.remark, p.remarkSeverity
		FROM patrulje p
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		WHERE p.teamId = ?`
	var p Patrulje
	err := q.db.QueryRow(query, teamID).Scan(
		&p.TeamID,
		&p.Year,
		&p.TeamNumber,
		&p.Name,
		&p.Group,
		&p.Korps,
		&p.Liga,
		&p.MemberCount,
		&p.ActiveMemberCount,
		&p.SignupStatus,
		&p.Remark,
		&p.RemarkSeverity,
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

	query := `SELECT teamId, teamNumber, name, groupName, activeMemberCount, startedUts FROM patrulje WHERE signupStatus = ?`
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
		if err := rows.Scan(&t.TeamID, &t.TeamNumber, &t.Name, &t.Group, &t.ActiveMemberCount, &t.StartedUts); err != nil {
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
// whatever budget the caller set, and it aggregates over spejder, orders and payment
// to build the fat row. Under replay load in production that exceeded three seconds,
// the saga treated it as "cannot read existing numbers", stayed dormant to avoid
// re-issuing a number, and nothing was ever numbered. This query touches one table
// and reads two varchar columns.
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
