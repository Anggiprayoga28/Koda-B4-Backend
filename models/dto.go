package models

type RegisterRequest struct {
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required,min=6"`
	FullName string `json:"full_name" form:"full_name" binding:"required,min=3"`
	Phone    string `json:"phone" form:"phone" binding:"omitempty"`
	Role     string `json:"role" form:"role" binding:"omitempty,oneof=customer admin"`
}

type LoginRequest struct {
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name" form:"full_name"`
	Phone    string `json:"phone" form:"phone"`
	Address  string `json:"address" form:"address"`
}

type UpdateUserRequest struct {
	Email    string `json:"email" form:"email"`
	Role     string `json:"role" form:"role"`
	FullName string `json:"full_name" form:"full_name"`
	Phone    string `json:"phone" form:"phone"`
	Address  string `json:"address" form:"address"`
}

type CreateProductRequest struct {
	Name         string `json:"name" form:"name" binding:"required"`
	Description  string `json:"description" form:"description" binding:"required"`
	CategoryID   int    `json:"category_id" form:"category_id" binding:"required"`
	Price        int    `json:"price" form:"price" binding:"required"`
	Stock        int    `json:"stock" form:"stock" binding:"required"`
	IsFlashSale  bool   `json:"is_flash_sale" form:"is_flash_sale"`
	IsFavorite   bool   `json:"is_favorite" form:"is_favorite"`
	IsBuy1Get1   bool   `json:"is_buy1get1" form:"is_buy1get1"`
	IsActive     bool   `json:"is_active" form:"is_active"`
	ImageURL     string `json:"image_url" form:"image_url"`
	CloudinaryID string `json:"cloudinary_id" form:"cloudinary_id"`
}

type UpdateProductRequest struct {
	Name         string `json:"name" form:"name"`
	Description  string `json:"description" form:"description"`
	CategoryID   int    `json:"category_id" form:"category_id"`
	Price        int    `json:"price" form:"price"`
	Stock        int    `json:"stock" form:"stock"`
	IsFlashSale  bool   `json:"is_flash_sale" form:"is_flash_sale"`
	IsFavorite   bool   `json:"is_favorite" form:"is_favorite"`
	IsBuy1Get1   bool   `json:"is_buy1get1" form:"is_buy1get1"`
	IsActive     bool   `json:"is_active" form:"is_active"`
	ImageURL     string `json:"image_url" form:"image_url"`
	CloudinaryID string `json:"cloudinary_id" form:"cloudinary_id"`
}

type CheckoutItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type CheckoutRequest struct {
	Email          string                `json:"email"`
	FullName       string                `json:"full_name"`
	Address        string                `json:"address"`
	DeliveryMethod string                `json:"delivery_method"`
	PaymentMethod  string                `json:"payment_method"`
	Notes          string                `json:"notes"`
	Items          []CheckoutItemRequest `json:"items"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type MetaData struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type PaginationResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    MetaData    `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type PaginationLinks struct {
	Self string `json:"self"`
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

type HATEOASResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    interface{}     `json:"data"`
	Meta    PaginationMeta  `json:"meta"`
	Links   PaginationLinks `json:"links"`
}
