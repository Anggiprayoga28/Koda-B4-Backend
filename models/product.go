package models

import "time"

type Product struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CategoryID   int       `json:"category_id"`
	Price        int       `json:"price"`
	Stock        int       `json:"stock"`
	ImageURL     string    `json:"image_url"`
	CloudinaryID string    `json:"cloudinary_id,omitempty"`
	IsFlashSale  bool      `json:"is_flash_sale"`
	IsFavorite   bool      `json:"is_favorite"`
	IsBuy1Get1   bool      `json:"is_buy1get1"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
