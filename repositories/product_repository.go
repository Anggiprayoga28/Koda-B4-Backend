package repositories

import (
	"coffee-shop/config"
	"coffee-shop/models"
	"context"
	"time"
)

type ProductRepository struct{}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) GetAllCategories() ([]models.Category, error) {
	query := `SELECT id, name, is_active, created_at FROM categories ORDER BY name`

	rows, err := config.DB.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []models.Category{}
	for rows.Next() {
		var cat models.Category
		rows.Scan(&cat.ID, &cat.Name, &cat.IsActive, &cat.CreatedAt)
		categories = append(categories, cat)
	}
	return categories, nil
}

func (r *ProductRepository) CreateProduct(product *models.Product) error {
	query := `
		INSERT INTO products (name, description, category_id, price, stock, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, $6, $7)
		RETURNING id, created_at, updated_at
	`
	now := time.Now()
	return config.DB.QueryRow(context.Background(), query,
		product.Name, product.Description, product.CategoryID, product.Price, product.Stock, now, now,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}

func (r *ProductRepository) GetAllProducts(page, limit int) ([]models.Product, int, error) {
	offset := (page - 1) * limit

	var total int
	config.DB.QueryRow(context.Background(), `SELECT COUNT(*) FROM products WHERE is_active = true`).Scan(&total)

	query := `SELECT id, name, description, category_id, price, stock, is_active, created_at, updated_at 
	          FROM products WHERE is_active = true ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := config.DB.Query(context.Background(), query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		products = append(products, p)
	}
	return products, total, nil
}

func (r *ProductRepository) GetProductByID(id int) (*models.Product, error) {
	query := `SELECT id, name, description, category_id, price, stock, is_active, created_at, updated_at 
	          FROM products WHERE id = $1`

	var p models.Product
	err := config.DB.QueryRow(context.Background(), query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	return &p, err
}

func (r *ProductRepository) UpdateProduct(product *models.Product) error {
	query := `UPDATE products SET name = $1, description = $2, category_id = $3, price = $4, 
	          stock = $5, is_active = $6, updated_at = $7 WHERE id = $8`
	_, err := config.DB.Exec(context.Background(), query,
		product.Name, product.Description, product.CategoryID, product.Price,
		product.Stock, product.IsActive, time.Now(), product.ID,
	)
	return err
}

func (r *ProductRepository) DeleteProduct(id int) error {
	query := `UPDATE products SET is_active = false WHERE id = $1`
	_, err := config.DB.Exec(context.Background(), query, id)
	return err
}
