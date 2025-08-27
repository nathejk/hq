package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/klan"
)

func (app *application) showKlanListHandler(w http.ResponseWriter, r *http.Request) {
	filter := klan.Filter{}
	teams, err := app.models.Klan.GetAll(context.Background(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teams": teams}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) showKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamId := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamId == "" {
		app.NotFoundResponse(w, r)
		return
	}
	team, err := app.models.Teams.GetKlan(teamId)
	if err != nil {
		log.Printf("GetKlan %q", err)
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	members, _, err := app.models.Members.GetSeniore(data.Filters{TeamID: teamId})
	if err != nil {
		log.Printf("GetSenior %q", err)
	}

	config := TeamConfig{
		MinMemberCount: 1,
		MaxMemberCount: 4,
		MemberPrice:    250,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
	}
	//contact, _ := app.models.Teams.GetContact(teamId)

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "members": members, "payments": []any{}}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) patchKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Lok *string `json:"lok"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	if _, err := app.models.Teams.GetKlan(teamID); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if input.Lok != nil {
		if err := app.commands.Team.AssignToLok(teamID, *input.Lok); err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team    commands.Klan     `json:"team"`
		Members []commands.Senior `json:"members"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	_, err := app.models.Teams.GetKlan(teamID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Team.UpdateKlan(teamID, input.Team, input.Members)
	if err != nil {
		log.Printf("UpdateKlan  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	/*
		page := fmt.Sprintf("/patrulje/%s", input.TeamID)
		err = app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"team": map[string]string{"teamPage": page}}, nil)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
		}*/
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) excelKlanHandler(w http.ResponseWriter, r *http.Request) {
	filter := klan.Filter{}
	teams, err := app.models.Klan.GetAll(context.Background(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}

	xlsx := excelize.NewFile()
	styleTitle, _ := xlsx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#000080"}, // navy blue
			Pattern: 1,
		},
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	zoomScale := 150.0
	sheetViewOptions := &excelize.ViewOptions{
		ZoomScale: &zoomScale,
	}

	xlsx.SetSheetView("Sheet1", 0, sheetViewOptions)
	xlsx.SetRowStyle("Sheet1", 1, 1, styleTitle)
	headers := []string{"ID", "Klannavn", "Gruppe", "Korps", "Antal banditter", "Lok", "Betalt", "Tilmelding email", "Tilmelding tlf."}
	widths := []int{40, 30, 30, 10, 10, 10, 10, 30, 15}
	for i, header := range headers {
		col := string(rune(65 + i))
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("%s%d", col, 1), header)
		xlsx.SetColWidth("Sheet1", col, col, float64(widths[i]))
	}

	xlsx.NewSheet("Sheet2")
	xlsx.SetSheetView("Sheet2", 0, sheetViewOptions)
	xlsx.SetRowStyle("Sheet2", 1, 1, styleTitle)
	headers = []string{"TeamID", "Klannavn", "Navn", "Adresse", "Postnr.", "By", "Email", "Telefon", "Fødselsdag", "Vegetar", "T-shirt"}
	widths = []int{40, 30, 30, 30, 10, 10, 30, 10, 10, 10, 10}
	for i, header := range headers {
		col := string(rune(65 + i))
		xlsx.SetCellValue("Sheet2", fmt.Sprintf("%s%d", col, 1), header)
		xlsx.SetColWidth("Sheet2", col, col, float64(widths[i]))
	}

	row1, row2 := 2, 2
	for _, team := range teams {
		if team.PaidAmount == 0 {
			continue
		}
		signup, _ := app.models.Signup.GetByID(team.ID)
		row := []any{
			team.ID,
			team.Name,
			team.Group,
			team.Korps,
			team.MemberCount,
			team.Lok,
			team.PaidAmount / 100,
			signup.EmailPending,
			signup.PhonePending,
		}
		for j, col := range row {
			xlsx.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(rune(65+j)), row1), col)
		}
		row1++

		members, _, _ := app.models.Members.GetSeniore(data.Filters{TeamID: team.ID})
		for _, member := range members {
			row := []any{
				team.ID,
				team.Name,
				member.Name,
				member.Address,
				member.PostalCode,
				member.City,
				member.Email,
				member.Phone,
				member.Birthday,
				member.Diet,
				member.TShirtSize,
			}
			for j, value := range row {
				xlsx.SetCellValue("Sheet2", fmt.Sprintf("%s%d", string(rune(65+j)), row2), value)
			}
			row2++
		}
	}

	xlsx.SetSheetName("Sheet1", "Klaner")
	xlsx.SetSheetName("Sheet2", "Banditter")

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+"banditter-"+time.Now().Format("20060102150405")+".xlsx")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Expires", "0")
	xlsx.Write(w)
}
