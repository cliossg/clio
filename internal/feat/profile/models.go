package profile

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID          uuid.UUID
	SiteID      uuid.UUID
	ShortID     string
	Slug        string
	Name        string
	Surname     string
	Bio         string
	SocialLinks string
	PhotoPath   string
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewProfile(siteID uuid.UUID, slug, name, surname, createdBy string) *Profile {
	now := time.Now()
	return &Profile{
		ID:          uuid.New(),
		SiteID:      siteID,
		ShortID:     uuid.New().String()[:8],
		Slug:        slug,
		Name:        name,
		Surname:     surname,
		Bio:         "",
		SocialLinks: "[]",
		PhotoPath:   "",
		CreatedBy:   createdBy,
		UpdatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (p *Profile) FullName() string {
	if p.Surname == "" {
		return p.Name
	}
	return p.Name + " " + p.Surname
}
