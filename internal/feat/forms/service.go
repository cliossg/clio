package forms

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

// Service defines the forms service interface.
type Service interface {
	Start(ctx context.Context) error
	CreateSubmission(ctx context.Context, sub *FormSubmission) error
	GetSubmission(ctx context.Context, id uuid.UUID) (*FormSubmission, error)
	ListSubmissions(ctx context.Context, siteID uuid.UUID) ([]*FormSubmission, error)
	CountUnread(ctx context.Context, siteID uuid.UUID) (int64, error)
	MarkRead(ctx context.Context, id uuid.UUID) error
	DeleteSubmission(ctx context.Context, id uuid.UUID) error
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

// NewService creates a new forms service.
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
	s.log.Info("Forms service started")
	return nil
}

func (s *service) CreateSubmission(ctx context.Context, sub *FormSubmission) error {
	s.ensureQueries()

	_, err := s.queries.CreateFormSubmission(ctx, sqlc.CreateFormSubmissionParams{
		ID:        sub.ID.String(),
		SiteID:    sub.SiteID.String(),
		FormType:  sub.FormType,
		Name:      sql.NullString{String: sub.Name, Valid: sub.Name != ""},
		Email:     sql.NullString{String: sub.Email, Valid: sub.Email != ""},
		Message:   sub.Message,
		IpAddress: sql.NullString{String: sub.IPAddress, Valid: sub.IPAddress != ""},
		UserAgent: sql.NullString{String: sub.UserAgent, Valid: sub.UserAgent != ""},
		CreatedAt: sub.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("cannot create form submission: %w", err)
	}
	return nil
}

func (s *service) GetSubmission(ctx context.Context, id uuid.UUID) (*FormSubmission, error) {
	s.ensureQueries()

	row, err := s.queries.GetFormSubmission(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get form submission: %w", err)
	}
	return formSubmissionFromSQLC(row), nil
}

func (s *service) ListSubmissions(ctx context.Context, siteID uuid.UUID) ([]*FormSubmission, error) {
	s.ensureQueries()

	rows, err := s.queries.ListFormSubmissionsBySite(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot list form submissions: %w", err)
	}

	subs := make([]*FormSubmission, 0, len(rows))
	for _, row := range rows {
		subs = append(subs, formSubmissionFromSQLC(row))
	}
	return subs, nil
}

func (s *service) CountUnread(ctx context.Context, siteID uuid.UUID) (int64, error) {
	s.ensureQueries()

	count, err := s.queries.CountUnreadFormSubmissions(ctx, siteID.String())
	if err != nil {
		return 0, fmt.Errorf("cannot count unread submissions: %w", err)
	}
	return count, nil
}

func (s *service) MarkRead(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	return s.queries.MarkFormSubmissionRead(ctx, sqlc.MarkFormSubmissionReadParams{
		ReadAt: sql.NullTime{Time: time.Now(), Valid: true},
		ID:     id.String(),
	})
}

func (s *service) DeleteSubmission(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	return s.queries.DeleteFormSubmission(ctx, id.String())
}
