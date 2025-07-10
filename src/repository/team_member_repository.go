package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
)

type TeamMemberRepository struct {
	db *sql.DB
}

func NewTeamMemberRepository(db *sql.DB) *TeamMemberRepository {
	return &TeamMemberRepository{db: db}
}

// Create creates a new team member
func (r *TeamMemberRepository) Create(ctx context.Context, member *models.TeamMember) error {
	query := `
		INSERT INTO team_members (username, password, created_by, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	
	err := r.db.QueryRowContext(ctx, query,
		member.Username,
		member.Password,
		member.CreatedBy,
		member.IsActive,
	).Scan(&member.ID, &member.CreatedAt, &member.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create team member: %w", err)
	}
	
	return nil
}

// GetByID retrieves a team member by ID
func (r *TeamMemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TeamMember, error) {
	query := `
		SELECT id, username, password, created_by, created_at, updated_at, is_active
		FROM team_members
		WHERE id = $1
	`
	
	member := &models.TeamMember{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&member.ID,
		&member.Username,
		&member.Password,
		&member.CreatedBy,
		&member.CreatedAt,
		&member.UpdatedAt,
		&member.IsActive,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team member: %w", err)
	}
	
	return member, nil
}

// GetByUsername retrieves a team member by username
func (r *TeamMemberRepository) GetByUsername(ctx context.Context, username string) (*models.TeamMember, error) {
	query := `
		SELECT id, username, password, created_by, created_at, updated_at, is_active
		FROM team_members
		WHERE username = $1
	`
	
	member := &models.TeamMember{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&member.ID,
		&member.Username,
		&member.Password,
		&member.CreatedBy,
		&member.CreatedAt,
		&member.UpdatedAt,
		&member.IsActive,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team member by username: %w", err)
	}
	
	return member, nil
}

// GetAll retrieves all team members
func (r *TeamMemberRepository) GetAll(ctx context.Context) ([]models.TeamMember, error) {
	query := `
		SELECT id, username, password, created_by, created_at, updated_at, is_active
		FROM team_members
		ORDER BY username
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	defer rows.Close()
	
	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		err := rows.Scan(
			&member.ID,
			&member.Username,
			&member.Password,
			&member.CreatedBy,
			&member.CreatedAt,
			&member.UpdatedAt,
			&member.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		members = append(members, member)
	}
	
	return members, nil
}

// Update updates a team member
func (r *TeamMemberRepository) Update(ctx context.Context, member *models.TeamMember) error {
	query := `
		UPDATE team_members
		SET username = $2, password = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	
	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.Username,
		member.Password,
		member.IsActive,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update team member: %w", err)
	}
	
	return nil
}

// Delete deletes a team member
func (r *TeamMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM team_members WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete team member: %w", err)
	}
	
	return nil
}

// CreateSession creates a new session for a team member
func (r *TeamMemberRepository) CreateSession(ctx context.Context, memberID uuid.UUID) (*models.TeamSession, error) {
	session := &models.TeamSession{
		ID:           uuid.New(),
		TeamMemberID: memberID,
		Token:        uuid.New().String(),
		ExpiresAt:    time.Now().Add(24 * time.Hour), // 24 hour session
		CreatedAt:    time.Now(),
	}
	
	query := `
		INSERT INTO team_sessions (id, team_member_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		session.ID,
		session.TeamMemberID,
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create team session: %w", err)
	}
	
	return session, nil
}

// GetSessionByToken retrieves a session by token
func (r *TeamMemberRepository) GetSessionByToken(ctx context.Context, token string) (*models.TeamSession, error) {
	query := `
		SELECT id, team_member_id, token, expires_at, created_at
		FROM team_sessions
		WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP
	`
	
	session := &models.TeamSession{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.TeamMemberID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	return session, nil
}

// DeleteSession deletes a session
func (r *TeamMemberRepository) DeleteSession(ctx context.Context, token string) error {
	query := `DELETE FROM team_sessions WHERE token = $1`
	
	_, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	return nil
}

// GetAllWithDeviceCount retrieves all team members with their device counts
func (r *TeamMemberRepository) GetAllWithDeviceCount(ctx context.Context) ([]models.TeamMemberWithDevices, error) {
	query := `
		SELECT 
			tm.id, tm.username, tm.password, tm.created_by, 
			tm.created_at, tm.updated_at, tm.is_active,
			COUNT(DISTINCT ud.id) as device_count
		FROM team_members tm
		LEFT JOIN user_devices ud ON ud.device_name = tm.username
		GROUP BY tm.id, tm.username, tm.password, tm.created_by, 
				 tm.created_at, tm.updated_at, tm.is_active
		ORDER BY tm.username
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members with device count: %w", err)
	}
	defer rows.Close()
	
	var members []models.TeamMemberWithDevices
	for rows.Next() {
		var member models.TeamMemberWithDevices
		err := rows.Scan(
			&member.ID,
			&member.Username,
			&member.Password,
			&member.CreatedBy,
			&member.CreatedAt,
			&member.UpdatedAt,
			&member.IsActive,
			&member.DeviceCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team member with devices: %w", err)
		}
		
		// Get device IDs for this member
		deviceIDs, err := r.GetDeviceIDsForMember(ctx, member.Username)
		if err != nil {
			return nil, err
		}
		member.DeviceIDs = deviceIDs
		
		members = append(members, member)
	}
	
	return members, nil
}

// GetDeviceIDsForMember gets all device IDs for a team member based on username
func (r *TeamMemberRepository) GetDeviceIDsForMember(ctx context.Context, username string) ([]string, error) {
	query := `
		SELECT id FROM user_devices WHERE device_name = $1
	`
	
	rows, err := r.db.QueryContext(ctx, query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get device IDs: %w", err)
	}
	defer rows.Close()
	
	var deviceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan device ID: %w", err)
		}
		deviceIDs = append(deviceIDs, id)
	}
	
	return deviceIDs, nil
}
