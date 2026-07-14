package order_items

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrItemNotFound = errors.New("order item not found")
)

// Repository handles database operations for order items
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// orderItemColumns is the canonical column list for order_items, in the order
// scanOrderItem reads them. Used in every SELECT and RETURNING so the projection
// and the scan can never drift apart.
const orderItemColumns = `
	id::text,
	order_id::text,
	product_id::text,
	product_name,
	product_description,
	product_emoji,
	quantity,
	unit_price_cents,
	total_price_cents,
	created_at`

// rowScanner is satisfied by both pgx.Row (QueryRow) and pgx.Rows (Query loop).
type rowScanner interface {
	Scan(dest ...any) error
}

// orderItemScanTargets returns pointers to i's fields in the SAME order as
// orderItemColumns. Count asserted in TestOrderItemColumnsMatchScanTargets.
func orderItemScanTargets(i *OrderItem) []any {
	return []any{
		&i.ID,
		&i.OrderID,
		&i.ProductID,
		&i.ProductName,
		&i.ProductDescription,
		&i.ProductEmoji,
		&i.Quantity,
		&i.UnitPriceCents,
		&i.TotalPriceCents,
		&i.CreatedAt,
	}
}

// scanOrderItem scans a single order_items row (columns in orderItemColumns order).
func scanOrderItem(row rowScanner) (OrderItem, error) {
	var i OrderItem
	err := row.Scan(orderItemScanTargets(&i)...)
	return i, err
}

// CreateItems creates multiple order items in a transaction
// This method expects to be called within an existing transaction.
func (r *Repository) CreateItems(ctx context.Context, tx pgx.Tx, orderID string, items []CreateItemRequest) ([]OrderItem, error) {
	insertItemSQL := `
		INSERT INTO order_items (
			order_id,
			product_id,
			product_name,
			product_description,
			product_emoji,
			quantity,
			unit_price_cents,
			total_price_cents
		)
		SELECT 
			$1,
			$2,
			p.name,
			p.description,
			p.emoji,
			$3,
			p.price_cents,
			p.price_cents * $3
		FROM products p
		WHERE p.id = $2
		RETURNING ` + orderItemColumns

	createdItems := make([]OrderItem, 0, len(items))

	for _, item := range items {
		orderItem, err := scanOrderItem(tx.QueryRow(ctx, insertItemSQL,
			orderID,
			item.ProductID,
			item.Quantity,
		))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: product %s", ErrItemNotFound, item.ProductID)
			}
			return nil, fmt.Errorf("insert order item: %w", err)
		}
		createdItems = append(createdItems, orderItem)
	}

	return createdItems, nil
}

// GetItemsByOrderID retrieves all items for a specific order
func (r *Repository) GetItemsByOrderID(ctx context.Context, orderID string) ([]OrderItem, error) {
	itemsSQL := `SELECT ` + orderItemColumns + `
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, itemsSQL, orderID)
	if err != nil {
		return nil, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()

	items := make([]OrderItem, 0)
	for rows.Next() {
		item, err := scanOrderItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return items, nil
}

// GetItemsByOrderIDs retrieves all items for multiple orders in a single query
// This prevents N+1 query problems when loading multiple orders with their items
func (r *Repository) GetItemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]OrderItem, error) {
	if len(orderIDs) == 0 {
		return make(map[string][]OrderItem), nil
	}

	itemsSQL := `SELECT ` + orderItemColumns + `
		FROM order_items
		WHERE order_id = ANY($1)
		ORDER BY order_id, created_at ASC`

	rows, err := r.db.Query(ctx, itemsSQL, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("query order items by order ids: %w", err)
	}
	defer rows.Close()

	// Group items by order_id
	itemsByOrder := make(map[string][]OrderItem)
	for rows.Next() {
		item, err := scanOrderItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		itemsByOrder[item.OrderID] = append(itemsByOrder[item.OrderID], item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return itemsByOrder, nil
}

func (r *Repository) SubtotalFromCatalog(ctx context.Context, tx pgx.Tx, items []CreateItemRequest) (int, error) {
	subtotal := 0
	for _, item := range items {
		var priceCents int
		err := tx.QueryRow(ctx,
			`SELECT price_cents FROM products WHERE id = $1`, item.ProductID,
		).Scan(&priceCents)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, fmt.Errorf("%w: product %s", ErrItemNotFound, item.ProductID)
			}
			return 0, fmt.Errorf("lookup product price: %w", err)
		}
		subtotal += priceCents * item.Quantity
	}
	return subtotal, nil
}
