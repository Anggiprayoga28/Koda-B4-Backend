package services

import (
	"coffee-shop/models"
	"coffee-shop/repositories"
	"coffee-shop/utils"
	"errors"
	"math"
)

type UserService struct {
	userRepo *repositories.UserRepository
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: repositories.NewUserRepository(),
	}
}

func (s *UserService) GetAllUsers(page, limit int) (*models.PaginationResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	users, totalItems, err := s.userRepo.FindAll(page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	return &models.PaginationResponse{
		Success: true,
		Message: "Users retrieved successfully",
		Data:    users,
		Meta: models.MetaData{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *UserService) GetUserByID(id int) (*models.UserWithProfile, error) {
	return s.userRepo.GetUserWithProfile(id)
}

func (s *UserService) CreateUser(req models.CreateUserRequest) (*models.UserWithProfile, error) {
	existingUser, _ := s.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:    req.Email,
		Password: hashedPassword,
		Role:     req.Role,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	profile := &models.UserProfile{
		UserID:   user.ID,
		FullName: req.FullName,
		Phone:    req.Phone,
	}

	if err := s.userRepo.CreateProfile(profile); err != nil {
		return nil, err
	}

	return s.userRepo.GetUserWithProfile(user.ID)
}

func (s *UserService) UpdateUser(id int, req models.UpdateUserRequest) (*models.UserWithProfile, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if req.Email != "" && req.Email != user.Email {
		existingUser, _ := s.userRepo.FindByEmail(req.Email)
		if existingUser != nil {
			return nil, errors.New("email already registered")
		}
		user.Email = req.Email
	}

	if req.Role != "" {
		user.Role = req.Role
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	if req.FullName != "" || req.Phone != "" || req.Address != "" {
		profile, err := s.userRepo.GetProfile(id)
		if err != nil {
			return nil, err
		}

		if req.FullName != "" {
			profile.FullName = req.FullName
		}
		if req.Phone != "" {
			profile.Phone = req.Phone
		}
		if req.Address != "" {
			profile.Address = req.Address
		}

		if err := s.userRepo.UpdateProfile(profile); err != nil {
			return nil, err
		}
	}

	return s.userRepo.GetUserWithProfile(id)
}

func (s *UserService) DeleteUser(id int) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user not found")
	}

	profile, err := s.userRepo.GetProfile(id)
	if err == nil && profile.PhotoURL != "" {
		utils.DeleteFile(profile.PhotoURL)
	}

	return s.userRepo.Delete(user.ID)
}
