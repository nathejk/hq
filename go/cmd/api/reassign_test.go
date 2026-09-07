package main

import (
	"testing"

	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/types"
)

// A transfer's charge order is closed by the payment saga, which runs in another service.
// Until it does, the member's seat sits on an `open` order and is invisible to
// PaidLinesByMember — so a second move would transfer nothing and report that nobody had
// paid for them (task 166). These tests pin the detection that refuses instead.
func TestPendingTransferOrder(t *testing.T) {
	const member = types.MemberID("m-1")

	transferLine := func(memberID string) order.Line {
		return order.Line{LineID: "transfer:ABC123:0", MemberID: memberID}
	}
	derivedLine := func(memberID string) order.Line {
		return order.Line{LineID: "derived:participation.patrulje:" + memberID, MemberID: memberID}
	}

	tests := []struct {
		name   string
		orders []order.Order
		want   string
	}{
		{
			name: "unsettled transfer holding this member's seat",
			orders: []order.Order{
				{OrderID: "o-1", Status: order.StatusOpen, Lines: []order.Line{transferLine("m-1")}},
			},
			want: "o-1",
		},
		{
			// The whole point: once it is paid the seat is visible again and a move is safe.
			name: "settled transfer is not pending",
			orders: []order.Order{
				{OrderID: "o-1", Status: order.StatusPaid, Lines: []order.Line{transferLine("m-1")}},
			},
			want: "",
		},
		{
			// An ordinary unpaid order is not a transfer. Refusing on it would block every
			// member of a team that simply has an outstanding bill.
			name: "open non-transfer order is not pending",
			orders: []order.Order{
				{OrderID: "o-1", Status: order.StatusOpen, Lines: []order.Line{derivedLine("m-1")}},
			},
			want: "",
		},
		{
			// Teams share orders; another member's in-flight transfer must not freeze this one.
			name: "unsettled transfer for a different member",
			orders: []order.Order{
				{OrderID: "o-1", Status: order.StatusOpen, Lines: []order.Line{transferLine("m-2")}},
			},
			want: "",
		},
		{
			name: "cancelled transfer is not pending",
			orders: []order.Order{
				{OrderID: "o-1", Status: order.StatusCancelled, Lines: []order.Line{transferLine("m-1")}},
			},
			want: "",
		},
		{
			name: "found among several orders and lines",
			orders: []order.Order{
				{OrderID: "o-1", Status: order.StatusPaid, Lines: []order.Line{derivedLine("m-1")}},
				{OrderID: "o-2", Status: order.StatusOpen, Lines: []order.Line{derivedLine("m-9"), transferLine("m-1")}},
			},
			want: "o-2",
		},
		{
			name:   "no orders at all",
			orders: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pendingTransferOrder(tt.orders, member); got != tt.want {
				t.Errorf("pendingTransferOrder() = %q, want %q", got, tt.want)
			}
		})
	}
}
