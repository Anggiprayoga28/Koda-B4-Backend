package repositories

import (
	"coffee-shop/config"
	"coffee-shop/models"
	"context"
	"errors"
	"fmt"
	"time"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (email, password, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	now := time.Now()
	err := config.DB.QueryRow(
		context.Background(),
		query,
		user.Email,
		user.Password,
		user.Role,
		now,
		now,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	return err
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	query := `SELECT id, email, password, role, created_at, updated_at FROM users WHERE email = $1`

	user := &models.User{}
	err := config.DB.QueryRow(context.Background(), query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) FindByID(id int) (*models.User, error) {
	query := `SELECT id, email, password, role, created_at, updated_at FROM users WHERE id = $1`

	user := &models.User{}
	err := config.DB.QueryRow(context.Background(), query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) FindAll(page, limit int) ([]models.UserWithProfile, int, error) {
	offset := (page - 1) * limit

	var totalCount int
	countQuery := `SELECT COUNT(*) FROM users`
	if err := config.DB.QueryRow(context.Background(), countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT 
			u.id, u.email, u.role, u.created_at,
			COALESCE(up.full_name, '') as full_name,
			COALESCE(up.phone, '') as phone,
			COALESCE(up.address, '') as address,
			COALESCE(up.photo_url, '') as photo_url
		FROM users u
		LEFT JOIN user_profiles up ON u.id = up.user_id
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := config.DB.Query(context.Background(), query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []models.UserWithProfile{}
	for rows.Next() {
		var user models.UserWithProfile
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Role,
			&user.CreatedAt,
			&user.FullName,
			&user.Phone,
			&user.Address,
			&user.PhotoURL,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, totalCount, nil
}

func (r *UserRepository) Update(user *models.User) error {
	query := `
		UPDATE users 
		SET email = $1, role = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := config.DB.Exec(
		context.Background(),
		query,
		user.Email,
		user.Role,
		time.Now(),
		user.ID,
	)

	return err
}

func (r *UserRepository) UpdatePassword(userID int, hashedPassword string) error {
	query := `UPDATE users SET password = $1, updated_at = $2 WHERE id = $3`
	_, err := config.DB.Exec(context.Background(), query, hashedPassword, time.Now(), userID)
	return err
}

func (r *UserRepository) Delete(id int) error {
	tx, err := config.DB.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), "DELETE FROM user_profiles WHERE user_id = $1", id)
	if err != nil {
		return err
	}

	result, err := tx.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return tx.Commit(context.Background())
}

func (r *UserRepository) CreateProfile(profile *models.UserProfile) error {
	query := `
		INSERT INTO user_profiles (user_id, full_name, phone, address, photo_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	now := time.Now()
	err := config.DB.QueryRow(
		context.Background(),
		query,
		profile.UserID,
		profile.FullName,
		profile.Phone,
		profile.Address,
		profile.PhotoURL,
		now,
		now,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)

	return err
}

func (r *UserRepository) GetProfile(userID int) (*models.UserProfile, error) {
	query := `
		SELECT id, user_id, full_name, phone, address, photo_url, created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`

	profile := &models.UserProfile{}
	err := config.DB.QueryRow(context.Background(), query, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FullName,
		&profile.Phone,
		&profile.Address,
		&profile.PhotoURL,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *UserRepository) UpdateProfile(profile *models.UserProfile) error {
	query := `
		UPDATE user_profiles 
		SET full_name = $1, phone = $2, address = $3, photo_url = $4, updated_at = $5
		WHERE user_id = $6
	`

	result, err := config.DB.Exec(
		context.Background(),
		query,
		profile.FullName,
		profile.Phone,
		profile.Address,
		profile.PhotoURL,
		time.Now(),
		profile.UserID,
	)

	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("profile not found for user_id: %d", profile.UserID)
	}

	return nil
}

func (r *UserRepository) GetUserWithProfile(userID int) (*models.UserWithProfile, error) {
	query := `
		SELECT 
			u.id, u.email, u.role, u.created_at,
			COALESCE(up.full_name, '') as full_name,
			COALESCE(up.phone, '') as phone,
			COALESCE(up.address, '') as address,
			COALESCE(up.photo_url, '') as photo_url
		FROM users u
		LEFT JOIN user_profiles up ON u.id = up.user_id
		WHERE u.id = $1
	`

	user := &models.UserWithProfile{}
	err := config.DB.QueryRow(context.Background(), query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.FullName,
		&user.Phone,
		&user.Address,
		&user.PhotoURL,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}
