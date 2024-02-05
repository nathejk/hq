package commands

import (
	"time"

	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type Year struct {
	Slug            types.YearSlug `json:"slug"`
	Name            string         `json:"name"`
	Theme           string         `json:"theme"`
	Story           string         `json:"story"`
	CityDeparture   string         `json:"cityDeparture"`
	CityDestination string         `json:"cityDestination"`
	SignupStartTime *time.Time     `json:"signupStartTime"`
	StartTime       *time.Time     `json:"startTime"`
	EndTime         *time.Time     `json:"endTime"`
	MapOutlineFile  string         `json:"mapOutlineFile"`
	DiplomaFile     string         `json:"diplomaTemplateFile"`
}

type querier interface {
	Get(types.YearSlug) (*data.Year, error)
}
type yearCmd struct {
	p streaminterface.Publisher
	q querier
}

func NewYear(p streaminterface.Publisher, q querier) *yearCmd {
	return &yearCmd{
		p: p,
		q: q,
	}
}

func (c *yearCmd) Create(year *Year) error {
	body := messages.NathejkYearCreated{
		Slug:            year.Slug,
		Name:            year.Name,
		Theme:           year.Theme,
		Story:           year.Story,
		CityDeparture:   year.CityDeparture,
		CityDestination: year.CityDestination,
		SignupStartTime: year.SignupStartTime,
		StartTime:       year.StartTime,
		EndTime:         year.EndTime,
		MapOutlineFile:  year.MapOutlineFile,
		DiplomaFile:     year.DiplomaFile,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:year.created"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "hq-api"}
	msg.SetMeta(&meta)

	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}
func (c *yearCmd) Update(year *Year) error {
	return nil
}
func (c *yearCmd) Delete(year *Year) error {
	return nil
}
