package repository

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/google/uuid"
)

// UserRepository handles user data persistence in PostgreSQL
type UserRepository struct {
	db *sql.DB
}

var (
	userRepo     *UserRepository
	userRepoOnce sync.Once
)

// GetUserRepository returns singleton instance of UserRepository
func GetUserRepository() *UserRepository {
	userRepoOnce.Do(func() {
		userRepo = &UserRepository{
			db: database.GetDB(),
		}
		
		// Create default admin user if not exists
		// The schema.sql already handles this, but we'll check anyway
		_, err := userRepo.GetUserByEmail("admin@whatsapp.com")
		if err != nil {
			userRepo.CreateUser("admin@whatsapp.com", "Administrator", "changeme123")
		}
	})
	return userRepo
}

// DB returns the database connection
func (r *UserRepository) DB() *sql.DB {
	return r.db
}

// CreateUser creates a new user in the database
func (r *UserRepository) CreateUser(email, fullName, password string) (*models.User, error) {
	// Check if user exists
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}
	
	// Encode password with base64 (for easy viewing)
	encodedPassword := base64.StdEncoding.EncodeToString([]byte(password))
	
	// Create user
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        email,
		FullName:     fullName,
		PasswordHash: encodedPassword,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	// Insert into database
	query := `
		INSERT INTO users (id, email, full_name, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	err = r.db.QueryRow(query, user.ID, user.Email, user.FullName, user.PasswordHash, 
		user.IsActive, user.CreatedAt, user.UpdatedAt).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	fmt.Printf("Debug GetUserByEmail: Looking for email: '%s'\n", email)
	user := &models.User{}
	var lastLogin sql.NullTime
	query := `
		SELECT id, email, full_name, password_hash, is_active, created_at, updated_at, last_login
		FROM users WHERE email = $1
	`
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PasswordHash,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &lastLogin,
	)
	if err == sql.ErrNoRows {
		fmt.Printf("Debug GetUserByEmail: User not found for email: '%s'\n", email)
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		fmt.Printf("Debug GetUserByEmail: Database error: %v\n", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	
	fmt.Printf("Debug GetUserByEmail: Found user - Email: '%s', ID: %s\n", user.Email, user.ID)
	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, full_name, password_hash, is_active, created_at, updated_at, last_login
		FROM users WHERE id = $1
	`
	var lastLogin sql.NullTime
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PasswordHash,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &lastLogin,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	
	return user, nil
}

// ValidatePassword checks if password is correct
func (r *UserRepository) ValidatePassword(email, password string) (*models.User, error) {
	user, err := r.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}
	
	// Debug logging
	fmt.Printf("Debug: Validating password for email: %s\n", email)
	fmt.Printf("Debug: Password provided: %s\n", password)
	fmt.Printf("Debug: Encoded password from DB: %s\n", user.PasswordHash)
	
	// Decode the stored password
	decodedPassword, err := base64.StdEncoding.DecodeString(user.PasswordHash)
	if err != nil {
		fmt.Printf("Debug: Failed to decode password: %v\n", err)
		return nil, fmt.Errorf("invalid password format")
	}
	
	// Compare passwords
	if string(decodedPassword) != password {
		fmt.Printf("Debug: Password mismatch - Stored: '%s', Provided: '%s'\n", string(decodedPassword), password)
		return nil, fmt.Errorf("invalid password")
	}
	
	fmt.Printf("Debug: Password validation successful\n")
	
	// Update last login
	_, err = r.db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = $1", user.ID)
	if err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login: %v\n", err)
	}
	
	return user, nil
}

