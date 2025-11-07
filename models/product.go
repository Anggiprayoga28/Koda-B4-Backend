package models

import "time"

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Price       float64   `json:"price"`
	Category    *string   `json:"category,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	Stock       int       `json:"stock"`
	IsAvailable bool      `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
