package notification

import "time"

type Notification struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    int64     `json:"user_id" bson:"user_id"`
	OrderID   int64     `json:"order_id" bson:"order_id"`
	Type      string    `json:"type" bson:"type"`
	Message   string    `json:"message" bson:"message"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
