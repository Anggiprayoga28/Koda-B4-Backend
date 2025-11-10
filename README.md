# Coffee Shop Backend

REST API untuk sistem manajemen coffee shop yang dibangun dengan Go (Golang), Gin Framework, dan PostgreSQL.

## Entity Relationship Diagram (ERD)

```mermaid
erDiagram
    users {
        int id PK
        varchar email UK
        varchar password
        varchar role
        timestamp created_at
        timestamp updated_at
    }
    
    user_profiles {
        int id PK
        int user_id FK
        varchar full_name
        varchar phone
        text address
        varchar photo_url
        timestamp created_at
        timestamp updated_at
    }
    
    categories {
        int id PK
        varchar name UK
        varchar slug UK
        boolean is_active
        timestamp created_at
    }
    
    products {
        int id PK
        varchar name
        text description
        int category_id FK
        int price
        boolean is_flash_sale
        boolean is_favorite
        boolean is_buy1get1
        boolean is_active
        int stock
        timestamp created_at
        timestamp updated_at
    }
    
    product_images {
        int id PK
        int product_id FK
        varchar image_url
        boolean is_primary
        int display_order
        timestamp created_at
    }
    
    product_reviews {
        int id PK
        int product_id FK
        int user_id FK
        int rating
        text review_text
        timestamp created_at
    }
    
    promos {
        int id PK
        varchar code UK
        varchar title
        text description
        varchar bg_color
        int discount_percentage
        date start_date
        date end_date
        boolean is_active
        timestamp created_at
    }
    
    promo_products {
        int id PK
        int promo_id FK
        int product_id FK
        timestamp created_at
    }
    
    delivery_methods {
        int id PK
        varchar name UK
        int base_fee
        text description
        boolean is_active
        timestamp created_at
    }
    
    payment_methods {
        int id PK
        varchar name UK
        text description
        boolean is_active
        timestamp created_at
    }
    
    tax_rates {
        int id PK
        varchar name
        decimal rate_percentage
        boolean is_active
        timestamp created_at
    }
    
    orders {
        int id PK
        varchar order_number UK
        int user_id FK
        varchar status
        text delivery_address
        int delivery_method_id FK
        int subtotal
        int delivery_fee
        int tax_amount
        int tax_rate_id FK
        int total
        int promo_id FK
        int payment_method_id FK
        timestamp order_date
        timestamp created_at
        timestamp updated_at
    }
    
    order_items {
        int id PK
        int order_id FK
        int product_id FK
        int quantity
        varchar size
        varchar temperature
        int unit_price
        boolean is_flash_sale
        timestamp created_at
    }
    
    cart_items {
        int id PK
        int user_id FK
        int product_id FK
        int quantity
        varchar size
        varchar temperature
        timestamp created_at
        timestamp updated_at
    }

    users ||--o{ cart_items : "has"
    users ||--o{ orders : "places"
    users ||--o| user_profiles : "has"
    users ||--o{ product_reviews : "writes"
    product_reviews }o--|| products : "reviewed for"
    products ||--o{ product_images : "has"
    promos ||--o{ promo_products : "applies to"
    cart_items }o--|| products : "added to"
    promo_products }o--|| products : "included in"
    orders ||--o{ order_items : "contains"
    delivery_methods ||--o{ orders : "used for"
    payment_methods ||--o{ orders : "paid with"
    tax_rates ||--o{ orders : "taxed by"
    categories ||--o{ products : "categorizes"
```

## Fitur

- **Autentikasi & Otorisasi**
  - Register & Login
  - JWT Token
  - Role-based access (Admin & Customer)
  
- **Manajemen User**
  - CRUD User (Admin)
  - Update Profile
  - Upload Profile Photo
  - Change Password

- **Manajemen Produk**
  - CRUD Produk (Admin)
  - List Produk & Kategori (Public)
  - Pagination

- **Manajemen Order**
  - Create Order (Customer)
  - View & Update Order Status (Admin)
  - Dashboard Statistics (Admin)

## Tech Stack

- **Backend**: Golang + Gin Framework
- **Database**: PostgreSQL
- **Authentication**: JWT
- **Password Hashing**: Argon2
- **API Documentation**: Swagger



## Endpoints

### Public Endpoints
- `POST /auth/register` - Register user baru
- `POST /auth/login` - Login user
- `GET /categories` - List kategori
- `GET /products` - List produk
- `GET /products/:id` - Detail produk

### Authenticated Endpoints (Customer)
- `GET /auth/profile` - Get profile
- `PATCH /auth/profile` - Update profile
- `POST /auth/profile/photo` - Upload profile photo
- `POST /auth/change-password` - Change password
- `POST /orders` - Create order

### Admin Endpoints
- `GET /admin/dashboard` - Dashboard statistics
- `GET /admin/users` - List users
- `POST /admin/users` - Create user
- `PATCH /admin/users/:id` - Update user
- `DELETE /admin/users/:id` - Delete user
- `POST /admin/products` - Create product
- `PATCH /admin/products/:id` - Update product
- `DELETE /admin/products/:id` - Delete product
- `GET /admin/orders` - List orders
- `GET /admin/orders/:id` - Detail order
- `PATCH /admin/orders/:id/status` - Update order status

## Authentication

Semua endpoint yang memerlukan autentikasi harus menyertakan JWT token di header:

```
Authorization: Bearer <your_token_here>
```

## Project Structure

```
coffee-shop/
├── controllers/        # Request handlers
├── middleware/         # Middleware (Auth, CORS)
├── models/            # Data models & database
├── routes/            # Route definitions
├── uploads/           # Upload directory
├── docs/              # Swagger documentation
├── main.go            # Entry point
└── .env               # Environment variables
```

## Response Format

### Success Response
```json
{
  "success": true,
  "message": "Success message",
  "data": {}
}
```

### Error Response
```json
{
  "success": false,
  "message": "Error message"
}
```

### Pagination Response
```json
{
  "success": true,
  "message": "Data retrieved",
  "data": [],
  "meta": {
    "page": 1,
    "limit": 10,
    "total_items": 50,
    "total_pages": 5
  }
}
```
