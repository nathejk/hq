package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nathejk/shared-go/types"
	"github.com/xuri/excelize/v2"
)

// The picking list (plukliste): who ordered a t-shirt, in which size, so the shirts
// can be counted and bagged per person. Four sheets — spejdere, seniorer, gøglere,
// crew — because the shirts are handed out through four different channels and each
// needs a different bit of routing context (a patrol number, a klan, nothing).
//
// The size is the *ordered* size from the order line's attributes, not the size on
// the member's own record: an order can differ from the profile, and the shirt that
// gets bagged is the one that was paid for. Only tshirt.adult exists today; if a
// second t-shirt SKU is ever added, widen the predicate to `productSku LIKE 'tshirt%'`.
//
// order_line has no year of its own, so each list is scoped by its member table's
// year instead — which also is what keeps a memberId that recurs across years from
// matching the wrong edition.
//
// Only lines belonging to a *paid* order are counted: a shirt is bagged when it has
// been paid for, so open (still-payable) and cancelled orders are excluded. This is
// what keeps the picking list's totals in step with the order summary, which likewise
// disregards unpaid lines.

// tshirtLines is the shared per-member t-shirt tally: one row per (member, size) with
// the summed quantity. Joined to each member table below.
const tshirtLines = `
	SELECT ol.memberId,
	       JSON_UNQUOTE(JSON_EXTRACT(ol.attributes, '$.size')) AS size,
	       SUM(ol.quantity) AS quantity
	FROM order_line ol
	JOIN orders o ON o.orderId = ol.orderId AND o.status = 'paid'
	WHERE ol.productSku = 'tshirt.adult'
	GROUP BY ol.memberId, size
	HAVING SUM(ol.quantity) > 0`

func (app *application) excelPluklisteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	year := app.YearSlug(r)

	spejdere, err := app.pluklisteRows(ctx, `
		SELECT sp.name, sp.phone, t.quantity, t.size,
		       COALESCE(p.teamNumber, ''), COALESCE(p.name, '')
		FROM (`+tshirtLines+`) t
		JOIN spejder sp ON sp.memberId = t.memberId AND sp.year = ?
		LEFT JOIN patrulje p ON p.teamId = sp.teamId
		ORDER BY p.teamNumber, sp.name`, year, 6)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	seniorer, err := app.pluklisteRows(ctx, `
		SELECT se.name, se.phone, t.quantity, t.size, COALESCE(k.name, '')
		FROM (`+tshirtLines+`) t
		JOIN senior se ON se.memberId = t.memberId AND se.year = ?
		LEFT JOIN klan k ON k.teamId = se.teamId
		ORDER BY k.name, se.name`, year, 5)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	goglere, err := app.pluklisteRows(ctx, `
		SELECT pe.name, pe.phone, t.quantity, t.size
		FROM (`+tshirtLines+`) t
		JOIN personnel pe ON pe.userId = t.memberId
		WHERE pe.userType = 'gøgler' AND pe.year = ?
		ORDER BY pe.name`, year, 4)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	crew, err := app.pluklisteRows(ctx, `
		SELECT c.name, c.phone, t.quantity, t.size,
		       COALESCE(NULLIF(s.label, ''), c.sectionSlug) AS section
		FROM (`+tshirtLines+`) t
		JOIN crewmember c ON c.userId = t.memberId AND c.deleted = 0
		LEFT JOIN section s ON s.slug = c.sectionSlug AND s.year = c.year
		WHERE c.year = ?
		ORDER BY section, c.name`, year, 5)
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

	base := []string{"Navn", "Telefon", "Antal", "Størrelse"}
	baseW := []int{30, 15, 8, 12}

	writePluklisteSheet(xlsx, "Sheet1", styleTitle, view,
		append(base, "Patrulje nr.", "Patrulje"), append(baseW, 12, 30), spejdere)
	xlsx.SetSheetName("Sheet1", "Spejdere")

	xlsx.NewSheet("Seniorer")
	writePluklisteSheet(xlsx, "Seniorer", styleTitle, view,
		append(base, "Klan"), append(baseW, 30), seniorer)

	xlsx.NewSheet("Gøglere")
	writePluklisteSheet(xlsx, "Gøglere", styleTitle, view, base, baseW, goglere)

	xlsx.NewSheet("Crew")
	writePluklisteSheet(xlsx, "Crew", styleTitle, view,
		append(base, "Sektion"), append(baseW, 30), crew)

	writeXlsx(app, w, r, xlsx, "plukliste")
}

// pluklisteRows runs one list query and returns its rows as spreadsheet cells. cols is
// how many columns the SELECT returns; it is passed rather than inferred so a mismatch
// between the query and the sheet's headers fails loudly here rather than silently
// dropping a column.
func (app *application) pluklisteRows(ctx context.Context, query string, year types.YearSlug, cols int) ([][]any, error) {
	rows, err := app.db.DB().QueryContext(ctx, query, string(year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]any{}
	for rows.Next() {
		cells := make([]any, cols)
		dest := make([]any, cols)
		for i := range cells {
			dest[i] = &cells[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		// []byte from the driver renders as an unreadable byte array in a cell; the text
		// columns come back that way, so decode them to strings.
		for i, c := range cells {
			if b, ok := c.([]byte); ok {
				cells[i] = string(b)
			}
		}
		out = append(out, cells)
	}
	return out, rows.Err()
}

// writePluklisteSheet lays out one sheet: a styled header row with column widths, then
// the data. Columns stay within A–Z, which the picking list's handful of columns never
// exceeds.
func writePluklisteSheet(xlsx *excelize.File, sheet string, style int, view *excelize.ViewOptions, headers []string, widths []int, rows [][]any) {
	xlsx.SetSheetView(sheet, 0, view)
	xlsx.SetRowStyle(sheet, 1, 1, style)
	for i, h := range headers {
		col := string(rune(65 + i))
		xlsx.SetCellValue(sheet, fmt.Sprintf("%s%d", col, 1), h)
		xlsx.SetColWidth(sheet, col, col, float64(widths[i]))
	}
	for r, row := range rows {
		for c, val := range row {
			xlsx.SetCellValue(sheet, fmt.Sprintf("%s%d", string(rune(65+c)), r+2), val)
		}
	}
}
