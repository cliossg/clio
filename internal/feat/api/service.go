package api

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cliossg/clio/internal/db/sqlc"
	"github.com/cliossg/clio/pkg/cl/config"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

// Service defines the API token management interface.
type Service interface {
	Start(ctx context.Context) error
	CreateToken(ctx context.Context, userID uuid.UUID, name string) (token string, t *APIToken, err error)
	ValidateToken(ctx context.Context, rawToken string) (*APIToken, error)
	ListTokens(ctx context.Context, userID uuid.UUID) ([]*APIToken, error)
	DeleteToken(ctx context.Context, id uuid.UUID) error
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
}

// NewService creates a new API service.
func NewService(dbProvider DBProvider, cfg *config.Config, log logger.Logger) Service {
	return &service{
		dbProvider: dbProvider,
		cfg:        cfg,
		log:        log,
	}
}

func (s *service) ensureQueries() {
	if s.queries == nil && s.dbProvider != nil {
		s.queries = sqlc.New(s.dbProvider.GetDB())
	}
}

func (s *service) Start(ctx context.Context) error {
	s.ensureQueries()
	s.log.Info("API service started")
	return nil
}

func (s *service) CreateToken(ctx context.Context, userID uuid.UUID, name string) (string, *APIToken, error) {
	s.ensureQueries()

	rawToken, token, err := NewAPIToken(userID, name)
	if err != nil {
		return "", nil, err
	}

	_, err = s.queries.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
		ID:        token.ID.String(),
		UserID:    userID.String(),
		Name:      name,
		TokenHash: token.TokenHash,
		CreatedAt: token.CreatedAt,
	})
	if err != nil {
		return "", nil, fmt.Errorf("cannot create API token: %w", err)
	}

	return rawToken, token, nil
}

func (s *service) ValidateToken(ctx context.Context, rawToken string) (*APIToken, error) {
	s.ensureQueries()

	hash := HashToken(rawToken)
	row, err := s.queries.GetAPITokenByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	token := apiTokenFromSQLC(row)

	// Check expiration
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	// Update last_used_at in the background
	now := time.Now()
	_ = s.queries.UpdateAPITokenLastUsed(ctx, sqlc.UpdateAPITokenLastUsedParams{
		LastUsedAt: sql.NullTime{Time: now, Valid: true},
		ID:         token.ID.String(),
	})

	return token, nil
}

func (s *service) ListTokens(ctx context.Context, userID uuid.UUID) ([]*APIToken, error) {
	s.ensureQueries()

	rows, err := s.queries.ListAPITokensByUser(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot list API tokens: %w", err)
	}

	tokens := make([]*APIToken, 0, len(rows))
	for _, row := range rows {
		tokens = append(tokens, apiTokenFromSQLC(row))
	}
	return tokens, nil
}

func (s *service) DeleteToken(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	if err := s.queries.DeleteAPIToken(ctx, id.String()); err != nil {
		return fmt.Errorf("cannot delete API token: %w", err)
	}
	return nil
}
