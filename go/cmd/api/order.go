package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	tables "nathejk.dk/nathejk/table"
)

// orderView is an order.Order enriched with a human-readable owner label and the
// payments that settled it.
// The embedded order.Order fields are promoted onto the JSON object; the
// ownerName is resolved by a secondary lookup (see resolveOwnerName) rather
// than a SQL join in the orders query.
type orderView struct {
	order.Order
	OwnerName string       `json:"ownerName"`
	Payments  []paymentRef `json:"payments"`
}

// paymentRef is what an operator needs to find a payment at the provider: who
// took it and under which reference. Amount and time are included because an
// order can be settled by more than one payment, and then a bare list of
// references says nothing about which is which.
//
// LinkedBy records how the payment was tied to the order, which the operator needs
// in order to trust the reference. See settledPaymentsFor.
type paymentRef struct {
	Provider  string `json:"provider"`
	Reference string `json:"reference"`
	Amount    int    `json:"amount"`
	Status    string `json:"status"`
	ChangedAt string `json:"changedAt"`
	LinkedBy  string `json:"linkedBy"`
}

const (
	// linkedByOrder: the payment names this order, so the reference is exact.
	linkedByOrder = "order"
	// linkedByOwner: the payment predates the order entity and names the team or
	// user, so it is attributed to the owner's orders rather than to one of them.
	linkedByOwner = "owner"
)

// settledPaymentsFor groups the year's payments by the order they settled.
//
// Two linkages, because the data has two eras. Payments created by the order flow
// put the order id in orderForeignKey. Older ones put a team or user id there and
// name the kind in orderType — and they are the majority: of 189 paid orders in the
// live 2026 data, only 44 are linked by order id while 151 are only reachable
// through their owner. Matching order ids alone would leave those rows claiming no
// payment exists while the status column says Betalt.
//
// An owner-linked payment cannot say *which* of that owner's orders it paid, so it
// is reported against each of them and flagged, rather than silently presented as
// if it were exact. Order-linked payments win where both exist.
//
// Only payments that actually secured money are included. Opening a payment page
// records a `requested` payment, and abandoning it leaves that record behind — the
// live data has orders carrying 33 of them. Listing those would bury the one
// reference an operator is looking for, and would suggest money was taken when
// none was.
//
// One query for the whole list rather than one per order: this feeds a table of
// every order in the year.
func (app *application) settledPaymentsFor(ctx context.Context, year types.YearSlug, orders []order.Order) (map[string][]paymentRef, error) {
	payments, err := app.models.Payment.GetAll(ctx, payment.Filter{Year: year})
	if err != nil {
		return nil, err
	}

	byOrderID := map[string][]paymentRef{}
	byOwnerID := map[string][]paymentRef{}
	for _, p := range payments {
		if p.OrderForeignKey == "" {
			continue
		}
		switch p.Status {
		case types.PaymentStatusReserved, types.PaymentStatusReceived:
		default:
			continue
		}
		ref := paymentRef{
			Provider:  p.Method,
			Reference: p.Reference,
			Amount:    p.Amount,
			Status:    string(p.Status),
			ChangedAt: p.ChangedAt,
		}
		if p.OrderType == payment.OrderTypeOrder {
			ref.LinkedBy = linkedByOrder
			byOrderID[p.OrderForeignKey] = append(byOrderID[p.OrderForeignKey], ref)
			continue
		}
		ref.LinkedBy = linkedByOwner
		byOwnerID[p.OrderForeignKey] = append(byOwnerID[p.OrderForeignKey], ref)
	}

	out := make(map[string][]paymentRef, len(orders))
	for _, o := range orders {
		if refs := byOrderID[o.OrderID]; len(refs) > 0 {
			out[o.OrderID] = refs
			continue
		}
		if refs := byOwnerID[o.OwnerID]; len(refs) > 0 {
			out[o.OrderID] = refs
		}
	}
	return out, nil
}

// listOrdersHandler lists all orders for the active year.
//
// @Summary     List orders for the active year
// @Description Returns every order for the year given by the X-YearSlug header (or the current year), each with a resolved owner label, totals, payment status (OPEN/PAID) and the payments that settled it.
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
	payments, err := app.settledPaymentsFor(r.Context(), year, orders)
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
			Payments:  payments[o.OrderID],
		})
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"orders": views}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// showOrderHandler returns a single order with its line items.
//
// @Summary     Get a single order
// @Description Returns one order (including its line items, computed paid/due amounts and the payments that settled it) with a resolved owner label.
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

	// Scoped to the order's own year, so this reads the same slice of payments the
	// list endpoint does rather than a second, differently-filtered set.
	payments, err := app.settledPaymentsFor(r.Context(), o.Year, []order.Order{*o})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	view := orderView{
		Order:     *o,
		OwnerName: app.resolveOwnerName(r.Context(), map[string]string{}, o.OwnerType, o.OwnerID),
		Payments:  payments[o.OrderID],
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
	case types.TeamTypeCrew:
		// Crew members are projected into their own table with a generated UUID
		// as userId — not into personnel.
		if m, err := app.models.CrewMember.GetByID(ctx, types.UserID(ownerID)); err == nil && m != nil && m.Name != "" {
			name = m.Name
			break
		}
		if pers, err := app.models.Personnel.GetByID(ctx, types.UserID(ownerID)); err == nil && pers != nil && pers.Name != "" {
			name = pers.Name
		}
	default:
		// gøgler and other person owners live in the personnel projection; fall
		// back to crewmember so a person owner resolves either way.
		if pers, err := app.models.Personnel.GetByID(ctx, types.UserID(ownerID)); err == nil && pers != nil && pers.Name != "" {
			name = pers.Name
			break
		}
		if m, err := app.models.CrewMember.GetByID(ctx, types.UserID(ownerID)); err == nil && m != nil && m.Name != "" {
			name = m.Name
		}
	}

	cache[key] = name
	return name
}
