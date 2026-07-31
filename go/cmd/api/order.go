package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	tables "nathejk.dk/nathejk/table"
	"nathejk.dk/nathejk/table/order"
)

// orderView is an order.Order enriched with a human-readable owner label.
// The embedded order.Order fields are promoted onto the JSON object; the
// ownerName is resolved by a secondary lookup (see resolveOwnerName) rather
// than a SQL join in the orders query.
type orderView struct {
	order.Order
	OwnerName string `json:"ownerName"`
}

// listOrdersHandler lists all orders for the active year.
//
// @Summary     List orders for the active year
// @Description Returns every order for the year given by the X-YearSlug header (or the current year), each with a resolved owner label, totals and payment status (OPEN/PAID). Line items are not included — fetch a single order for those.
// @Tags        orders
// @Produce     json
// @Success     200 {object} map[string]interface{} "envelope with an \"orders\" array"
// @Failure     500 {object} map[string]interface{}
// @Router      /api/orders [get]
func (app *application) listOrdersHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)
	orders, err := app.models.Order.ListByYear(r.Context(), year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	cache := map[string]string{}
	views := make([]orderView, 0, len(orders))
	for _, o := range orders {
		views = append(views, orderView{
			Order:     o,
			OwnerName: app.resolveOwnerName(r.Context(), cache, o.OwnerType, o.OwnerID),
		})
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"orders": views}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// showOrderHandler returns a single order with its line items.
//
// @Summary     Get a single order
// @Description Returns one order (including its line items and computed paid/due amounts) with a resolved owner label.
// @Tags        orders
// @Produce     json
// @Param       id path string true "Order ID"
// @Success     200 {object} map[string]interface{} "envelope with an \"order\" object"
// @Failure     404 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/order/{id} [get]
func (app *application) showOrderHandler(w http.ResponseWriter, r *http.Request) {
	id := app.ReadNamedParam(r, "id")
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	o, err := app.models.Order.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, tables.ErrRecordNotFound) {
			app.NotFoundResponse(w, r)
			return
		}
		app.ServerErrorResponse(w, r, err)
		return
	}

	view := orderView{
		Order:     *o,
		OwnerName: app.resolveOwnerName(r.Context(), map[string]string{}, o.OwnerType, o.OwnerID),
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"order": view}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// resolveOwnerName resolves a human-readable label for an order owner via a
// secondary lookup against the patrulje / klan / personnel read models — never
// a SQL join in the orders query. Results are memoised in cache (keyed by
// "ownerType\x00ownerId") so a year with many orders for the same owner does
// only one lookup per owner. On any miss it falls back to the raw owner id.
func (app *application) resolveOwnerName(ctx context.Context, cache map[string]string, ownerType types.TeamType, ownerID string) string {
	if ownerID == "" {
		return ""
	}
	key := string(ownerType) + "\x00" + ownerID
	if name, ok := cache[key]; ok {
		return name
	}

	name := ownerID
	switch ownerType {
	case types.TeamTypePatrulje:
		if p, err := app.models.Patrulje.GetByID(ctx, types.TeamID(ownerID)); err == nil && p != nil {
			if p.TeamNumber != "" {
				name = p.TeamNumber + " - " + p.Name
			} else if p.Name != "" {
				name = p.Name
			}
		}
	case types.TeamTypeKlan:
		if k, err := app.models.Klan.GetByID(ctx, types.TeamID(ownerID)); err == nil && k != nil && k.Name != "" {
			name = k.Name
		}
	default:
		// crew / gøgler and any other owner type are personnel users.
		if pers, err := app.models.Personnel.GetByID(ctx, types.UserID(ownerID)); err == nil && pers != nil && pers.Name != "" {
			name = pers.Name
		}
	}

	cache[key] = name
	return name
}
