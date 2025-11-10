package models

type Category struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type CategoryRequest struct {
	Name string `json:"name" binding:"required"`
}
