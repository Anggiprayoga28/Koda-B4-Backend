package models

import "time"

type Order struct {
	ID            int         `json:"id"`
	UserID        int         `json:"user_id"`
	TotalAmount   float64     `json:"total_amount"`
	Status        string      `json:"status"`
	PaymentMethod *string     `json:"payment_method,omitempty"`
	Notes         *string     `json:"notes,omitempty"`
	Items         []OrderItem `json:"items,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID          int     `json:"id"`
	OrderID     int     `json:"order_id"`
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name,omitempty"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Size        *string `json:"size,omitempty"`
	Temperature *string `json:"temperature,omitempty"`
}
