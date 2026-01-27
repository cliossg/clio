package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/cliossg/clio/internal/db/sqlc"
	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActive      = errors.New("user is not active")
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
)

// Service defines the auth service interface.
type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Authenticate(ctx context.Context, email, password string) (*User, error)
	CreateUser(ctx context.Context, email, password, name string, mustChangePassword bool) (*User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)
	UpdateUser(ctx context.Context, user *User) error
	CreateSession(ctx context.Context, userID uuid.UUID) (*Session, error)
	ValidateSession(ctx context.Context, sessionID string) (string, error)
	DeleteSession(ctx context.Context, sessionID string) error
	GetSessionTTL() time.Duration
}

// DBProvider provides access to the database.
type DBProvider interface {
	GetDB() *sql.DB
}

type service struct {
	dbProvider DBProvider
	queries    *sqlc.Queries
	cfg        *config.Config
	log        logger.Logger
	sessionTTL time.Duration
}

// NewService creates a new auth service.
func NewService(dbProvider DBProvider, cfg *config.Config, log logger.Logger) Service {
	return &service{
		dbProvider: dbProvider,
		cfg:        cfg,
		log:        log,
	}
}

func (s *service) Start(ctx context.Context) error {
	ttl, err := time.ParseDuration(s.cfg.Auth.SessionTTL)
	if err != nil {
		ttl = 24 * time.Hour
		s.log.Infof("Invalid session TTL, using default: %v", ttl)
	}
	s.sessionTTL = ttl
	s.log.Info("Auth service started")
	return nil
}

func (s *service) Stop(ctx context.Context) error {
	s.log.Info("Auth service stopped")
	return nil
}

func (s *service) ensureQueries() {
	if s.queries == nil && s.dbProvider != nil {
		s.queries = sqlc.New(s.dbProvider.GetDB())
	}
}

func (s *service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	s.ensureQueries()

	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.CheckPassword(password) {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, ErrUserNotActive
	}

	return user, nil
}

func (s *service) CreateUser(ctx context.Context, email, password, name string, mustChangePassword bool) (*User, error) {
	s.ensureQueries()

	user, err := NewUser(email, password, name)
	if err != nil {
		return nil, fmt.Errorf("cannot create user: %w", err)
	}
	user.MustChangePassword = mustChangePassword

	params := sqlc.CreateUserParams{
		ID:                 user.ID.String(),
		ShortID:            user.ShortID,
		Email:              user.Email,
		PasswordHash:       user.PasswordHash,
		Name:               user.Name,
		Status:             user.Status,
		MustChangePassword: boolToInt(user.MustChangePassword),
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}

	_, err = s.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("cannot create user in database: %w", err)
	}

	return user, nil
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	s.ensureQueries()

	sqlcUser, err := s.queries.GetUser(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	return fromSQLCUser(sqlcUser), nil
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	s.ensureQueries()

	sqlcUser, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("cannot get user by email: %w", err)
	}

	return fromSQLCUser(sqlcUser), nil
}

func (s *service) ListUsers(ctx context.Context) ([]*User, error) {
	s.ensureQueries()

	sqlcUsers, err := s.queries.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot list users: %w", err)
	}

	users := make([]*User, len(sqlcUsers))
	for i, sqlcUser := range sqlcUsers {
		users[i] = fromSQLCUser(sqlcUser)
	}

	return users, nil
}

func (s *service) UpdateUser(ctx context.Context, user *User) error {
	s.ensureQueries()

	params := sqlc.UpdateUserParams{
		ID:                 user.ID.String(),
		Email:              user.Email,
		PasswordHash:       user.PasswordHash,
		Name:               user.Name,
		Status:             user.Status,
		MustChangePassword: boolToInt(user.MustChangePassword),
		UpdatedAt:          user.UpdatedAt,
	}

	_, err := s.queries.UpdateUser(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update user: %w", err)
	}

	return nil
}

func (s *service) CreateSession(ctx context.Context, userID uuid.UUID) (*Session, error) {
	s.ensureQueries()

	session := NewSession(userID, s.sessionTTL)

	params := sqlc.CreateSessionParams{
		ID:        session.ID,
		UserID:    userID.String(),
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
	}

	_, err := s.queries.CreateSession(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("cannot create session: %w", err)
	}

	return session, nil
}

func (s *service) ValidateSession(ctx context.Context, sessionID string) (string, error) {
	s.ensureQueries()

	sqlcSession, err := s.queries.GetValidSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSessionNotFound
		}
		return "", fmt.Errorf("cannot get session: %w", err)
	}

	return sqlcSession.UserID, nil
}

func (s *service) DeleteSession(ctx context.Context, sessionID string) error {
	s.ensureQueries()

	err := s.queries.DeleteSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("cannot delete session: %w", err)
	}

	return nil
}

func (s *service) GetSessionTTL() time.Duration {
	return s.sessionTTL
}

// fromSQLCUser converts a sqlc User to our domain User.
func fromSQLCUser(u sqlc.User) *User {
	id, _ := uuid.Parse(u.ID)
	return &User{
		ID:                 id,
		ShortID:            u.ShortID,
		Email:              u.Email,
		PasswordHash:       u.PasswordHash,
		Name:               u.Name,
		Status:             u.Status,
		MustChangePassword: u.MustChangePassword != 0,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
