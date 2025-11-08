package services

import (
	"coffee-shop/models"
	"coffee-shop/repositories"
	"errors"
	"math"
)

type ProductService struct {
	productRepo *repositories.ProductRepository
}

func NewProductService() *ProductService {
	return &ProductService{
		productRepo: repositories.NewProductRepository(),
	}
}

func (s *ProductService) GetAllCategories() ([]models.Category, error) {
	return s.productRepo.GetAllCategories()
}

func (s *ProductService) GetAllProducts(page, limit int) (*models.PaginationResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	products, total, err := s.productRepo.GetAllProducts(page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &models.PaginationResponse{
		Success: true,
		Message: "Products retrieved successfully",
		Data:    products,
		Meta: models.MetaData{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *ProductService) GetProductByID(id int) (*models.Product, error) {
	return s.productRepo.GetProductByID(id)
}

func (s *ProductService) CreateProduct(req models.CreateProductRequest) (*models.Product, error) {
	product := &models.Product{
		Name:        req.Name,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Price:       req.Price,
		Stock:       req.Stock,
		IsActive:    true,
	}

	if err := s.productRepo.CreateProduct(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) UpdateProduct(id int, req models.UpdateProductRequest) (*models.Product, error) {
	product, err := s.productRepo.GetProductByID(id)
	if err != nil {
		return nil, errors.New("product not found")
	}

	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.CategoryID > 0 {
		product.CategoryID = req.CategoryID
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Stock >= 0 {
		product.Stock = req.Stock
	}
	product.IsActive = req.IsActive

	if err := s.productRepo.UpdateProduct(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) DeleteProduct(id int) error {
	return s.productRepo.DeleteProduct(id)
}
