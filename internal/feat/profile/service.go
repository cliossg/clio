package profile

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
	ErrProfileNotFound = errors.New("profile not found")
)

type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	CreateProfile(ctx context.Context, siteID uuid.UUID, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*Profile, error)
	GetProfile(ctx context.Context, id uuid.UUID) (*Profile, error)
	GetProfileBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*Profile, error)
	ListProfiles(ctx context.Context, siteID uuid.UUID) ([]*Profile, error)
	UpdateProfile(ctx context.Context, profile *Profile) error
	DeleteProfile(ctx context.Context, id uuid.UUID) error
}

type DBProvider interface {
	GetDB() *sql.DB
}

type service struct {
	dbProvider DBProvider
	queries    *sqlc.Queries
	cfg        *config.Config
	log        logger.Logger
}

func NewService(dbProvider DBProvider, cfg *config.Config, log logger.Logger) Service {
	return &service{
		dbProvider: dbProvider,
		cfg:        cfg,
		log:        log,
	}
}

func (s *service) Start(ctx context.Context) error {
	s.log.Info("Profile service started")
	return nil
}

func (s *service) Stop(ctx context.Context) error {
	s.log.Info("Profile service stopped")
	return nil
}

func (s *service) ensureQueries() {
	if s.queries == nil && s.dbProvider != nil {
		s.queries = sqlc.New(s.dbProvider.GetDB())
	}
}

func (s *service) CreateProfile(ctx context.Context, siteID uuid.UUID, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*Profile, error) {
	s.ensureQueries()

	profile := NewProfile(siteID, slug, name, surname, createdBy)
	profile.Bio = bio
	profile.SocialLinks = socialLinks
	profile.PhotoPath = photoPath

	params := sqlc.CreateProfileParams{
		ID:          profile.ID.String(),
		SiteID:      profile.SiteID.String(),
		ShortID:     profile.ShortID,
		Slug:        profile.Slug,
		Name:        profile.Name,
		Surname:     profile.Surname,
		Bio:         profile.Bio,
		SocialLinks: profile.SocialLinks,
		PhotoPath:   profile.PhotoPath,
		CreatedBy:   profile.CreatedBy,
		UpdatedBy:   profile.UpdatedBy,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
	}

	_, err := s.queries.CreateProfile(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("cannot create profile: %w", err)
	}

	return profile, nil
}

func (s *service) GetProfile(ctx context.Context, id uuid.UUID) (*Profile, error) {
	s.ensureQueries()

	p, err := s.queries.GetProfile(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("cannot get profile: %w", err)
	}

	return fromSQLCProfile(p), nil
}

func (s *service) GetProfileBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*Profile, error) {
	s.ensureQueries()

	p, err := s.queries.GetProfileBySlug(ctx, sqlc.GetProfileBySlugParams{
		SiteID: siteID.String(),
		Slug:   slug,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("cannot get profile by slug: %w", err)
	}

	return fromSQLCProfile(p), nil
}

func (s *service) ListProfiles(ctx context.Context, siteID uuid.UUID) ([]*Profile, error) {
	s.ensureQueries()

	profiles, err := s.queries.ListProfiles(ctx, siteID.String())
	if err != nil {
		return nil, fmt.Errorf("cannot list profiles: %w", err)
	}

	result := make([]*Profile, len(profiles))
	for i, p := range profiles {
		result[i] = fromSQLCProfile(p)
	}

	return result, nil
}

func (s *service) UpdateProfile(ctx context.Context, profile *Profile) error {
	s.ensureQueries()

	profile.UpdatedAt = time.Now()

	params := sqlc.UpdateProfileParams{
		ID:          profile.ID.String(),
		Slug:        profile.Slug,
		Name:        profile.Name,
		Surname:     profile.Surname,
		Bio:         profile.Bio,
		SocialLinks: profile.SocialLinks,
		PhotoPath:   profile.PhotoPath,
		UpdatedBy:   profile.UpdatedBy,
		UpdatedAt:   profile.UpdatedAt,
	}

	_, err := s.queries.UpdateProfile(ctx, params)
	if err != nil {
		return fmt.Errorf("cannot update profile: %w", err)
	}

	return nil
}

func (s *service) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	s.ensureQueries()

	err := s.queries.DeleteProfile(ctx, id.String())
	if err != nil {
		return fmt.Errorf("cannot delete profile: %w", err)
	}

	return nil
}

func fromSQLCProfile(p sqlc.Profile) *Profile {
	id, _ := uuid.Parse(p.ID)
	siteID, _ := uuid.Parse(p.SiteID)
	return &Profile{
		ID:          id,
		SiteID:      siteID,
		ShortID:     p.ShortID,
		Slug:        p.Slug,
		Name:        p.Name,
		Surname:     p.Surname,
		Bio:         p.Bio,
		SocialLinks: p.SocialLinks,
		PhotoPath:   p.PhotoPath,
		CreatedBy:   p.CreatedBy,
		UpdatedBy:   p.UpdatedBy,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
