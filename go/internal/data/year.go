package data

import (
	"context"
	"database/sql"
	"time"

	"nathejk.dk/nathejk/types"
)

type Year struct {
	Slug            string     `json:"slug"`
	Name            string     `json:"name"`
	Theme           string     `json:"theme"`
	Story           string     `json:"story"`
	CityDeparture   string     `json:"cityDeparture"`
	CityDestination string     `json:"cityDestination"`
	SignupStartTime *time.Time `json:"signupStartTime"`
	StartTime       *time.Time `json:"startTime"`
	EndTime         *time.Time `json:"endTime"`
	MapOutlineFile  string     `json:"mapOutlineFile"`
	DiplomaFile     string     `json:"diplomaTemplateFile"`
	Patruljer       struct {
		SignupCount   int
		StartedCount  int
		FinishedCount int
	}
	Bandits struct {
		SignupCount int
		ScanCount   int
	}
}

type YearModel struct {
	DB *sql.DB
}

func (m YearModel) GetAll(filters Filters) ([]*Year, Metadata, error) {
	_, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	years := []*Year{
		&Year{Slug: "2023", Name: "Nathejk 2024", Theme: "Ex Nihilo", CityDeparture: "Kalundborg", CityDestination: "Stenlille"},
		&Year{Slug: "2022", Name: "Nathejk 2022", Theme: "Ufomania", CityDeparture: "Faxe", CityDestination: "Ringsted"},
		&Year{Slug: "2021", Name: "Nathejk 2021", Theme: "Kong Etruds Sværd", CityDeparture: "Helsingør", CityDestination: "Hillerød"},
	}
	return years, Metadata{}, nil
}

func (m YearModel) Get(types.YearSlug) (*Year, error) {
	_, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	year := &Year{Slug: "2023", Name: "Nathejk 2024", Theme: "Ex Nihilo", CityDeparture: "Kalundborg", CityDestination: "Stenlille"}
	return year, nil
}
