package orders

import (
	"strings"
	"testing"
	"time"
)

func TestOrderColumnsMatchScanTargets(t *testing.T) {
	cols := splitColumns(orderColumns)
	targets := len(orderScanTargets(&Order{}))

	if len(cols) != targets {
		t.Fatalf("orderColumns has %d columns but orderScanTargets has %d entries; "+
			"they must stay aligned or Scan misaligns", len(cols), targets)
	}
}

func TestOrderScanTargetOrder(t *testing.T) {
	wantCols := []string{
		"id::text", "public_code", "client_user_id::text", "restaurant_id::text",
		"delivery_address", "status", "subtotal_cents", "delivery_fee_cents",
		"discount_cents", "total_cents", "robot_dispatched", "gateway_mode",
		"mqtt_connected", "placed_at", "dispatched_at", "completed_at",
		"cancelled_at", "cancel_reason", "notes", "created_at", "updated_at",
	}
	cols := splitColumns(orderColumns)
	if len(cols) != len(wantCols) {
		t.Fatalf("orderColumns count: got %d, want %d", len(cols), len(wantCols))
	}
	for i := range wantCols {
		if cols[i] != wantCols[i] {
			t.Fatalf("column %d: got %q, want %q", i, cols[i], wantCols[i])
		}
	}

	var o Order
	tg := orderScanTargets(&o)
	if len(tg) != len(wantCols) {
		t.Fatalf("targets count: got %d, want %d", len(tg), len(wantCols))
	}
	setStr(tg[0], "id-0")
	setStr(tg[1], "code-1")
	setStr(tg[2], "client-2")
	setStr(tg[3], "rest-3")
	setStr(tg[4], "addr-4")
	setStr(tg[5], "status-5")
	setInt(tg[6], 6)
	setInt(tg[7], 7)
	setInt(tg[8], 8)
	setInt(tg[9], 9)
	setBool(tg[10], true)
	setStr(tg[11], "mode-11")
	setBool(tg[12], true)
	setTime(tg[13], time.Unix(13, 0))
	setTime(tg[14], time.Unix(14, 0))
	setTime(tg[15], time.Unix(15, 0))
	setTime(tg[16], time.Unix(16, 0))
	setStr(tg[17], "reason-17")
	setStr(tg[18], "notes-18")
	setTime(tg[19], time.Unix(19, 0))
	setTime(tg[20], time.Unix(20, 0))

	checks := []struct {
		col  string
		got  any
		want any
	}{
		{"id::text", o.ID, "id-0"},
		{"public_code", o.PublicCode, "code-1"},
		{"client_user_id::text", o.ClientUserID, "client-2"},
		{"restaurant_id::text", o.RestaurantID, "rest-3"},
		{"delivery_address", o.DeliveryAddress, "addr-4"},
		{"status", o.Status, "status-5"},
		{"subtotal_cents", o.SubtotalCents, 6},
		{"delivery_fee_cents", o.DeliveryFeeCents, 7},
		{"discount_cents", o.DiscountCents, 8},
		{"total_cents", o.TotalCents, 9},
		{"robot_dispatched", o.RobotDispatched, true},
		{"gateway_mode", derefStr(o.GatewayMode), "mode-11"},
		{"mqtt_connected", o.MQTTConnected, true},
		{"cancel_reason", derefStr(o.CancelReason), "reason-17"},
		{"notes", derefStr(o.Notes), "notes-18"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("column %q maps to the wrong field: got %v, want %v", c.col, c.got, c.want)
		}
	}
}

func setStr(target any, v string) {
	switch p := target.(type) {
	case *string:
		*p = v
	case **string:
		*p = &v
	}
}
func setInt(target any, v int)   { *(target.(*int)) = v }
func setBool(target any, v bool) { *(target.(*bool)) = v }
func setTime(target any, v time.Time) {
	switch p := target.(type) {
	case *time.Time:
		*p = v
	case **time.Time:
		*p = &v
	}
}
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// splitColumns returns the trimmed comma-separated columns of a SELECT fragment.
func splitColumns(cols string) []string {
	out := make([]string, 0)
	for _, c := range strings.Split(cols, ",") {
		if s := strings.TrimSpace(c); s != "" {
			out = append(out, s)
		}
	}
	return out
}
