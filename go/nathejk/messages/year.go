package messages

import (
	"time"

	"nathejk.dk/nathejk/types"
)

type NathejkYearCreated struct {
	Slug            types.YearSlug `json:"slug"`
	Name            string         `json:"name"`
	Theme           string         `json:"theme,omitempty"`
	Story           string         `json:"story,omitempty"`
	CityDeparture   string         `json:"cityDeparture,omitempty"`
	CityDestination string         `json:"cityDestination,omitempty"`
	SignupStartTime *time.Time     `json:"signupStartTimei,omitempty"`
	StartTime       *time.Time     `json:"startTime,omitempty"`
	EndTime         *time.Time     `json:"endTime,omitempty"`
	MapOutlineFile  string         `json:"mapOutlineFile,omitempty"`
	DiplomaFile     string         `json:"diplomaTemplateFile,omitempty"`
}
