package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"

	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/klan"
	"nathejk.dk/nathejk/table/patrulje"
)

func (app *application) excelPatruljeHandler(w http.ResponseWriter, r *http.Request) {
	filter := patrulje.Filter{}
	teams, err := app.models.Patrulje.GetAll(r.Context(), filter)
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
	headers := []string{"ID", "Nummer", "Patrulje", "Gruppe", "Korps", "Antal spejdere", "Adv. ligg ID", "Betalt", "Tilmelding email", "Tilmelding tlf.", "Kontakt navn", "kontakt tlf.", "Kontakt Email", "Kontakt rolle"}
	widths := []int{40, 10, 30, 30, 10, 10, 10, 10, 30, 10, 20, 10, 30, 15}
	for i, header := range headers {
		col := string(rune(65 + i))
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("%s%d", col, 1), header)
		xlsx.SetColWidth("Sheet1", col, col, float64(widths[i]))
	}

	xlsx.NewSheet("Sheet2")
	xlsx.SetSheetView("Sheet2", 0, sheetViewOptions)
	xlsx.SetRowStyle("Sheet2", 1, 1, styleTitle)
	headers = []string{"TeamID", "Nummer", "Patrulje", "Navn", "Adresse", "Postnr.", "By", "Email", "Telefon", "Pårørende tlf.", "Fødselsdag", "T-shirt"}
	widths = []int{40, 10, 30, 30, 30, 10, 10, 30, 10, 10, 10, 10}
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
		signup, _ := app.models.Signup.GetByID(team.TeamID)
		row := []any{
			team.TeamID,
			team.TeamNumber,
			team.Name,
			team.Group,
			team.Korps,
			team.MemberCount,
			team.Liga,
			team.PaidAmount / 100,
			signup.EmailPending,
			signup.PhonePending,
			team.ContactName,
			team.ContactPhone,
			team.ContactEmail,
			team.ContactRole,
		}
		for j, col := range row {
			xlsx.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(rune(65+j)), row1), col)
		}
		row1++

		members, _, _ := app.models.Members.GetSpejdere(data.Filters{TeamID: team.TeamID})
		for _, member := range members {
			row := []any{
				team.TeamID,
				team.TeamNumber,
				team.Name,
				member.Name,
				member.Address,
				member.PostalCode,
				member.City,
				member.Email,
				member.Phone,
				member.PhoneParent,
				member.Birthday,
				member.TShirtSize,
			}
			for j, value := range row {
				xlsx.SetCellValue("Sheet2", fmt.Sprintf("%s%d", string(rune(65+j)), row2), value)
			}
			row2++
		}
	}

	xlsx.SetSheetName("Sheet1", "Patruljer")
	xlsx.SetSheetName("Sheet2", "Spejdere")

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+"patruljer-"+time.Now().Format("20060102150405")+".xlsx")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Expires", "0")
	xlsx.Write(w)
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
