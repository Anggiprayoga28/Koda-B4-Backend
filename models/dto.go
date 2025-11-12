package models

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type UpdateUserRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
}

type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CategoryID  int    `json:"category_id"`
	Price       int    `json:"price"`
	Stock       int    `json:"stock"`
}

type UpdateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CategoryID  int    `json:"category_id"`
	Price       int    `json:"price"`
	Stock       int    `json:"stock"`
	IsActive    bool   `json:"is_active"`
}

type CreateOrderRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
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

type PaginationResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    MetaData    `json:"meta"`
}

type MetaData struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type LoginResponse struct {
	Token string          `json:"token"`
	User  UserWithProfile `json:"user"`
}

type DashboardStats struct {
	TotalOrders     int `json:"total_orders"`
	PendingOrders   int `json:"pending_orders"`
	ShippingOrders  int `json:"shipping_orders"`
	CompletedOrders int `json:"completed_orders"`
	TotalRevenue    int `json:"total_revenue"`
}

type CartItem struct {
	CartID, ProductID, Qty, Stock, SizeID, TempID, VariantID int
	Name                                                     string
	Price, SizeAdj, TempPrice, VariantPrice                  int
	IsFlashSale                                              bool
}

type CheckoutRequest struct {
	Email          string `json:"email"`
	FullName       string `json:"full_name"`
	Address        string `json:"address"`
	DeliveryMethod string `json:"delivery_method"`
	PaymentMethod  int    `json:"payment_method_id"`
}

type TransactionResponse struct {
	ID             int    `json:"id"`
	OrderNumber    string `json:"order_number"`
	Status         string `json:"status"`
	Subtotal       int    `json:"subtotal"`
	DeliveryFee    int    `json:"delivery_fee"`
	Total          int    `json:"total"`
	Email          string `json:"email"`
	FullName       string `json:"full_name"`
	Address        string `json:"address"`
	DeliveryMethod string `json:"delivery_method"`
}
