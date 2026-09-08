package order

import "errors"

var ErrInvalidStatus = errors.New("invalid order status")

var allowedStatuses = map[string]bool{
	"CREATED":    true,
	"CONFIRMED":  true,
	"PROCESSING": true,
	"SHIPPED":    true,
	"DELIVERED":  true,
	"CANCELLED":  true,
}

func isAllowedStatus(status string) bool {
	return allowedStatuses[status]
}
