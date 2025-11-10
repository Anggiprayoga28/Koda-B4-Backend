package models

import "time"

type Order struct {
	ID          int       `json:"id"`
	OrderNumber string    `json:"order_number"`
	UserID      int       `json:"user_id"`
	Status      string    `json:"status"`
	Total       int       `json:"total"`
	CreatedAt   time.Time `json:"created_at"`
}
