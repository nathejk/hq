package year

import (
	"context"
	"errors"
	"fmt"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/requestctx"
	tables "nathejk.dk/nathejk/table"
	"nathejk.dk/superfluids/streaminterface"
)

type Commands interface {
	Create(context.Context, types.YearSlug) error
	Update(context.Context, types.YearSlug, UpdateCommand) error
	Delete(context.Context, types.YearSlug) error
}

type commander struct {
	p streaminterface.Publisher
	q Queries
}

func (c commander) Create(ctx context.Context, slug types.YearSlug) error {
	u, ok := requestctx.UserFrom(ctx)
	if !ok {
		return errors.New("user not found in context")
	}
	if !slug.Valid() {
		return errors.New("not a valid slug format")
	}
	_, err := c.q.GetByID(ctx, slug)
	if err == nil {
		return errors.New("year already exists")
	}
	if err != tables.ErrRecordNotFound {
		return err
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.created", slug)))
	msg.SetBody(&messages.NathejkYearDeleted{Slug: slug})
	msg.SetMeta(&messages.Metadata{UserID: u.ID})
	return c.p.Publish(msg)
}

type UpdateCommand struct {
	Headline        *string
	Description     *string
	CityDeparture   *string
	CityDestination *string
	DateStart       *types.Date
	DateEnd         *types.Date
}

func (c commander) Update(ctx context.Context, slug types.YearSlug, cmd UpdateCommand) error {
	u, ok := requestctx.UserFrom(ctx)
	if !ok {
		return errors.New("user not found in context")
	}

	y, err := c.q.GetByID(ctx, slug)
	if err != nil {
		return err
	}
	dirty := (y == nil) ||
		((cmd.Headline != nil) && (*cmd.Headline != y.Headline)) ||
		((cmd.Description != nil) && (*cmd.Description != y.Description)) ||
		((cmd.CityDeparture != nil) && (*cmd.CityDeparture != y.CityDeparture)) ||
		((cmd.CityDestination != nil) && (*cmd.CityDestination != y.CityDestination)) ||
		((cmd.DateStart != nil) && (*cmd.DateStart != y.DateStart)) ||
		((cmd.DateEnd != nil) && (*cmd.DateEnd != y.DateEnd))

	if !dirty {
		return nil
	}
	body := messages.NathejkYearUpdated{
		Slug:            slug,
		Headline:        cmd.Headline,
		Description:     cmd.Description,
		CityDeparture:   cmd.CityDeparture,
		CityDestination: cmd.CityDestination,
		DateStart:       cmd.DateStart,
		DateEnd:         cmd.DateEnd,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.updated", slug)))
	msg.SetBody(&body)
	msg.SetMeta(&messages.Metadata{UserID: u.ID})

	return c.p.Publish(msg)
}

func (c commander) Delete(ctx context.Context, slug types.YearSlug) error {
	u, ok := requestctx.UserFrom(ctx)
	if !ok {
		return errors.New("context values missing")
	}
	_, err := c.q.GetByID(ctx, slug)
	if err != nil {
		return err
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.deleted", slug)))
	msg.SetBody(&messages.NathejkYearDeleted{Slug: slug})
	msg.SetMeta(&messages.Metadata{UserID: u.ID})
	return c.p.Publish(msg)
}

/*
type Date string

func (d Date) ToTime() time.Time {
	loc, err := time.LoadLocation("Europe/Copenhagen")
	if err != nil {
		log.Printf("Recoverable error %q", err)
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", string(d), loc)
	if err != nil {
		return time.Time{}
	}
	return t
}

type CreateRequest struct {
	Name     string `json:"name"`
	Controls []struct {
		Name                 string               `json:"name"`
		Scheme               string               `json:"scheme"`
		RelativeCheckgroupID types.ControlGroupID `json:"relativeControlGroupId"`
		DateRange            struct {
			StartDate time.Time `json:"startDate"`
			EndDate   time.Time `json:"endDate"`
		} `json:"dateRange"`
		Minus    int `json:"minus"`
		Plus     int `json:"plus"`
		Scanners []struct {
			DateRange struct {
				StartDate time.Time `json:"startDate"`
				EndDate   time.Time `json:"endDate"`
			} `json:"dateRange"`
			UserID types.UserID `json:"userId"`
		} `json:"scanners"`
	} `json:"controls"`
}
type ReadRequest struct {
}
type UpdateRequest struct {
	ID types.ControlGroupID `json:"controlGroupId"`

	CreateRequest
}

type DeleteRequest struct {
	ID types.ControlGroupID `json:"controlGroupId"`
}
*/
