package order_items

import (
	"strings"
	"testing"
	"time"
)

func TestOrderItemColumnsMatchScanTargets(t *testing.T) {
	cols := splitColumns(orderItemColumns)
	targets := len(orderItemScanTargets(&OrderItem{}))

	if len(cols) != targets {
		t.Fatalf("orderItemColumns has %d columns but orderItemScanTargets has %d entries; "+
			"they must stay aligned or Scan misaligns", len(cols), targets)
	}
}

func TestOrderItemScanTargetOrder(t *testing.T) {
	// Expected column -> field name at each position.
	wantCols := []string{
		"id::text", "order_id::text", "product_id::text", "product_name",
		"product_description", "product_emoji", "quantity", "unit_price_cents",
		"total_price_cents", "created_at",
	}
	cols := splitColumns(orderItemColumns)
	if len(cols) != len(wantCols) {
		t.Fatalf("orderItemColumns count: got %d, want %d", len(cols), len(wantCols))
	}
	for i := range wantCols {
		if cols[i] != wantCols[i] {
			t.Fatalf("column %d: got %q, want %q", i, cols[i], wantCols[i])
		}
	}

	var it OrderItem
	targets := orderItemScanTargets(&it)
	if len(targets) != len(wantCols) {
		t.Fatalf("targets count: got %d, want %d", len(targets), len(wantCols))
	}
	set(targets[0], "id-0")
	set(targets[1], "order-1")
	set(targets[2], "product-2")
	set(targets[3], "name-3")
	set(targets[4], "desc-4")
	set(targets[5], "emoji-5")
	setInt(targets[6], 6)
	setInt(targets[7], 7)
	setInt(targets[8], 8)
	setTime(targets[9], time.Unix(9, 0))

	checks := []struct {
		col  string
		got  any
		want any
	}{
		{"id::text", it.ID, "id-0"},
		{"order_id::text", it.OrderID, "order-1"},
		{"product_id::text", derefStr(it.ProductID), "product-2"},
		{"product_name", it.ProductName, "name-3"},
		{"product_description", it.ProductDescription, "desc-4"},
		{"product_emoji", it.ProductEmoji, "emoji-5"},
		{"quantity", it.Quantity, 6},
		{"unit_price_cents", it.UnitPriceCents, 7},
		{"total_price_cents", it.TotalPriceCents, 8},
		{"created_at", it.CreatedAt.Unix(), int64(9)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("column %q maps to the wrong field: got %v, want %v", c.col, c.got, c.want)
		}
	}
}

func set(target any, v string) {
	switch p := target.(type) {
	case *string:
		*p = v
	case **string:
		*p = &v
	}
}
func setInt(target any, v int)        { *(target.(*int)) = v }
func setTime(target any, v time.Time) { *(target.(*time.Time)) = v }
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
