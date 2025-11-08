package services

import (
	"coffee-shop/models"
	"coffee-shop/repositories"
	"coffee-shop/utils"
	"errors"
)

type AuthService struct {
	userRepo *repositories.UserRepository
}

func NewAuthService() *AuthService {
	return &AuthService{
		userRepo: repositories.NewUserRepository(),
	}
}

func (s *AuthService) Register(req models.RegisterRequest) (*models.LoginResponse, error) {
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
		Role:     "customer",
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

	token, err := utils.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	userWithProfile, err := s.userRepo.GetUserWithProfile(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token: token,
		User:  *userWithProfile,
	}, nil
}

func (s *AuthService) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	valid, err := utils.VerifyPassword(user.Password, req.Password)
	if err != nil || !valid {
		return nil, errors.New("invalid email or password")
	}

	token, err := utils.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	userWithProfile, err := s.userRepo.GetUserWithProfile(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token: token,
		User:  *userWithProfile,
	}, nil
}

func (s *AuthService) GetProfile(userID int) (*models.UserWithProfile, error) {
	return s.userRepo.GetUserWithProfile(userID)
}

func (s *AuthService) UpdateProfile(userID int, req models.UpdateProfileRequest) error {
	profile, err := s.userRepo.GetProfile(userID)
	if err != nil {
		return err
	}

	profile.FullName = req.FullName
	profile.Phone = req.Phone
	profile.Address = req.Address

	return s.userRepo.UpdateProfile(profile)
}

func (s *AuthService) UpdateProfilePhoto(userID int, photoURL string) error {
	profile, err := s.userRepo.GetProfile(userID)
	if err != nil {
		return err
	}

	if profile.PhotoURL != "" {
		utils.DeleteFile(profile.PhotoURL)
	}

	profile.PhotoURL = photoURL
	return s.userRepo.UpdateProfile(profile)
}

func (s *AuthService) ChangePassword(userID int, req models.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	valid, err := utils.VerifyPassword(user.Password, req.OldPassword)
	if err != nil || !valid {
		return errors.New("invalid old password")
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, hashedPassword)
}
