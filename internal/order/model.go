package order

import "time"

type Order struct {
	ID          int64       `json:"id"`
	UserID      int64       `json:"user_id"`
	TotalAmount float64     `json:"total_amount"`
	Status      string      `json:"status"`
	Items       []OrderItem `json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type CreateOrderRequest struct {
	Items []CreateOrderItem `json:"items"`
}

type CreateOrderItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type OrderCreatedPayload struct {
	OrderID int64       `json:"order_id"`
	UserID  int64       `json:"user_id"`
	Items   []OrderItem `json:"items"`
}

type InventoryResultPayload struct {
	OrderID int64
	UserID  int64  `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}
