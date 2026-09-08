package inventory

type Inventory struct {
	ProductID         int64 `json:"product_id"`
	AvailableQuantity int   `json:"available_quantity"`
	ReservedQuantity  int   `json:"reserved_quantity"`
}

type ReservationItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type InventoryOrderPayload struct {
	OrderID int64             `json:"order_id"`
	UserID  int64             `json:"user_id"`
	Items   []ReservationItem `json:"items"`
}

type InventoryResultPayload struct {
	OrderID int64
	UserID  int64  `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}