// CreateSession creates a new user session
func (r *UserRepository) CreateSession(userID string) (*models.UserSession, error) {
	session := &models.UserSession{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     uuid.New().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	
	query := `
		INSERT INTO user_sessions (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(query, session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	return session, nil
}

// GetSession retrieves a session by token
func (r *UserRepository) GetSession(token string) (*models.UserSession, error) {
	session := &models.UserSession{}
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM user_sessions 
		WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP
	`
	err := r.db.QueryRow(query, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	return session, nil
}

// AddUserDevice adds a device for a user
func (r *UserRepository) AddUserDevice(userID, deviceName string) (*models.UserDevice, error) {
	device := &models.UserDevice{
		ID:         uuid.New().String(),
		UserID:     userID,
		DeviceName: deviceName,
		Status:     "offline",
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
	}
	
	query := `
		INSERT INTO user_devices (id, user_id, device_name, status, last_seen, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, last_seen
	`
	err := r.db.QueryRow(query, device.ID, device.UserID, device.DeviceName, 
		device.Status, device.LastSeen, device.CreatedAt).Scan(&device.CreatedAt, &device.LastSeen)
	if err != nil {
		return nil, fmt.Errorf("failed to add device: %w", err)
	}
	
	return device, nil
}

// AddUserDeviceWithPhone adds a device for a user with phone number
func (r *UserRepository) AddUserDeviceWithPhone(userID, deviceName, phone string) (*models.UserDevice, error) {
	device := &models.UserDevice{
		ID:         uuid.New().String(),
		UserID:     userID,
		DeviceName: deviceName,
		Phone:      phone,
		Status:     "offline",
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
	}
	
	query := `
		INSERT INTO user_devices (id, user_id, device_name, phone, status, last_seen, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, last_seen
	`
	err := r.db.QueryRow(query, device.ID, device.UserID, device.DeviceName, 
		device.Phone, device.Status, device.LastSeen, device.CreatedAt).Scan(&device.CreatedAt, &device.LastSeen)
	if err != nil {
		return nil, fmt.Errorf("failed to add device with phone: %w", err)
	}
	
	return device, nil
}

// GetUserDevices gets all devices for a user
func (r *UserRepository) GetUserDevices(userID string) ([]*models.UserDevice, error) {
	query := `
		SELECT id, user_id, device_name, phone, jid, status, last_seen, created_at
		FROM user_devices 
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	defer rows.Close()
	
	var devices []*models.UserDevice
	for rows.Next() {
		device := &models.UserDevice{}
		var phone, jid sql.NullString
		
		err := rows.Scan(
			&device.ID, &device.UserID, &device.DeviceName, 
			&phone, &jid, &device.Status, &device.LastSeen, &device.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		
		if phone.Valid {
			device.Phone = phone.String
		}
		if jid.Valid {
			device.JID = jid.String
		}
		
		devices = append(devices, device)
	}
	
	return devices, nil
}

// GetDeviceByID gets a device by ID
func (r *UserRepository) GetDeviceByID(deviceID string) (*models.UserDevice, error) {
	var device models.UserDevice
	query := `
		SELECT id, user_id, device_name, COALESCE(phone, ''), COALESCE(jid, ''), status, 
		       COALESCE(min_delay_seconds, 5), COALESCE(max_delay_seconds, 15),
		       created_at, COALESCE(updated_at, created_at), last_seen
		FROM user_devices
		WHERE id = $1
	`
	
	err := r.db.QueryRow(query, deviceID).Scan(
		&device.ID, &device.UserID, &device.DeviceName, &device.Phone,
		&device.JID, &device.Status, &device.MinDelaySeconds, &device.MaxDelaySeconds,
		&device.CreatedAt, &device.UpdatedAt, &device.LastSeen,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	
	return &device, nil
}

// UpdateDeviceStatus updates device status
func (r *UserRepository) UpdateDeviceStatus(deviceID, status string, phone, jid string) error {
	query := `
		UPDATE user_devices 
		SET status = $2, last_seen = CURRENT_TIMESTAMP, phone = $3, jid = $4
		WHERE id = $1
	`
	_, err := r.db.Exec(query, deviceID, status, phone, jid)
	if err != nil {
		return fmt.Errorf("failed to update device status: %w", err)
	}
	
	return nil
}

// DeleteDevice deletes a device
func (r *UserRepository) DeleteDevice(deviceID string) error {
	_, err := r.db.Exec("DELETE FROM user_devices WHERE id = $1", deviceID)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}
	
	return nil
}

// GetDevice gets a specific device by ID
func (r *UserRepository) GetDevice(userID, deviceID string) (*models.UserDevice, error) {
	query := `
		SELECT id, user_id, device_name, COALESCE(phone, ''), COALESCE(jid, ''), status, created_at, last_seen
		FROM user_devices
		WHERE user_id = $1 AND id = $2
	`
	
	var device models.UserDevice
	
	err := r.db.QueryRow(query, userID, deviceID).Scan(
		&device.ID, &device.UserID, &device.DeviceName,
		&device.Phone, &device.JID, &device.Status,
		&device.CreatedAt, &device.LastSeen,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	
	return &device, nil
}

// GetAllUsers returns all users (for admin)
func (r *UserRepository) GetAllUsers() ([]*models.User, error) {
	query := `
		SELECT id, email, full_name, is_active, created_at, updated_at, last_login
		FROM users 
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var lastLogin sql.NullTime
		
		err := rows.Scan(
			&user.ID, &user.Email, &user.FullName, 
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &lastLogin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		
		if lastLogin.Valid {
			user.LastLogin = lastLogin.Time
		}
		
		users = append(users, user)
	}
	
	return users, nil
}

// CleanupExpiredSessions removes expired sessions
func (r *UserRepository) CleanupExpiredSessions() error {
	_, err := r.db.Exec("DELETE FROM user_sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

// UpdateDevicePhone updates the phone number for a user's device
func (r *UserRepository) UpdateDevicePhone(userID, deviceID, phone string) error {
	query := `
		UPDATE user_devices 
		SET phone = $1, last_seen = CURRENT_TIMESTAMP
		WHERE user_id = $2 AND id = $3
	`
	result, err := r.db.Exec(query, phone, userID, deviceID)
	if err != nil {
		return fmt.Errorf("failed to update device phone: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("device not found or not owned by user")
	}
	
	return nil
}