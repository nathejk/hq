package main

import (
	"fmt"
	"net/http"
	"time"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/types"
)

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
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

	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	year := &commands.Year{
		Slug:            input.Slug,
		Name:            input.Name,
		Theme:           input.Theme,
		Story:           input.Story,
		CityDeparture:   input.CityDeparture,
		CityDestination: input.CityDestination,
	}
	/*
		v := validator.New()
		if year.Validate(v); !v.Valid() {
			app.FailedValidationResponse(w, r, v.Errors)
			return
		}
	*/
	if err := app.commands.Years.Create(year); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/year/%s", year.Slug))

	err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"year": year}, headers)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

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

func (app *application) listYearsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Year = app.ReadString(qs, "year", fmt.Sprintf("%d", time.Now().Year()))
	input.Filters.Page = app.ReadInt(qs, "page", 1, v)
	input.Filters.PageSize = app.ReadInt(qs, "page_size", 1000, v)
	input.Filters.Sort = app.ReadString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id"}

	if input.Filters.Validate(v); !v.Valid() {
		app.FailedValidationResponse(w, r, v.Errors)
		return
	}

	var err error
	//status, metadata, err := app.models.GetCheckgroupsStatus(input.Filters)
	//checkgroups, metadata, err := app.models.Checkgroups.GetAll(input.Filters) //, startedTeamIDs, discontinuedTeamIDs)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	years := []Year{
		Year{Slug: "2023", Name: "Nathejk 2024", Theme: "Ex Nihilo", CityDeparture: "Kalundborg", CityDestination: "Stenlille"},
		Year{Slug: "2022", Name: "Nathejk 2022", Theme: "Ufomania", CityDeparture: "Faxe"},
		Year{Slug: "2021", Name: "Nathejk 2021", Theme: "Kong Etruds Sværd"},
	}
	envelope := jsonapi.Envelope{
		//"metadata": metadata,
		"years": years,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
