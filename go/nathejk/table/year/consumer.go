package year

import (
	"fmt"

	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
)

type consumer struct {
	w tablerow.Consumer
}

func (c *consumer) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK.*.created"),
		streaminterface.SubjectFromStr("NATHEJK.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK.*.deleted"),
	}
}

func (c *consumer) HandleMessage(msg streaminterface.Message) error {
	var dialect = goqu.Dialect("mysql")

	switch true {
	case msg.Subject().Match("NATHEJK.*.created"):
		row := goqu.Record{"slug": msg.Subject().Parts()[1]}
		sqlStr, _, _ := dialect.Insert("years").OnConflict(goqu.DoNothing()).Rows(row).ToSQL()
		return c.w.Consume(sqlStr)

	case msg.Subject().Match("NATHEJK.*.updated"):
		var body messages.NathejkYearUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}

		update := goqu.Record{}
		if body.Headline != nil {
			update["headline"] = *body.Headline
		}
		if body.Description != nil {
			update["description"] = *body.Description
		}
		if body.CityDeparture != nil {
			update["cityDeparture"] = *body.CityDeparture
		}
		if body.CityDestination != nil {
			update["cityDestination"] = *body.CityDestination
		}
		if body.DateStart != nil {
			update["dateStart"] = *body.DateStart
		}
		if body.DateEnd != nil {
			update["dateEnd"] = *body.DateEnd
		}
		condition := goqu.C("slug").Eq(msg.Subject().Parts()[1])
		ds := dialect.Update("years").Set(update).Where(condition)
		sqlStr, _, _ := ds.ToSQL()

		return c.w.Consume(sqlStr)

	case msg.Subject().Match("NATHEJK.*.deleted"):
		return c.w.Consume(fmt.Sprintf("DELETE FROM years WHERE slug=%q", msg.Subject().Parts()[1]))

	}
	return nil
}
