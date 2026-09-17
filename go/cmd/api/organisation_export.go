package main

import (
	"net/http"

	"github.com/xuri/excelize/v2"
)

// The organisation export: two sheets mirroring the two panes of the org screen —
// crew members and vehicles — so the roster and the vehicle inventory can be printed,
// counted or handed off outside the app. It reuses the picking list's row-scanning and
// sheet-writing helpers (pluklisteRows / writePluklisteSheet) because the shape is the
// same: a header row and generic string/number cells.
//
// Both lists are scoped to the request's year and exclude soft-deleted rows, and both
// resolve sectionSlug to the section's human label so a sheet reads the way the tree
// does rather than in slugs. A member or vehicle with no section shows a blank cell
// (Ikke tildelt is the tree's word for it, but a blank sorts and filters more simply
// in a spreadsheet).
func (app *application) excelOrganisationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	year := app.YearSlug(r)

	members, err := app.pluklisteRows(ctx, `
		SELECT c.name, c.phone, c.email, c.medlemNr, c.groupName, c.corps, c.diet,
		       COALESCE(s.label, '')
		FROM crewmember c
		LEFT JOIN section s ON s.slug = c.sectionSlug AND s.year = c.year
		WHERE c.year = ? AND c.deleted = 0
		ORDER BY COALESCE(s.label, ''), c.name`, year, 8)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	vehicles, err := app.pluklisteRows(ctx, `
		SELECT v.licensePlate, v.brand, v.model, v.color, v.seatCount,
		       COALESCE(d.name, ''), COALESCE(s.label, ''), v.description
		FROM vehicle v
		LEFT JOIN crewmember d ON d.userId = v.driverUserId AND d.year = v.year
		LEFT JOIN section s ON s.slug = v.sectionSlug AND s.year = v.year
		WHERE v.year = ? AND v.deleted = 0
		ORDER BY COALESCE(s.label, ''), v.licensePlate`, year, 8)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
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
	view := &excelize.ViewOptions{ZoomScale: &zoomScale}

	writePluklisteSheet(xlsx, "Sheet1", styleTitle, view,
		[]string{"Navn", "Telefon", "Email", "Medlemsnr.", "Gruppe", "Korps", "Diæt", "Sektion"},
		[]int{30, 15, 30, 14, 20, 16, 20, 25}, members)
	xlsx.SetSheetName("Sheet1", "Crew")

	xlsx.NewSheet("Køretøjer")
	writePluklisteSheet(xlsx, "Køretøjer", styleTitle, view,
		[]string{"Nummerplade", "Mærke", "Model", "Farve", "Pladser", "Chauffør", "Sektion", "Beskrivelse"},
		[]int{15, 16, 16, 12, 8, 30, 25, 40}, vehicles)

	writeXlsx(app, w, r, xlsx, "organisation")
}
