package main

import (
	"net/http"
	"strings"

	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
)

// Vehicles are part of the organisation screen: a car is brought by a crew
// member and, like that crew member, belongs to at most one section. The
// handlers therefore mirror the crew-member ones — register, edit, assign,
// withdraw — and the list itself is served by showOrganisationHandler.

// registerVehicleHandler enrols a new vehicle for the current year.
func (app *application) registerVehicleHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		LicensePlate    string `json:"licensePlate"`
		CustodianUserID string `json:"custodianUserId"`
		Color           string `json:"color"`
		Brand           string `json:"brand"`
		Model           string `json:"model"`
		SeatCount       uint   `json:"seatCount"`
		Description     string `json:"description"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	fields := vehicle.RegisterFields{
		LicensePlate:    strings.TrimSpace(input.LicensePlate),
		CustodianUserID: types.UserID(strings.TrimSpace(input.CustodianUserID)),
		Color:           strings.TrimSpace(input.Color),
		Brand:           strings.TrimSpace(input.Brand),
		Model:           strings.TrimSpace(input.Model),
		SeatCount:       input.SeatCount,
		Description:     strings.TrimSpace(input.Description),
	}
	// The custodian must be a crew member of this year, or the car would list a
	// keeper the organisation screen cannot show — and nobody could be reached
	// about it.
	if fields.CustodianUserID != "" {
		if _, err := app.models.CrewMember.GetByID(r.Context(), fields.CustodianUserID); err != nil {
			app.BadRequestResponse(w, r, errFromString("custodian is not a crew member in the current year"))
			return
		}
	}
	vehicleID, err := app.commands.Vehicle.Register(r.Context(), app.YearSlug(r), fields)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"vehicleId": vehicleID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateVehicleHandler edits a vehicle's details.
//
// The input mirrors NathejkVehicleUpdated's delta semantics: a field absent from
// the request body is left alone, a field present but empty is cleared. That is
// why every field is a pointer — with plain values, "no description supplied"
// and "clear the description" would be the same request.
func (app *application) updateVehicleHandler(w http.ResponseWriter, r *http.Request) {
	id := types.VehicleID(app.ReadNamedParam(r, "vehicleId"))
	if id == "" {
		app.BadRequestResponse(w, r, errFromString("vehicleId is required"))
		return
	}
	var input struct {
		LicensePlate    *string `json:"licensePlate"`
		CustodianUserID *string `json:"custodianUserId"`
		Color           *string `json:"color"`
		Brand           *string `json:"brand"`
		Model           *string `json:"model"`
		SeatCount       *uint   `json:"seatCount"`
		Description     *string `json:"description"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	fields := vehicle.UpdateFields{
		Color:       trimmed(input.Color),
		Brand:       trimmed(input.Brand),
		Model:       trimmed(input.Model),
		SeatCount:   input.SeatCount,
		Description: trimmed(input.Description),
	}
	if plate := trimmed(input.LicensePlate); plate != nil {
		if *plate == "" {
			app.BadRequestResponse(w, r, errFromString("license plate is required"))
			return
		}
		fields.LicensePlate = plate
	}
	if input.CustodianUserID != nil {
		custodian := types.UserID(strings.TrimSpace(*input.CustodianUserID))
		if custodian == "" {
			app.BadRequestResponse(w, r, errFromString("custodian is required: somebody has to answer for the vehicle"))
			return
		}
		if _, err := app.models.CrewMember.GetByID(r.Context(), custodian); err != nil {
			app.BadRequestResponse(w, r, errFromString("custodian is not a crew member in the current year"))
			return
		}
		fields.CustodianUserID = &custodian
	}
	if err := app.commands.Vehicle.Update(r.Context(), app.YearSlug(r), id, fields); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"vehicleId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// assignVehicleSectionHandler assigns (or, with an empty slug, unassigns) a
// vehicle to a section.
func (app *application) assignVehicleSectionHandler(w http.ResponseWriter, r *http.Request) {
	id := types.VehicleID(app.ReadNamedParam(r, "vehicleId"))
	if id == "" {
		app.BadRequestResponse(w, r, errFromString("vehicleId is required"))
		return
	}
	var input struct {
		SectionSlug string `json:"sectionSlug"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	year := app.YearSlug(r)
	sectionSlug := types.Slug(input.SectionSlug)
	if sectionSlug != "" {
		if _, err := app.models.Section.GetBySlug(r.Context(), year, sectionSlug); err != nil {
			app.BadRequestResponse(w, r, errFromString("section not found in current year"))
			return
		}
	}
	if err := app.commands.Vehicle.AssignSection(r.Context(), year, id, sectionSlug); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"vehicleId":   id,
		"sectionSlug": sectionSlug,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteVehicleHandler withdraws a vehicle from the race (soft delete in the
// read model).
func (app *application) deleteVehicleHandler(w http.ResponseWriter, r *http.Request) {
	id := types.VehicleID(app.ReadNamedParam(r, "vehicleId"))
	if id == "" {
		app.BadRequestResponse(w, r, errFromString("vehicleId is required"))
		return
	}
	if err := app.commands.Vehicle.Delete(r.Context(), app.YearSlug(r), id); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// trimmed keeps the "absent means leave it" distinction while still trimming the
// values that are present.
func trimmed(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}
