package models

import "time"

type CartItem struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	ProductID   int       `json:"product_id"`
	Product     *Product  `json:"product,omitempty"`
	Quantity    int       `json:"quantity"`
	Size        *string   `json:"size,omitempty"`
	Temperature *string   `json:"temperature,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
